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
