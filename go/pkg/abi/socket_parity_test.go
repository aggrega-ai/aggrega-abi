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
