// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

package abi_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi/go/pkg/abi"
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

// readGoldenHex reads a canonical-CBOR golden and returns its hex body, holding
// the file to an exact on-disk shape: the lowercase hex digits, then exactly one
// terminating newline, and nothing else anywhere.
//
// The strictness is the point, and it is asserted rather than absorbed. A reader
// that trims surrounding whitespace cannot distinguish the committed file from
// one carrying a stray second newline, a carriage return, or leading indentation
// — so a fixture whose bytes drifted would still satisfy every downstream
// assertion, and the byte-level guarantee this whole vector corpus exists to
// prove would be quietly weaker than it reads. Whitespace tolerance also has to
// be identical in all three languages or the corpus is not one golden: a form one
// harness silently accepts and another rejects makes the same file mean two
// different things. Both are avoided by admitting exactly one form.
//
// The JSON goldens need no equivalent helper. Their committed bytes ARE the wire
// bytes, carrying no terminating newline, so the byte-compare against the encoder
// output is already the strictest possible check on the file's shape and a
// separate assertion would only restate it.
func readGoldenHex(t *testing.T, name string) string {
	t.Helper()

	raw := string(readGolden(t, name))
	body, ok := strings.CutSuffix(raw, "\n")
	if !ok {
		t.Fatalf("golden %s does not end in a newline, so its committed shape is not the canonical one: %q",
			name, raw)
	}
	for i := 0; i < len(body); i++ {
		if c := body[i]; (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("golden %s carries %q at offset %d, so it is not lowercase hex followed by one newline: %q",
				name, string(c), i, raw)
		}
	}
	return body
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
// Value, whose omission rule the host and the codec must agree on, a side
// effect, and a diagnostic, which every codec carries as the shared diagnostic's
// own structural encoding — so the byte-compare pins the any-value emission and
// the diagnostic's encoding.
func TestGoldenParity_FRPStepOutput(t *testing.T) {
	t.Parallel()
	roundTripBothGolden[abi.FRPStepOutput](t, "FRPStepOutput")
}
