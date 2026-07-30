// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
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
// The contracts proven per payload are the four the shared parity helper
// asserts, over both wire forms — a strict round-trip against the host's wire
// exercising the decode leg (token-walk into the typed value), the encode leg
// (manual field-ordered marshal honouring omit semantics), and the same pair for
// the canonical CBOR bytes.

// readGolden reads a shared vector document from the one testdata tree the Go,
// Rust, and TypeScript harnesses all validate against. The tree lives at the
// aggrega-abi root, above this Go module, reached by relative path (os.ReadFile,
// not embed, so the cross-directory read needs no module membership).
func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", name))
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
	roundTripBothGolden[abi.FRPStepInput](t, "FRPStepInput")
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
	roundTripBothGolden[abi.FRPStepOutput](t, "FRPStepOutput")
}
