// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// The mutator and linter socket goldens under testdata/ are the canonical wire
// bytes — the same goldens the Rust and TypeScript codecs validate against, so a
// byte-match here is a cross-language parity assertion: the three codecs are
// generated from one schema and produce one wire. Each golden carries the
// determinism-sensitive cases — sorted map keys, verbatim opaque carry, omitted
// empty fields — so a codec regression breaks the byte-compare loudly.
//
// The contract proven per payload: unmarshal the golden through the generated
// codec, re-marshal, and assert byte-identical output.

func roundTripGolden[T any](t *testing.T, name string) {
	t.Helper()
	golden := readGolden(t, name)
	var v T
	if err := json.Unmarshal(golden, &v); err != nil {
		t.Fatalf("codec unmarshal %s: %v", name, err)
	}
	got, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("codec re-marshal %s: %v", name, err)
	}
	if !bytes.Equal(golden, got) {
		t.Errorf("%s wire not byte-identical to golden:\n golden: %s\n    got: %s", name, golden, got)
	}
}

func TestSocketParity_CLIRunInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.CLIRunInput](t, "CLIRunInput.golden.json")
}

func TestSocketParity_MutatorInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.MutatorInput](t, "MutatorInput.golden.json")
}

func TestSocketParity_MutatorOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.MutatorOutput](t, "MutatorOutput.golden.json")
}

func TestSocketParity_LinterInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinterInput](t, "LinterInput.golden.json")
}

func TestSocketParity_LinterOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinterOutput](t, "LinterOutput.golden.json")
}

func TestSocketParity_IRGraph(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.IRGraph](t, "IRGraph.golden.json")
}

func TestSocketParity_PolicyInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.PolicyInput](t, "PolicyInput.golden.json")
}

func TestSocketParity_PolicyOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.PolicyOutput](t, "PolicyOutput.golden.json")
}

func TestSocketParity_RenderNodeInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.RenderNodeInput](t, "RenderNodeInput.golden.json")
}

func TestSocketParity_RenderNodeOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.RenderNodeOutput](t, "RenderNodeOutput.golden.json")
}

func TestSocketParity_AssembleBoundaryInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.AssembleBoundaryInput](t, "AssembleBoundaryInput.golden.json")
}

func TestSocketParity_AssembleBoundaryOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.AssembleBoundaryOutput](t, "AssembleBoundaryOutput.golden.json")
}

func TestSocketParity_LinkPublishInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinkPublishInput](t, "LinkPublishInput.golden.json")
}

func TestSocketParity_LinkPublishOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinkPublishOutput](t, "LinkPublishOutput.golden.json")
}

func TestSocketParity_LinkResolveInput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinkResolveInput](t, "LinkResolveInput.golden.json")
}

func TestSocketParity_LinkResolveOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.LinkResolveOutput](t, "LinkResolveOutput.golden.json")
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
	roundTripGolden[abi.PluginDescriptor](t, "PluginDescriptorCLI.golden.json")
}

func TestSocketParity_PluginDescriptorProvider(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.PluginDescriptor](t, "PluginDescriptorProvider.golden.json")
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
	roundTripGolden[abi.PluginDescriptor](t, "PluginDescriptorMinimal.golden.json")
}

func TestSocketParity_PluginDescriptorMoo(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.PluginDescriptor](t, "PluginDescriptorMoo.golden.json")
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
	roundTripGolden[abi.ParseInput](t, "ParseInput.golden.json")
}

func TestSocketParity_ParseOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.ParseOutput](t, "ParseOutput.golden.json")
}

func TestSocketParity_ParseOutputBare(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.ParseOutput](t, "ParseOutputBare.golden.json")
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
	roundTripGolden[abi.ImportInput](t, "ImportInput.golden.json")
}

func TestSocketParity_ImportOutput(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.ImportOutput](t, "ImportOutput.golden.json")
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
	roundTripGolden[abi.ParseSourceResponse](t, "ParseSourceResponse.golden.json")
}

func TestSocketParity_FetchExternalInputResponse(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponse.golden.json")
}

// The content golden is the success path: a base64 content string carried
// through the pointer-to-scalar content field, proving the field round-trips
// across the three codecs (a zero-length payload stays distinguishable from a
// failure because the content key is still present).
func TestSocketParity_FetchExternalInputResponseContent(t *testing.T) {
	t.Parallel()
	roundTripGolden[abi.FetchExternalInputResponse](t, "FetchExternalInputResponseContent.golden.json")
}
