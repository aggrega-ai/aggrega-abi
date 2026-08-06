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
// cases — sorted map keys, verbatim opaque carry, omitted empty fields — so a
// codec regression breaks the byte-compare loudly.
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
// other and stay green. And it is the only one that can observe a map-key sort
// being lost from the JSON encoder: the two wire forms order map keys by
// different rules — JSON lexicographically, CBOR by the encoded key bytes, whose
// length head leads — so a value arriving from the CBOR wire presents its keys in
// an order the JSON wire must actively re-sort. A value decoded from the JSON
// golden instead arrives already in JSON order, which is why the first contract
// stays green whether the encoder sorts or not.
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
// map is empty (pinning that all three codecs emit an empty object rather than
// dropping the member), a binding-key list whose entries are NOT in sorted
// order (pinning that a list is carried in author order rather than sorted),
// and an opaque settings object with out-of-order keys (pinning the verbatim
// carry). The output omits the failure member, pinning that omission.
//
// The annotation map's key sort is pinned in both wire forms, but by two
// different contracts, and the difference is worth knowing before trusting
// either. The CBOR half falls out of the golden round-trip itself, because CBOR
// orders map keys by their encoded bytes, whose length head leads, so a short key
// precedes a long one — an order that differs from the lexicographic order the
// golden's JSON carries, leaving the encoder no choice but to re-sort. The JSON
// half is invisible to that round-trip, whose value arrives from the JSON golden
// already in JSON order; it is pinned instead by the helper's fourth contract,
// which sources the value from the committed CBOR bytes and so presents the keys
// in the other order. TestAnnotationTargetMapKeyOrder below pins the same rule a
// third way, from a constructed value, and states both orders literally.
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

// TestAnnotationTargetMapKeyOrder pins the annotation map's key ordering in both
// wire forms by ENCODING a constructed value, the one path on which a lost JSON
// sort is observable: a value decoded from a golden already carries the golden's
// lexicographic order, so the round-trip assertions above stay green whether the
// JSON encoder sorts or not. Map iteration here is unordered, so without the
// sort this assertion fails on most runs rather than on every run — which is
// itself a reason to assert the order against a fixed expectation instead of
// relying on the round-trip.
//
// The two wire forms order the same three keys differently, and the test asserts
// both. JSON sorts lexicographically. CBOR sorts by the encoded key bytes, whose
// length head leads, so a shorter key always precedes a longer one and the three
// keys emit in the exact reverse of their JSON order. A codec that applied the
// JSON rule to the CBOR wire would satisfy neither assertion.
func TestAnnotationTargetMapKeyOrder(t *testing.T) {
	t.Parallel()

	v := abi.AnnotationTarget{
		BoundaryPath:  "prod/edge",
		ComponentName: "web",
		Annotations: map[string]string{
			"tier":             "frontend",
			"aggrega.ai/owner": "checkout",
			"cost-center":      "42",
		},
	}

	const wantJSON = `{"boundary_path":"prod/edge","component_name":"web",` +
		`"annotations":{"aggrega.ai/owner":"checkout","cost-center":"42","tier":"frontend"}}`
	gotJSON, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("codec marshal AnnotationTarget: %v", err)
	}
	if string(gotJSON) != wantJSON {
		t.Errorf("JSON annotation keys not in lexicographic order:\n want: %s\n  got: %s", wantJSON, gotJSON)
	}

	enc, err := v.EncodeCBOR()
	if err != nil {
		t.Fatalf("codec encode CBOR AnnotationTarget: %v", err)
	}
	// Shortest key first: "tier" (4), "cost-center" (11), "aggrega.ai/owner" (16).
	ordered := []string{"tier", "cost-center", "aggrega.ai/owner"}
	at := make([]int, len(ordered))
	for i, k := range ordered {
		if at[i] = bytes.Index(enc, []byte(k)); at[i] < 0 {
			t.Fatalf("CBOR wire does not carry annotation key %q: %s", k, hex.EncodeToString(enc))
		}
	}
	for i := 1; i < len(at); i++ {
		if at[i-1] >= at[i] {
			t.Errorf("CBOR annotation keys not in canonical length-first order (%q@%d before %q@%d): %s",
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
// boundary assembly, artifact delimiters, source filter, the assembly-name
// annotation, and the cross-boundary-linking claim) plus an opaque settings
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
// free-form error message plus the machine-matchable error_code. A byte-match
// proves the code field crosses the wire byte-identically across the three
// codecs, so a guest matches on the code without parsing the message.

func TestSocketParity_ParseSourceResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.ParseSourceResponse](t, "ParseSourceResponse")
}

func TestSocketParity_FetchExternalInputResponse(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponse")
}

// The content golden is the success path: a base64 content string carried
// through the pointer-to-scalar content field, proving the field round-trips
// across the three codecs (a zero-length payload stays distinguishable from a
// failure because the content key is still present).
func TestSocketParity_FetchExternalInputResponseContent(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponseContent")
}
