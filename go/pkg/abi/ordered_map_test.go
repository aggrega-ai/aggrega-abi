// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

package abi_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// The order-keeping container is asserted over an authored order that no sorting
// rule reproduces: it is neither lexicographic, nor length-first, nor the order a
// JavaScript object gives an array-index key such as "8080". A container that
// ordered its keys by any of those rules therefore fails on the first key rather
// than agreeing by coincidence.
var authoredProbeKeys = []string{"d", "8080", "bbbb", "aaaaaaaa", "cc"}

const authoredProbeObject = `{"d":"1","8080":"2","bbbb":"3","aaaaaaaa":"4","cc":"5"}`

// probeMap decodes the probe object through the container's own decoder.
func probeMap(t *testing.T) abi.OrderedMap[string] {
	t.Helper()
	var m abi.OrderedMap[string]
	if err := m.UnmarshalJSON([]byte(authoredProbeObject)); err != nil {
		t.Fatalf("decode the probe object: %v", err)
	}
	return m
}

func marshalString(t *testing.T, v json.Marshaler) string {
	t.Helper()
	b, err := v.MarshalJSON()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return string(b)
}

func TestOrderedMap_DecodeThenEncodeKeepsTheAuthoredOrderByteForByte(t *testing.T) {
	m := probeMap(t)
	if got := m.Keys(); !slices.Equal(got, authoredProbeKeys) {
		t.Fatalf("decoded key order = %v, want %v", got, authoredProbeKeys)
	}
	if m.Len() != len(authoredProbeKeys) || m.IsZero() {
		t.Fatalf("Len = %d, IsZero = %t for a map of %d entries", m.Len(), m.IsZero(), len(authoredProbeKeys))
	}
	for i, k := range authoredProbeKeys {
		v, ok := m.Get(k)
		if !ok || v != string(rune('1'+i)) {
			t.Errorf("Get(%q) = %q, %t; want %q, true", k, v, ok, string(rune('1'+i)))
		}
	}
	if got := marshalString(t, m); got != authoredProbeObject {
		t.Errorf("re-encoded = %s\nwant        %s", got, authoredProbeObject)
	}
}

// holder carries the container the way a struct field does, read and written by
// encoding/json, so the container's methods are reached through the standard
// library's dispatch rather than by direct call.
type holder struct {
	Name   string                 `json:"name"`
	Labels abi.OrderedMap[string] `json:"labels,omitzero"`
}

func TestOrderedMap_ThroughEncodingJSONInAStructField(t *testing.T) {
	doc := `{"name":"n","labels":` + authoredProbeObject + `}`
	var h holder
	if err := json.Unmarshal([]byte(doc), &h); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got := h.Labels.Keys(); !slices.Equal(got, authoredProbeKeys) {
		t.Fatalf("decoded key order = %v, want %v", got, authoredProbeKeys)
	}
	out, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(out) != doc {
		t.Errorf("re-encoded = %s\nwant        %s", out, doc)
	}

	// omitzero consults IsZero, so the empty map is left out and the populated
	// one is not: both arms on one field.
	empty, err := json.Marshal(holder{Name: "n"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(empty) != `{"name":"n"}` {
		t.Errorf("an empty map under omitzero encoded as %s, want it omitted", empty)
	}
}

func TestOrderedMap_EmptyAndZero(t *testing.T) {
	var zero abi.OrderedMap[string]
	if !zero.IsZero() || zero.Len() != 0 {
		t.Fatalf("zero value: IsZero = %t, Len = %d", zero.IsZero(), zero.Len())
	}
	if keys := zero.Keys(); keys == nil || len(keys) != 0 {
		t.Errorf("zero value Keys() = %#v, want an empty non-nil slice", keys)
	}
	if _, ok := zero.Get("d"); ok {
		t.Error("zero value Get reported a key present")
	}
	for k := range zero.All() {
		t.Errorf("zero value All yielded %q", k)
	}
	if zero.Delete("d") {
		t.Error("zero value Delete reported a key removed")
	}
	if got := marshalString(t, zero); got != "{}" {
		t.Errorf("zero value encoded as %s, want {}", got)
	}

	for _, doc := range []string{"{}", " { } ", "null"} {
		m := probeMap(t)
		if err := m.UnmarshalJSON([]byte(doc)); err != nil {
			t.Fatalf("decode %q: %v", doc, err)
		}
		if !m.IsZero() {
			t.Errorf("decoding %q into a populated map left %v; the decode replaces what the map held", doc, m.Keys())
		}
		if got := marshalString(t, m); got != "{}" {
			t.Errorf("%q re-encoded as %s, want {}", doc, got)
		}
	}
}

func TestOrderedMap_RefusesARepeatedKeyNamingIt(t *testing.T) {
	var m abi.OrderedMap[string]
	err := m.UnmarshalJSON([]byte(`{"a":"1","b":"2","a":"3"}`))
	if err == nil || !strings.Contains(err.Error(), `duplicate key "a"`) {
		t.Fatalf("decode of a repeated key: err = %v, want a refusal naming \"a\"", err)
	}

	var h holder
	err = json.Unmarshal([]byte(`{"name":"n","labels":{"8080":"1","8080":"2"}}`), &h)
	if err == nil || !strings.Contains(err.Error(), `duplicate key "8080"`) {
		t.Fatalf("json.Unmarshal of a repeated key: err = %v, want a refusal naming \"8080\"", err)
	}
}

func TestOrderedMap_RefusesANonObject(t *testing.T) {
	for _, doc := range []string{`[]`, `"d"`, `1`, `{"a":1}`} {
		var m abi.OrderedMap[string]
		if err := m.UnmarshalJSON([]byte(doc)); err == nil {
			t.Errorf("decode %s into an OrderedMap[string] succeeded, want a refusal", doc)
		}
	}
}

func TestOrderedMap_SetOfAPresentKeyReplacesTheValueAndKeepsItsPosition(t *testing.T) {
	m := probeMap(t)
	m.Set("8080", "x")
	if got := m.Keys(); !slices.Equal(got, authoredProbeKeys) {
		t.Fatalf("key order after replacing 8080 = %v, want %v", got, authoredProbeKeys)
	}
	if v, _ := m.Get("8080"); v != "x" || m.Len() != len(authoredProbeKeys) {
		t.Fatalf("after replacing 8080: Get = %q, Len = %d", v, m.Len())
	}
	m.Set("0", "6")
	want := `{"d":"1","8080":"x","bbbb":"3","aaaaaaaa":"4","cc":"5","0":"6"}`
	if got := marshalString(t, m); got != want {
		t.Errorf("encoded = %s\nwant      %s", got, want)
	}
}

func TestOrderedMap_DeleteClosesTheGapAndKeepsEveryLookup(t *testing.T) {
	m := probeMap(t)
	if !m.Delete("8080") {
		t.Fatal("Delete of a present key reported it absent")
	}
	if m.Delete("8080") {
		t.Fatal("a second Delete of the same key reported it present")
	}
	want := []string{"d", "bbbb", "aaaaaaaa", "cc"}
	if got := m.Keys(); !slices.Equal(got, want) {
		t.Fatalf("key order after Delete = %v, want %v", got, want)
	}
	for k, v := range map[string]string{"d": "1", "bbbb": "3", "aaaaaaaa": "4", "cc": "5"} {
		if got, ok := m.Get(k); !ok || got != v {
			t.Errorf("Get(%q) = %q, %t after Delete; want %q, true", k, got, ok, v)
		}
	}
	m.Set("8080", "2")
	if got := m.Keys(); !slices.Equal(got, append(want, "8080")) {
		t.Errorf("a deleted key set again goes last: got %v", got)
	}
}

func TestOrderedMap_AllRunsInOrderStopsEarlyAndSkipsADeletedEntry(t *testing.T) {
	m := probeMap(t)
	var seen []string
	for k, v := range m.All() {
		seen = append(seen, k+"="+v)
	}
	if want := []string{"d=1", "8080=2", "bbbb=3", "aaaaaaaa=4", "cc=5"}; !slices.Equal(seen, want) {
		t.Fatalf("All yielded %v, want %v", seen, want)
	}

	seen = nil
	for k := range m.All() {
		seen = append(seen, k)
		if len(seen) == 2 {
			break
		}
	}
	if !slices.Equal(seen, authoredProbeKeys[:2]) {
		t.Errorf("All after an early break yielded %v, want %v", seen, authoredProbeKeys[:2])
	}

	// Deleting an entry not yet reached means it is not produced, as a range over
	// a built-in map behaves; the order of the rest is untouched.
	seen = nil
	for k := range m.All() {
		seen = append(seen, k)
		if k == "8080" {
			m.Delete("bbbb")
		}
	}
	if want := []string{"d", "8080", "aaaaaaaa", "cc"}; !slices.Equal(seen, want) {
		t.Errorf("All with a deletion ahead of it yielded %v, want %v", seen, want)
	}
}

// A copy shares the entries the way a copy of a built-in map does. A copy that
// shared only part of them would leave one side with a key order that no longer
// matched its values — a Delete through the copy would make the original encode a
// key twice — so the sharing is asserted through every reader.
func TestOrderedMap_ACopySharesTheEntries(t *testing.T) {
	original := probeMap(t)
	copied := original
	copied.Set("new", "6")
	copied.Delete("d")
	want := []string{"8080", "bbbb", "aaaaaaaa", "cc", "new"}
	for name, m := range map[string]abi.OrderedMap[string]{"original": original, "copy": copied} {
		if got := m.Keys(); !slices.Equal(got, want) {
			t.Errorf("%s Keys() = %v, want %v", name, got, want)
		}
		if v, ok := m.Get("new"); !ok || v != "6" {
			t.Errorf("%s Get(new) = %q, %t", name, v, ok)
		}
		if m.Len() != len(want) {
			t.Errorf("%s Len() = %d, want %d", name, m.Len(), len(want))
		}
		wantJSON := `{"8080":"2","bbbb":"3","aaaaaaaa":"4","cc":"5","new":"6"}`
		if got := marshalString(t, m); got != wantJSON {
			t.Errorf("%s encoded = %s\nwant %s", name, got, wantJSON)
		}
	}

	var zero abi.OrderedMap[string]
	fromZero := zero
	fromZero.Set("k", "v")
	if !zero.IsZero() {
		t.Errorf("setting a copy of the zero value changed the zero value: %v", zero.Keys())
	}
}

// Each value shape the container is read and written through: a scalar from its
// one token, a type with its own JSON codec through that codec, and any other type
// through encoding/json. The documents are written in a non-sorted key order so a
// sorting arm anywhere fails them.
func TestOrderedMap_EveryValueShapeRoundTripsInOrder(t *testing.T) {
	roundTrip := func(t *testing.T, doc string, m interface {
		json.Marshaler
		json.Unmarshaler
	}) {
		t.Helper()
		if err := m.UnmarshalJSON([]byte(doc)); err != nil {
			t.Fatalf("decode %s: %v", doc, err)
		}
		if got := marshalString(t, m); got != doc {
			t.Errorf("re-encoded = %s\nwant        %s", got, doc)
		}
	}
	t.Run("bool", func(t *testing.T) {
		roundTrip(t, `{"z":true,"a":false}`, &abi.OrderedMap[bool]{})
	})
	t.Run("int", func(t *testing.T) {
		roundTrip(t, `{"z":3,"a":-1,"m":0}`, &abi.OrderedMap[int]{})
	})
	t.Run("int64", func(t *testing.T) {
		roundTrip(t, `{"z":9007199254740991,"a":-2}`, &abi.OrderedMap[int64]{})
	})
	t.Run("float64", func(t *testing.T) {
		roundTrip(t, `{"z":1.5,"a":1e+21,"m":0.000001}`, &abi.OrderedMap[float64]{})
	})
	t.Run("raw JSON carried verbatim", func(t *testing.T) {
		roundTrip(t, `{"z":{"y":1,"x":[2,{"w":3}]},"a":null,"m":"s"}`, &abi.OrderedMap[json.RawMessage]{})
	})
	t.Run("nested order-keeping map", func(t *testing.T) {
		roundTrip(t, `{"zz":`+authoredProbeObject+`,"8080":{},"a":{"y":"1","x":"2"}}`, &abi.OrderedMap[abi.OrderedMap[string]]{})
	})
	t.Run("a payload type with its own codec", func(t *testing.T) {
		// MutatorNode's labels are an order-keeping map too, written here in the
		// order they are held, so both levels round-trip; the order under test is
		// the outer container's own.
		roundTrip(t, `{"web":{"component":"c","resolved_version":"1","labels":{"a":"1","b":"2"}},`+
			`"8080":{"component":"d","resolved_version":"2","labels":{}}}`, &abi.OrderedMap[abi.MutatorNode]{})
	})
	t.Run("any, through encoding/json with numbers kept exact", func(t *testing.T) {
		m := &abi.OrderedMap[any]{}
		roundTrip(t, `{"z":12345678901234567890,"a":"s"}`, m)
		if v, _ := m.Get("z"); v != json.Number("12345678901234567890") {
			t.Errorf("an any-typed number decoded as %#v, want the exact json.Number", v)
		}
	})
}
