// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// The golden-fixture parity guards prove the generated reflect-free codec
// produces and consumes the exact JSON wire bytes the host's encoding/json
// produces for a populated payload. The goldens under testdata/ were captured
// once from the host hand types' encoding/json output; this module carries no
// dependency on the host packages, so the golden bytes — not a live host
// marshal — are the parity reference. A codec change that drifts the wire (an
// omit-logic regression, a field reorder, a base64 discrepancy) breaks the
// byte-compare loudly.
//
// The contract proven per payload: unmarshal the golden through the generated
// codec, re-marshal, and assert the result is byte-identical to the golden.
// This is a strict round-trip against the host's wire, exercising both the
// decode leg (token-walk into the typed value) and the encode leg (manual
// field-ordered marshal honouring omit semantics).

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return b
}

// TestGoldenParity_FRPStepInput round-trips the host's FRPStepInput wire
// through the generated codec and asserts byte-identical output. The golden
// carries the opaque context and step_input objects (which the codec carries
// as verbatim json.RawMessage), an own set with a structured unit and an
// opaque base64-bytes unit, a host unit, and a preceding step — so the decode
// descends every composite container and the opaque passthroughs are
// preserved.
func TestGoldenParity_FRPStepInput(t *testing.T) {
	t.Parallel()
	golden := readGolden(t, "frpstepinput.golden.json")

	var in abi.FRPStepInput
	if err := json.Unmarshal(golden, &in); err != nil {
		t.Fatalf("codec unmarshal FRPStepInput: %v", err)
	}
	got, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("codec re-marshal FRPStepInput: %v", err)
	}
	if !bytes.Equal(golden, got) {
		t.Errorf("FRPStepInput wire not byte-identical to host golden:\n golden: %s\n    got: %s", golden, got)
	}
}

// TestGoldenParity_FRPStepOutput round-trips the host's FRPStepOutput wire
// through the generated codec. The golden carries a patch with the any-typed
// Value (the field whose omit logic surfaced a host/codec discrepancy in the
// proving-ground), a side effect, and the opaque diagnostics array (a
// json.RawMessage carry-through in the codec, a typed slice on the host) — so
// the byte-compare pins the any-value emission and the opaque-array
// passthrough.
func TestGoldenParity_FRPStepOutput(t *testing.T) {
	t.Parallel()
	golden := readGolden(t, "frpstepoutput.golden.json")

	var out abi.FRPStepOutput
	if err := json.Unmarshal(golden, &out); err != nil {
		t.Fatalf("codec unmarshal FRPStepOutput: %v", err)
	}
	got, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("codec re-marshal FRPStepOutput: %v", err)
	}
	if !bytes.Equal(golden, got) {
		t.Errorf("FRPStepOutput wire not byte-identical to host golden:\n golden: %s\n    got: %s", golden, got)
	}
}
