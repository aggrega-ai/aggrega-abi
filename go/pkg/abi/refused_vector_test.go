// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

package abi_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi/go/pkg/abi"
)

// A refused vector is canonical CBOR that every decoder of its shape must REFUSE:
// a document no encoder of the shape can write, malformed in exactly one way. It
// is the other half of what the vector pairs prove. A pair holds the decoders to
// what they must read; a refused vector holds them to what they must not, and the
// two cannot stand in for each other — a decoder that keeps one of two repeated
// keys reads every well-formed vector correctly.
//
// The order-preserving map's array form is what these pin. A map entry is a
// two-element array of a text key and a value, and a key appears once. A decoder
// that met a repeated key and kept either value would be choosing silently which
// of the two the sender meant, so it refuses instead, naming the key; an entry of
// any other shape is refused as well. Every language's decoder is held to the
// same vectors, and the compiler's independent byte-parity oracle confirms, with a
// decoder this schema did not generate, that each vector is well-formed CBOR
// whose only fault is the one its name states.

// refusedSuffix is the file a refused vector is. It deliberately does not end in
// the vector pair's CBOR suffix, so no census of pairs counts it as half of one.
const refusedSuffix = ".refused-cbor.hex"

// refuseCBORGolden asserts that both decode entry points of a generated type
// refuse the refused vector at <name>.refused-cbor.hex with an error carrying
// want, and that neither panics. The vector's on-disk shape is held to the hex
// golden shape, so a refused vector cannot be read differently from a pair.
func refuseCBORGolden[T any, PT interface {
	*T
	DecodeCBOR([]byte) error
	DecodeCBORStrict([]byte) error
}](t *testing.T, name, want string) {
	t.Helper()

	data, err := hex.DecodeString(readGoldenHex(t, name+refusedSuffix))
	if err != nil {
		t.Fatalf("decode refused vector hex %s: %v", name, err)
	}
	for _, entry := range []struct {
		name   string
		decode func(PT, []byte) error
	}{
		{"DecodeCBOR", func(p PT, b []byte) error { return p.DecodeCBOR(b) }},
		{"DecodeCBORStrict", func(p PT, b []byte) error { return p.DecodeCBORStrict(b) }},
	} {
		var v T
		err := decodeWithoutPanic(func() error { return entry.decode(PT(&v), data) })
		if err == nil {
			t.Errorf("%s: %s accepted a vector every decoder must refuse, decoding it to %+v", name, entry.name, v)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %s refused it, but the error does not name the fault: got %q, want it to contain %q",
				name, entry.name, err, want)
		}
	}
}

// decodeWithoutPanic runs one decode and reports a panic as an error, so a
// decoder that crashes on malformed input fails the assertion that it refuses it
// rather than passing it.
func decodeWithoutPanic(decode func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("decode panicked rather than refusing: %v", r)
		}
	}()
	return decode()
}

// TestRefusedVectors holds the order-preserving map's decoders to the vectors
// they must refuse: a repeated key, named in the refusal; an entry holding three
// items; and an entry whose key is not a text string.
func TestRefusedVectors(t *testing.T) {
	t.Parallel()

	t.Run("RepeatedKey", func(t *testing.T) {
		t.Parallel()
		refuseCBORGolden[abi.MutatorNode](t, "MapOrderMutatorNodeRepeatedKey", `duplicate key "a"`)
	})
	t.Run("EntryArity", func(t *testing.T) {
		t.Parallel()
		refuseCBORGolden[abi.MutatorNode](t, "MapOrderMutatorNodeEntryArity", "holds 3 items, want 2")
	})
	t.Run("EntryKeyNotText", func(t *testing.T) {
		t.Parallel()
		refuseCBORGolden[abi.MutatorNode](t, "MapOrderMutatorNodeEntryKeyNotText", "expected text string")
	})
}
