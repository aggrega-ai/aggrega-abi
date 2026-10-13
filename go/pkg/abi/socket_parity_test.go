// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// The socket goldens under testdata/ are the canonical wire bytes — the same
// goldens the Rust and TypeScript codecs validate against, so a byte-match here
// is a cross-language parity assertion: the three codecs are generated from one
// schema and produce one wire. Each golden carries the determinism-sensitive
// cases — a plain map's sorted keys, an order-preserving map's own order,
// verbatim opaque carry, omitted empty fields — so a codec regression breaks the
// byte-compare loudly.
//
// Every payload runs through one helper covering both wire forms. A vector that
// exercised only JSON would leave the CBOR codec — a separate set of generated
// methods — entirely unpinned.

// cborEncoder is the encode half of a generated type's CBOR codec, declared on
// the value receiver. The decode half is declared on the pointer, so the
// helper below binds the two through a paired type parameter.
type cborEncoder interface {
	EncodeCBOR() ([]byte, error)
}

// roundTripBothGolden asserts a payload against BOTH committed wire forms: the
// JSON document at <name>.golden.json and the canonical CBOR bytes at
// <name>.cbor.hex. The pair matters because a type's JSON and CBOR codecs are
// separate generated methods — a vector that exercises only one leaves the
// other unpinned, so a CBOR-only regression crosses the wire unnoticed while
// the JSON assertion stays green.
//
// Four contracts are proven per vector. The JSON wire round-trips
// byte-identical. The CBOR encoder reproduces the committed canonical bytes.
// Decoding those bytes back re-encodes to the same bytes. And the COMMITTED CBOR
// bytes decode into a value whose JSON encoding is the committed JSON golden.
//
// That fourth contract carries two assertions the other three structurally
// cannot make. It is the only one that feeds the CBOR decoder the committed
// canonical bytes — the other CBOR legs decode this codec's own fresh output, so
// an encoder and decoder that shared a compensating defect would agree with each
// other and stay green. And it is the only one that can observe a plain map's
// key sort being lost from the JSON encoder: the two wire forms order a plain
// map's keys by different rules — JSON lexicographically, CBOR by the encoded key
// bytes, whose length head leads — so a value arriving from the CBOR wire
// presents its keys in an order the JSON wire must actively re-sort. A value
// decoded from the JSON golden instead arrives already in JSON order, which is
// why the first contract stays green whether the encoder sorts or not. An
// order-preserving map carries one order on both wires, the one it holds, so
// for it the same contract proves the order survives the trip through the
// binary form.
//
// Both halves of the codec are constrained rather than asserted at run time, so
// naming a type that carries no CBOR codec fails to compile at the call site
// instead of panicking inside the helper.
func roundTripBothGolden[T cborEncoder, PT interface {
	*T
	DecodeCBOR([]byte) error
}](t *testing.T, name string) {
	t.Helper()

	wantJSON := readGolden(t, name+".golden.json")
	var v T
	if err := json.Unmarshal(wantJSON, &v); err != nil {
		t.Fatalf("codec unmarshal %s: %v", name, err)
	}
	gotJSON, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("codec re-marshal %s: %v", name, err)
	}
	if !bytes.Equal(wantJSON, gotJSON) {
		t.Errorf("%s JSON wire not byte-identical to golden:\n golden: %s\n    got: %s", name, wantJSON, gotJSON)
	}

	wantHex := readGoldenHex(t, name+".cbor.hex")
	enc, err := v.EncodeCBOR()
	if err != nil {
		t.Fatalf("codec encode CBOR %s: %v", name, err)
	}
	if gotHex := hex.EncodeToString(enc); gotHex != wantHex {
		t.Errorf("%s CBOR wire not byte-identical to golden:\n golden: %s\n    got: %s", name, wantHex, gotHex)
	}

	var back T
	if err := PT(&back).DecodeCBOR(enc); err != nil {
		t.Fatalf("codec decode CBOR %s: %v", name, err)
	}
	reenc, err := back.EncodeCBOR()
	if err != nil {
		t.Fatalf("codec re-encode CBOR %s: %v", name, err)
	}
	if !bytes.Equal(enc, reenc) {
		t.Errorf("%s CBOR round-trip not stable:\n first: %s\n  then: %s",
			name, hex.EncodeToString(enc), hex.EncodeToString(reenc))
	}

	wantCBOR, err := hex.DecodeString(wantHex)
	if err != nil {
		t.Fatalf("decode committed CBOR hex %s: %v", name, err)
	}
	var fromCBOR T
	if err := PT(&fromCBOR).DecodeCBOR(wantCBOR); err != nil {
		t.Fatalf("codec decode committed CBOR %s: %v", name, err)
	}
	crossJSON, err := json.Marshal(fromCBOR)
	if err != nil {
		t.Fatalf("codec marshal JSON from committed CBOR %s: %v", name, err)
	}
	if !bytes.Equal(wantJSON, crossJSON) {
		t.Errorf("%s JSON wire encoded from the committed CBOR bytes is not the JSON golden:\n golden: %s\n    got: %s",
			name, wantJSON, crossJSON)
	}
}

func TestSocketParity_CLIRunInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.CLIRunInput](t, "CLIRunInput")
}

func TestSocketParity_MutatorInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.MutatorInput](t, "MutatorInput")
}

func TestSocketParity_MutatorOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.MutatorOutput](t, "MutatorOutput")
}

func TestSocketParity_LinterInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinterInput](t, "LinterInput")
}

func TestSocketParity_LinterOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinterOutput](t, "LinterOutput")
}

// TestSocketParity_LinterInputRelease pins the input of a rule that reads a whole
// Release, with both of its fields POPULATED: the Release and the Component map.
// Both are omitted when empty, so the vector above carries the same bytes whether
// a codec handles them or drops them; this one does not. The Release and each
// Component cross as opaque documents with their keys out of key order, so a codec
// that re-serialised one instead of carrying it would not match. The Component map
// is order-preserving and holds "agent" before "db", which is not the length-first
// order a sorted binary map would take, so a codec that sorted it on the binary
// wire fails the byte-compare.
func TestSocketParity_LinterInputRelease(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinterInput](t, "LinterInputRelease")
}

func TestSocketParity_IRGraph(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRGraph](t, "IRGraph")
}

func TestSocketParity_PolicyInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PolicyInput](t, "PolicyInput")
}

func TestSocketParity_PolicyOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PolicyOutput](t, "PolicyOutput")
}

// The governance goldens below cover the payload shapes reached only by a
// governance attachment, in both wire forms. RenderPolicyInput is the payload a
// policy attachment receives at the render stage — rendered artifacts instead
// of the IR graph — and carries one artifact with both optional members (the
// parsed tree and the declared format) and one that omits both, so the
// byte-compare pins the omission as well as the verbatim tree carry.
//
// The two annotation-transform goldens are the wire an annotation-transform
// attachment receives and returns. The input carries a target whose annotation
// map is empty (pinning that all three codecs emit an empty container rather
// than dropping the member), a binding-key list whose entries are NOT in sorted
// order (pinning that a list is carried in author order rather than sorted),
// and an opaque settings object with out-of-order keys (pinning the verbatim
// carry). The output omits the failure member, pinning that omission.
//
// The annotation map is order-preserving, so neither wire form sorts it: both
// carry the order the target holds. A golden round-trip shows that only where the
// golden's order is neither sorted order, which is what the map-order probe vector
// is for. TestAnnotationTargetKeepsTheAnnotationOrder below pins it a third way,
// from a constructed value, and states the order literally.
//
// MutatorInputSources is the label-transform payload WITH its pinned sources
// and settings populated. The pre-existing mutator golden leaves both absent,
// so the file list a governance attachment hands its guest crossed the wire
// with no parity coverage in any of the three codecs.

func TestSocketParity_RenderPolicyInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderPolicyInput](t, "RenderPolicyInput")
}

func TestSocketParity_AnnotationTransformInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.AnnotationTransformInput](t, "AnnotationTransformInput")
}

func TestSocketParity_AnnotationTransformOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.AnnotationTransformOutput](t, "AnnotationTransformOutput")
}

func TestSocketParity_MutatorInputSources(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.MutatorInput](t, "MutatorInputSources")
}

// TestAnnotationTargetKeepsTheAnnotationOrder pins the annotation map's order in
// both wire forms by ENCODING a constructed value whose order is neither of the
// two sorted orders: not the lexicographic one a JSON map sort would give
// ("8080", "aggrega.ai/owner", "cost-center", "tier"), not the length-first one a
// CBOR map sort would give ("8080", "tier", "cost-center", "aggrega.ai/owner"),
// and with a key that is an array index, which a plain JavaScript object would
// move to the front. An encoder applying any of those orders fails here.
//
// The map is order-preserving, so both wire forms carry the order it holds —
// the JSON form as an object written in that order, the binary form as an array
// of entries in that order — and the test asserts each, literally.
func TestAnnotationTargetKeepsTheAnnotationOrder(t *testing.T) {
	t.Parallel()

	ordered := []string{"tier", "aggrega.ai/owner", "8080", "cost-center"}
	values := map[string]string{"tier": "frontend", "aggrega.ai/owner": "checkout", "8080": "port", "cost-center": "42"}
	var annotations abi.OrderedMap[string]
	for _, k := range ordered {
		annotations.Set(k, values[k])
	}
	v := abi.AnnotationTarget{BoundaryPath: "prod/edge", ComponentName: "web", Annotations: annotations}

	const wantJSON = `{"boundary_path":"prod/edge","component_name":"web",` +
		`"annotations":{"tier":"frontend","aggrega.ai/owner":"checkout","8080":"port","cost-center":"42"}}`
	gotJSON, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("codec marshal AnnotationTarget: %v", err)
	}
	if string(gotJSON) != wantJSON {
		t.Errorf("JSON annotation keys not in the order the map holds:\n want: %s\n  got: %s", wantJSON, gotJSON)
	}

	enc, err := v.EncodeCBOR()
	if err != nil {
		t.Fatalf("codec encode CBOR AnnotationTarget: %v", err)
	}
	at := make([]int, len(ordered))
	for i, k := range ordered {
		if at[i] = bytes.Index(enc, []byte(k)); at[i] < 0 {
			t.Fatalf("CBOR wire does not carry annotation key %q: %s", k, hex.EncodeToString(enc))
		}
	}
	for i := 1; i < len(at); i++ {
		if at[i-1] >= at[i] {
			t.Errorf("CBOR annotation entries not in the order the map holds (%q@%d before %q@%d): %s",
				ordered[i], at[i], ordered[i-1], at[i-1], hex.EncodeToString(enc))
			break
		}
	}
}

func TestSocketParity_RenderNodeInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeInput](t, "RenderNodeInput")
}

func TestSocketParity_RenderNodeOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeOutput](t, "RenderNodeOutput")
}

// TestSocketParity_RenderNodeOutputRenderFacts pins a render output whose fact
// list is POPULATED. The field is omitted when empty, so every other vector on this
// socket carries the same bytes whether the codec handles it or drops it; this one
// does not. Every fact's value is a list, as the socket requires. The facts are
// listed out of name order and one list's entry is an object whose keys are out of
// key order, so a codec that sorted the list, or re-serialised a value instead of
// carrying its bytes, fails the byte-compare.
func TestSocketParity_RenderNodeOutputRenderFacts(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeOutput](t, "RenderNodeOutputRenderFacts")
}

// TestSocketParity_RenderNodeInputOmits pins a source render input whose omit
// entries are POPULATED. The list is omitted when empty, so every other vector on
// this socket carries the same bytes whether the codec handles it or drops it; this
// one does not. One entry carries a namespace and the other does not, which pins
// the optional member's omission inside an entry, and the two are listed out of
// kind order and out of name order, so a codec that sorted the list fails the
// byte-compare.
func TestSocketParity_RenderNodeInputOmits(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeInput](t, "RenderNodeInputOmits")
}

// TestSocketParity_RenderNodeInputOmitsSensitive pins an omit entry's sensitivity
// mark. The mark is omitted when false, so every other vector carrying omit entries
// has the same bytes whether the codec handles it or drops it; this one does not.
// The marked entry comes first and the unmarked one after it, so a codec that
// dropped a true mark, wrote a false one, or sorted the list fails the byte-compare.
func TestSocketParity_RenderNodeInputOmitsSensitive(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeInput](t, "RenderNodeInputOmitsSensitive")
}

// TestSocketParity_RenderNodeOutputOmitted pins a render output whose per-entry
// counts are POPULATED, for the same reason as the input above. The counts are a
// non-zero one followed by a zero: a codec that dropped a zero count, or sorted
// the list, fails the byte-compare.
func TestSocketParity_RenderNodeOutputOmitted(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.RenderNodeOutput](t, "RenderNodeOutputOmitted")
}

// TestSocketParity_IRNodeProjectionSensitiveRenderedKeys pins a node projection
// whose per-key sensitivity marking is POPULATED. The field is omitted from the
// payload when it is empty, so every other vector on this socket carries the same
// bytes whether the codec handles it or drops it: a regression there moves nothing
// and fails nowhere. This vector is the one that does not have that property, and
// it is a captured render payload rather than an authored fixture — the projection
// the host built for a real node of a worked example whose Component declares a
// credential slot as handling secret material.
func TestSocketParity_IRNodeProjectionSensitiveRenderedKeys(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRNodeProjection](t, "IRNodeProjectionSensitiveRenderedKeys")
}

// TestSocketParity_IRFeatureEntrySensitivePayloadLeaves pins an attachment entry
// whose per-leaf sensitivity marking is POPULATED, for the same reason the
// projection vector above exists: the field is omitted from the payload when it
// is empty, so every other vector carrying an attachment entry has the same
// bytes whether the codec handles the field or drops it, and a regression there
// moves nothing and fails nowhere.
//
// It is a captured payload rather than an authored fixture — the entry the host
// built for an attachment whose Feature declares one of its values leaves
// sensitive and whose deployment carries a literal at that leaf. The entry also
// carries the two other sensitivity units populated, so the vector shows the three
// coexisting at their three different units rather than one standing in for
// another.
func TestSocketParity_IRFeatureEntrySensitivePayloadLeaves(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRFeatureEntry](t, "IRFeatureEntrySensitivePayloadLeaves")
}

// TestSocketParity_IRSpecConnectionAttachment pins an attachment entry that
// carries the connection it was written on, beside one of the node's own that
// carries none. The field is omitted when absent, so every other vector carrying
// an attachment entry has the same bytes whether the codec handles it or drops
// it: this vector is the one where dropping it moves the bytes, and the entry
// beside it is the one where emitting it unconditionally would.
//
// The two sit in one node's attachment bucket, which is why the vector is the
// graph's spec rather than a single entry, and the edge the connection names is
// carried with them. The connection's own keys also discriminate the two wire
// forms' orders — socket before alias in the JSON, alias before socket in the
// canonical CBOR, whose length head leads.
func TestSocketParity_IRSpecConnectionAttachment(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRSpec](t, "IRSpecConnectionAttachment")
}

// TestSocketParity_IRSpecAttachmentRealization pins the realization an
// attachment entry carries — the FeatureRecipe the Manifest binds to it — on both
// shapes that carry one: a node's attachment entry and a Release boundary's. The
// field is omitted when absent, so every other vector carrying either shape has
// the same bytes whether a codec handles it or drops it; this is the vector where
// dropping it moves the bytes.
//
// One node carries three entries: one of its own block, one written on a
// connection, each naming a different realization, and one naming none, which
// is the entry where emitting the field unconditionally would move the bytes. A
// second node and the boundary carry the same boundary-level attachment, as a
// graph carries it in both places.
func TestSocketParity_IRSpecAttachmentRealization(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRSpec](t, "IRSpecAttachmentRealization")
}

// TestSocketParity_IRNodeSensitivePayloadLeaves pins a node whose OWN per-leaf
// marking is populated — the leaves a component's values and infra payload
// carries that the component's own schema declares sensitive, which is a
// different subject from the attachment marking above and rides a different
// part of the graph.
//
// It exists for the same reason its two siblings do: the marking is omitted
// when it names nothing, so every other vector carrying a node has the same
// bytes whether a codec handles the field or drops it, and a regression there
// moves nothing and fails nowhere. The captured node also carries a coordinate
// from each pillar, values pillar first, so a codec that sorted the sequence
// would be visible here rather than only in a build.
func TestSocketParity_IRNodeSensitivePayloadLeaves(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRNode](t, "IRNodeSensitivePayloadLeaves")
}

// TestSocketParity_IRBoundaryFeatureSensitivePayloadLeaves pins the third
// unit: a boundary-level attachment whose infra carries leaves the attached
// Feature declares sensitive. A boundary attachment has no values pillar, so
// every coordinate here is an infra one — which is the shape the payload
// itself forces rather than a narrowing this vector chose.
func TestSocketParity_IRBoundaryFeatureSensitivePayloadLeaves(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.IRBoundaryFeature](t, "IRBoundaryFeatureSensitivePayloadLeaves")
}

func TestSocketParity_AssembleBoundaryInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.AssembleBoundaryInput](t, "AssembleBoundaryInput")
}

func TestSocketParity_AssembleBoundaryOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.AssembleBoundaryOutput](t, "AssembleBoundaryOutput")
}

func TestSocketParity_LinkPublishInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinkPublishInput](t, "LinkPublishInput")
}

func TestSocketParity_LinkPublishOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinkPublishOutput](t, "LinkPublishOutput")
}

func TestSocketParity_LinkResolveInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinkResolveInput](t, "LinkResolveInput")
}

func TestSocketParity_LinkResolveOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.LinkResolveOutput](t, "LinkResolveOutput")
}

// The two describe goldens are both the PluginDescriptor wire — the shape every
// plugin returns from describe(). One exercises a CLI descriptor with commands
// and per-command flags (the surface unique to CLI plugins), the other a
// provider descriptor carrying a full render-behavior contract (input kinds,
// boundary assembly, artifact delimiters, the assembly-name annotation, and the
// cross-boundary-linking claim) plus an opaque settings
// schema — so the byte-compare pins both the command/flag surface and the whole
// contract block across the three codecs.

func TestSocketParity_PluginDescriptorCLI(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PluginDescriptor](t, "PluginDescriptorCLI")
}

func TestSocketParity_PluginDescriptorProvider(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PluginDescriptor](t, "PluginDescriptorProvider")
}

// TestSocketParity_PluginDescriptorProviderOmits pins a provider contract that
// claims to honour omit entries. The claim is omitted when false, so every other
// descriptor vector carries the same bytes whether the codec handles it or drops
// it; this one does not. It sits beside the reuse claim, so the JSON wire pins its
// declared position after that claim while the canonical CBOR, whose length head
// leads, places it first.
func TestSocketParity_PluginDescriptorProviderOmits(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PluginDescriptor](t, "PluginDescriptorProviderOmits")
}

// PluginDescriptorMinimal is a descriptor that declares none of the optional
// nested objects — no render-behavior contract, no authorship metadata, and so
// (transitively) no artifact-delimiter pair and no assembly-name annotation.
// The byte-compare pins that all three codecs omit every absent optional object
// identically rather than emitting an empty placeholder. PluginDescriptorMoo is
// the wire a real CLI plugin (moo) produces: no contract, but metadata present —
// the case the codec must reproduce without inventing a contract key.

func TestSocketParity_PluginDescriptorMinimal(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PluginDescriptor](t, "PluginDescriptorMinimal")
}

func TestSocketParity_PluginDescriptorMoo(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.PluginDescriptor](t, "PluginDescriptorMoo")
}

// The parse goldens are the parser socket wire — a ParseInput a parser guest
// receives and the ParseOutput it returns. ParseInput carries base64 source
// bytes and an opaque options document; ParseOutput carries the opaque tree
// plus diagnostics and node spans. ParseOutputBare is an output that omits
// every optional (no diagnostics, no spans) and carries a top-level-array tree,
// so the byte-compare pins that the codec omits absent optionals and carries a
// non-object tree verbatim.

func TestSocketParity_ParseInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseInput](t, "ParseInput")
}

// TestSocketParity_ParseInputNonUTF8Source pins the source bytes as a raw byte
// string. They are not valid UTF-8 — a bare 0xff 0xfe pair, a NUL, and a lead
// byte followed by a non-continuation byte — so a codec that carried them as text,
// or as base64 text, cannot reproduce the canonical bytes. It omits the options
// document, which the vector above carries, so it is also where that member's
// omission is pinned.
func TestSocketParity_ParseInputNonUTF8Source(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseInput](t, "ParseInputNonUTF8Source")
}

func TestSocketParity_ParseOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseOutput](t, "ParseOutput")
}

func TestSocketParity_ParseOutputBare(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseOutput](t, "ParseOutputBare")
}

// The import goldens are the importer socket wire — an ImportInput an importer
// guest receives and the ImportOutput it returns. ImportInput carries base64
// primary bytes, named multi-file sources, and an opaque settings document;
// ImportOutput carries a reference-mode entity and a draft-mode entity, a
// mapping decision with a full per-signal candidate breakdown, a diagnostic,
// and a scaffold file. The byte-compare pins the float-signal formatting, the
// omitted optional fields, and the verbatim opaque settings carry across the
// three codecs.

func TestSocketParity_ImportInput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ImportInput](t, "ImportInput")
}

// TestSocketParity_ImportInputCatalog pins the catalog view an importer is handed:
// the field is omitted when absent, so this is the vector where dropping it moves
// the bytes. One Component declares sockets and one declares none, the sockets are
// listed out of name order, and the label keys sort differently in the two wire
// forms, so a codec that sorted a list or lost a map-key sort fails it.
func TestSocketParity_ImportInputCatalog(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ImportInput](t, "ImportInputCatalog")
}

func TestSocketParity_ImportOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ImportOutput](t, "ImportOutput")
}

// TestImportOutputUnknownKind proves the emitted-entity kind is an open string:
// an ImportOutput whose entity kind is a value the current emission surface does
// not produce decodes without error and preserves the kind verbatim, so a
// consumer built against today's kinds tolerates a kind a later importer emits.
func TestImportOutputUnknownKind(t *testing.T) {
	t.Parallel()
	const wire = `{"entities":[{"kind":"Feature","name":"tls","mode":"draft","body":"kind: Feature\n"}]}`
	var out abi.ImportOutput
	if err := json.Unmarshal([]byte(wire), &out); err != nil {
		t.Fatalf("decode ImportOutput with unrecognized entity kind: %v", err)
	}
	if len(out.Entities) != 1 {
		t.Fatalf("got %d entities; want 1", len(out.Entities))
	}
	if got := out.Entities[0].Kind; got != "Feature" {
		t.Errorf("entity kind = %q; want the unrecognized value carried verbatim as %q", got, "Feature")
	}
}

// The two host-function reply goldens below carry a failure envelope: the
// free-form error message plus the machine-matchable hostfunc_error_code. A
// byte-match proves the code field crosses the wire byte-identically across the
// three codecs, so a guest matches on the code without parsing the message. The
// key names the closed host-function catalogue rather than failure in general: a
// socket output travelling the other way reports a guest-raised condition under
// condition_code and condition_short_code, from a different register.

func TestSocketParity_ParseSourceResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseSourceResponse](t, "ParseSourceResponse")
}

// The tree golden is the parse-source reply's success path, and it is the half
// the failure envelope above cannot stand in for: the two are the two directions
// of one omission property, and an encoder emitting an error key unconditionally
// passes any test built only from success fixtures while one dropping it
// unconditionally passes any test built only from failure fixtures. It also
// carries the opaque tree, which crosses as a byte string of the JSON document's
// verbatim bytes; the failure envelope reaches no opaque field at all, so the
// tree member's bytes are pinned here. The document is long enough that its
// byte-string length needs a two-byte head, so the length is carried in the head
// rather than in the initial byte and a head that disagreed with the body could
// not hide behind a constant.
func TestSocketParity_ParseSourceResponseTree(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseSourceResponse](t, "ParseSourceResponseTree")
}

func TestSocketParity_FetchExternalInputResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponse")
}

// The content golden is the success path: the fetched bytes carried through a
// byte field, which crosses the binary wire as a raw byte string and the
// textual one as base64, proving the field round-trips across the three
// codecs.
func TestSocketParity_FetchExternalInputResponseContent(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponseContent")
}

// The empty-content golden is a zero-length payload: the content field is
// omitted when empty, so the reply carries no field at all. It stays
// distinguishable from a failure because a failure always carries its error
// fields, which is why this shape and the failure golden above are asserted
// together: a codec that emitted the empty content, or dropped the failure
// fields, moves one of the two.
func TestSocketParity_FetchExternalInputResponseEmptyContent(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponseEmptyContent")
}

// The pack-source and read-source-file goldens below are the request and the
// SUCCESS reply of each. The success reply is the half a failure envelope cannot
// stand in for: the two are the two directions of one omission property, and an
// encoder that emits an error key unconditionally passes any test built only from
// success fixtures while one that drops it unconditionally passes any test built
// only from failure fixtures. Each type's failure envelope lives beside the other
// error-member goldens; together they are the assertion.
//
// The request goldens declare no omittable member, so their map head is a
// constant and there is no omission direction to prove. They pin the head and key
// order instead, on the payload the guest builds and the host decodes.

func TestSocketParity_SourcePackRequest(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.SourcePackRequest](t, "SourcePackRequest")
}

// The packed reply carries three files in the host's byte-order walk — Chart.yaml
// before charts/ before templates/, which is byte order and not case-insensitive
// order — and one file's content is deliberately not valid UTF-8, so the file
// bytes are proven to cross as an opaque byte string rather than as text. Its
// base64 is also the only place in the corpus that exercises both non-alphanumeric
// alphabet characters, which the hand-written base64 codecs in the other two
// languages reach on no other golden.
func TestSocketParity_SourcePackResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.SourcePackResponse](t, "SourcePackResponse")
}

func TestSocketParity_SourceFileRequest(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.SourceFileRequest](t, "SourceFileRequest")
}

func TestSocketParity_SourceFileResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.SourceFileResponse](t, "SourceFileResponse")
}
