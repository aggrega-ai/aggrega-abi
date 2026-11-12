// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

package abi_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// TestOrderPreservingFieldIsOmittedWhenEmptyAndEmittedWhenPopulated holds a
// non-required order-preserving field to its omission on both wires, from both
// directions at once. Each fixture hides the other's failure: an encoder emitting
// the key unconditionally passes the populated one, and one dropping it
// unconditionally passes the empty one. The container is a struct value, which has
// no absent state of its own, so the codec omits it only by asking the container
// whether it is empty. The binary form's map head declares its pair count, so each
// form's bytes are also read back, which a head disagreeing with its body fails.
func TestOrderPreservingFieldIsOmittedWhenEmptyAndEmittedWhenPopulated(t *testing.T) {
	t.Parallel()

	var empty abi.RenderNodeOutput
	assertDerivedKeys(t, "empty", empty, false)

	var populated abi.RenderNodeOutput
	populated.DerivedKeys.Set("resource-names", []string{"web"})
	assertDerivedKeys(t, "populated", populated, true)

	jb, err := json.Marshal(populated)
	if err != nil {
		t.Fatalf("marshal the populated form: %v", err)
	}
	if want := `"derived_keys":{"resource-names":["web"]}`; !strings.Contains(string(jb), want) {
		t.Errorf("the populated field is not written as its object: %s, want it to contain %s", jb, want)
	}
}

func assertDerivedKeys(t *testing.T, form string, v abi.RenderNodeOutput, want bool) {
	t.Helper()
	jb, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal the %s form: %v", form, err)
	}
	if got := bytes.Contains(jb, []byte(`"derived_keys"`)); got != want {
		t.Errorf("%s form: the JSON wire carries derived_keys = %v, want %v: %s", form, got, want, jb)
	}
	cb, err := v.EncodeCBOR()
	if err != nil {
		t.Fatalf("encode the %s form: %v", form, err)
	}
	if got := bytes.Contains(cb, []byte("derived_keys")); got != want {
		t.Errorf("%s form: the binary wire carries derived_keys = %v, want %v: %s", form, got, want, hex.EncodeToString(cb))
	}
	var back abi.RenderNodeOutput
	if err := back.DecodeCBOR(cb); err != nil {
		t.Fatalf("%s form: its own bytes do not read back: %v", form, err)
	}
	if back.DerivedKeys.Len() != v.DerivedKeys.Len() {
		t.Errorf("%s form: %d derived-key spaces read back, want %d", form, back.DerivedKeys.Len(), v.DerivedKeys.Len())
	}
}
