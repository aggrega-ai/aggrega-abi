// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// Map-key ordering is a determinism contract, not a cosmetic one: identical input
// has to produce byte-identical output, and a map is the one container whose
// iteration carries no inherent order. Every generated encoder therefore sorts a
// map's keys before emitting them, and this file pins that sort across the whole
// payload surface rather than wherever a golden happens to populate a map.
//
// A golden round-trip cannot do that job, for a reason worth stating because it is
// invisible: the value under test is decoded from a golden whose keys are ALREADY
// in the emitted order, so an encoder that dropped its sort would reproduce that
// order anyway and the byte-compare would stay green. The sort becomes observable
// only when the value reaches the encoder with its keys in some other order, which
// is what constructing the value here achieves.
//
// The probe keys are chosen so the two wire forms disagree about their order in
// every position: descending length with ascending first letter means the
// lexicographic order JSON emits is the exact reverse of the length-led order CBOR
// emits. A codec that applied the CBOR rule to the JSON wire would therefore fail
// on the first key rather than coincidentally agreeing.
//
// Four keys rather than two, because map iteration order is unspecified: a
// constructed map arrives at the encoder in an arbitrary order, so a dropped sort
// is only observable when the arbitrary order happens to differ from the sorted
// one. Two keys leave that to a coin flip per map; four leave one chance in
// twenty-four, and the stability assertion below closes the residue by encoding
// repeatedly.
//
// Scope: this file asserts the JSON wire. That is the leg a golden round-trip is
// blind to. The CBOR leg is pinned by the golden vectors themselves, because the
// canonical CBOR order differs from the JSON order the goldens carry, so a CBOR
// encoder that dropped its sort emits different bytes and the vector fails.
var probeKeysInJSONOrder = []string{"aaaaaaaa", "bbbb", "cc", "d"}

// stabilityRepeats is how many times a constructed payload is re-encoded when
// asserting that identical input yields byte-identical output. Each repeat is an
// independent draw of map iteration order, so a dropped sort survives all of them
// with probability that falls off geometrically.
const stabilityRepeats = 24

// mapBearingPayloads is EVERY generated payload that declares a map field, taken
// from the census of map field declarations across the generated codec rather than
// from a sample. TestMapBearingCensusIsComplete guards it against drift.
var mapBearingPayloads = []any{
	abi.AnnotationTarget{},
	abi.ArtifactProjectionEntry{},
	abi.BoundaryEnvOverridesInput{},
	abi.BoundaryEnvOverridesOutput{},
	abi.DocumentSet{},
	abi.EmittedEntity{},
	abi.FRPStepInput{},
	abi.FRPStepOutput{},
	abi.IRFeatureEntry{},
	abi.IRNode{},
	abi.IRNodeProjection{},
	abi.IRSpec{},
	abi.LinkEdge{},
	abi.LinkPublishOutput{},
	abi.LinkResolveInput{},
	abi.MutatorGraph{},
	abi.MutatorNode{},
	abi.RenderedNode{},
	abi.SubChart{},
	abi.WorkloadProjection{},
}

var rawMessageType = reflect.TypeOf(json.RawMessage(nil))

// probeEntryValue returns a value of type t usable as a map entry: a composite
// gets an empty-but-present value so the encoder walks it without dereferencing a
// nil, and a nested map is itself populated with both probe keys so the inner
// level's sort is pinned too.
func probeEntryValue(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.Map:
		return probePopulatedMap(t)
	case reflect.Slice:
		// A raw-message field is carried verbatim, so it has to hold valid JSON.
		if t == rawMessageType {
			return reflect.ValueOf(json.RawMessage(`{}`))
		}
		return reflect.MakeSlice(t, 0, 0)
	case reflect.Pointer:
		p := reflect.New(t.Elem())
		p.Elem().Set(probeEntryValue(t.Elem()))
		return p
	default:
		return reflect.New(t).Elem()
	}
}

func probePopulatedMap(t reflect.Type) reflect.Value {
	m := reflect.MakeMap(t)
	for _, k := range probeKeysInJSONOrder {
		m.SetMapIndex(reflect.ValueOf(k), probeEntryValue(t.Elem()))
	}
	return m
}

// populateMapFields fills every map field of struct v with both probe keys and
// reports how many it filled, descending into nested struct fields so a map on an
// inlined payload is reached as well.
//
// It also seeds any required raw-message field with an empty object. Such a field
// is carried verbatim, so leaving it at its zero value would put an empty byte
// slice where the wire needs a JSON value and the marshal would fail before
// reaching the ordering assertion.
func populateMapFields(v reflect.Value) int {
	n := 0
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch {
		case f.Kind() == reflect.Map:
			f.Set(probePopulatedMap(f.Type()))
			n++
		case f.Type() == rawMessageType:
			if f.Len() == 0 && !strings.Contains(v.Type().Field(i).Tag.Get("json"), "omitempty") {
				f.Set(reflect.ValueOf(json.RawMessage(`{}`)))
			}
		case f.Kind() == reflect.Struct:
			n += populateMapFields(f)
		}
	}
	return n
}

// probeMapOrders walks the JSON token stream and returns the emitted key order of
// every object whose key set is exactly the two probe keys. Reading the token
// stream rather than pattern-matching the text gets nesting exactly right, and
// keying on the probe set is what separates a map from a struct: no generated
// struct declares a field by either probe name, and struct fields emit in
// declaration order rather than sorted.
func probeMapOrders(t *testing.T, doc []byte) [][]string {
	t.Helper()

	type frame struct {
		isObject  bool
		expectKey bool
		keys      []string
	}
	var stack []*frame
	var found [][]string

	dec := json.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("walk JSON token stream: %v\n wire: %s", err, doc)
		}

		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{':
				stack = append(stack, &frame{isObject: true, expectKey: true})
				continue
			case '[':
				stack = append(stack, &frame{})
				continue
			case '}', ']':
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if top.isObject && isProbeMap(top.keys) {
					found = append(found, top.keys)
				}
				// The completed object or array was a value, so its parent object
				// is awaiting the next key.
				if len(stack) > 0 && stack[len(stack)-1].isObject {
					stack[len(stack)-1].expectKey = true
				}
				continue
			}
		}

		if len(stack) == 0 {
			continue
		}
		if top := stack[len(stack)-1]; top.isObject {
			if top.expectKey {
				top.keys = append(top.keys, tok.(string))
				top.expectKey = false
			} else {
				top.expectKey = true
			}
		}
	}
	return found
}

// isProbeMap reports whether keys is exactly the probe key set, which is what
// identifies an emitted object as one of the maps this test populated.
func isProbeMap(keys []string) bool {
	if len(keys) != len(probeKeysInJSONOrder) {
		return false
	}
	for _, want := range probeKeysInJSONOrder {
		found := false
		for _, k := range keys {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// TestJSONMapKeyOrderAcrossEveryMapBearingPayload constructs each map-bearing
// payload with both probe keys in every map it declares and asserts the JSON wire
// emits them lexicographically. Because the map is built rather than decoded, a
// lost sort is observable here in a way no golden round-trip can reproduce.
func TestJSONMapKeyOrderAcrossEveryMapBearingPayload(t *testing.T) {
	t.Parallel()

	for _, proto := range mapBearingPayloads {
		proto := proto
		name := reflect.TypeOf(proto).Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			v := reflect.New(reflect.TypeOf(proto)).Elem()
			if filled := populateMapFields(v); filled == 0 {
				t.Fatalf("%s is listed as map-bearing but no map field was populated", name)
			}

			doc, err := json.Marshal(v.Interface())
			if err != nil {
				t.Fatalf("marshal %s: %v", name, err)
			}

			orders := probeMapOrders(t, doc)
			if len(orders) == 0 {
				t.Fatalf("%s emitted no probe map onto the JSON wire, so the ordering assertion proved nothing:\n wire: %s",
					name, doc)
			}
			for _, keys := range orders {
				if !slices.Equal(keys, probeKeysInJSONOrder) {
					t.Errorf("%s JSON map keys not in lexicographic order: emitted %v, want %v\n wire: %s",
						name, keys, probeKeysInJSONOrder, doc)
				}
			}

			// Identical input has to produce byte-identical output. Each repeat
			// draws a fresh map iteration order, so an encoder that emitted keys
			// in iteration order rather than sorted order diverges here even when
			// a single draw happens to come out sorted.
			for i := 0; i < stabilityRepeats; i++ {
				again, err := json.Marshal(v.Interface())
				if err != nil {
					t.Fatalf("re-marshal %s: %v", name, err)
				}
				if !bytes.Equal(doc, again) {
					t.Fatalf("%s JSON wire is not stable across repeated encoding of one value:\n first: %s\n  then: %s",
						name, doc, again)
				}
			}
		})
	}
}

// The map-order probe vectors carry one committed golden pair per map-owning
// payload, with every map that payload declares populated by the probe key set.
// They exist so the Rust and TypeScript harnesses reach the same ordering surface
// this file covers here: neither can walk the payload types the way the assertions
// above do — a TypeScript interface is erased before it runs — so the coverage has
// to arrive as committed fixture content rather than as constructed values.
//
// Running them through the shared parity helper is what makes them work: the
// helper's fourth contract sources the value from the committed CBOR bytes, whose
// key order is the reverse of the JSON golden's, so each language's JSON encoder
// has to re-sort to reproduce the golden.
func TestSocketParity_MapOrderProbes(t *testing.T) {
	t.Parallel()

	t.Run("AnnotationTarget", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.AnnotationTarget](t, "MapOrderAnnotationTarget")
	})
	t.Run("ArtifactProjectionEntry", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.ArtifactProjectionEntry](t, "MapOrderArtifactProjectionEntry")
	})
	t.Run("BoundaryEnvOverridesInput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.BoundaryEnvOverridesInput](t, "MapOrderBoundaryEnvOverridesInput")
	})
	t.Run("BoundaryEnvOverridesOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.BoundaryEnvOverridesOutput](t, "MapOrderBoundaryEnvOverridesOutput")
	})
	t.Run("DocumentSet", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.DocumentSet](t, "MapOrderDocumentSet")
	})
	t.Run("EmittedEntity", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.EmittedEntity](t, "MapOrderEmittedEntity")
	})
	t.Run("FRPStepInput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.FRPStepInput](t, "MapOrderFRPStepInput")
	})
	t.Run("FRPStepOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.FRPStepOutput](t, "MapOrderFRPStepOutput")
	})
	t.Run("IRFeatureEntry", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.IRFeatureEntry](t, "MapOrderIRFeatureEntry")
	})
	t.Run("IRNode", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.IRNode](t, "MapOrderIRNode")
	})
	t.Run("IRNodeProjection", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.IRNodeProjection](t, "MapOrderIRNodeProjection")
	})
	t.Run("IRSpec", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.IRSpec](t, "MapOrderIRSpec")
	})
	t.Run("LinkEdge", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.LinkEdge](t, "MapOrderLinkEdge")
	})
	t.Run("LinkPublishOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.LinkPublishOutput](t, "MapOrderLinkPublishOutput")
	})
	t.Run("LinkResolveInput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.LinkResolveInput](t, "MapOrderLinkResolveInput")
	})
	t.Run("MutatorGraph", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.MutatorGraph](t, "MapOrderMutatorGraph")
	})
	t.Run("MutatorNode", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.MutatorNode](t, "MapOrderMutatorNode")
	})
	t.Run("RenderedNode", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.RenderedNode](t, "MapOrderRenderedNode")
	})
	t.Run("SubChart", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.SubChart](t, "MapOrderSubChart")
	})
	t.Run("WorkloadProjection", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.WorkloadProjection](t, "MapOrderWorkloadProjection")
	})
}

// TestMapBearingCensusIsComplete guards the census against a schema change that
// puts a map field on a payload the list does not name, or adds one to a payload
// it does. Without it, a new map field would simply never be asserted and nothing
// would say so.
func TestMapBearingCensusIsComplete(t *testing.T) {
	t.Parallel()

	// The declared map fields across the generated codec. Counted from the field
	// declarations, so a nested map counts once as the field that declares it.
	const wantMapFields = 39

	got := 0
	for _, proto := range mapBearingPayloads {
		got += countDeclaredMapFields(reflect.TypeOf(proto))
	}
	if got != wantMapFields {
		t.Errorf("map fields reachable from the census = %d; want %d.\n"+
			"A payload gained or lost a map field. Update mapBearingPayloads and this count together "+
			"so the ordering assertion keeps covering every map that reaches the wire.", got, wantMapFields)
	}
}

// countDeclaredMapFields counts the map fields a type declares itself. It does
// NOT descend into struct fields, because a payload that embeds another
// map-bearing payload would otherwise count that payload's maps twice — once
// through the embedding field and again through the embedded type's own census
// entry. Two payloads embed another this way, so the difference is not
// hypothetical: it is six fields on a surface of forty-five.
func countDeclaredMapFields(t reflect.Type) int {
	n := 0
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type.Kind() == reflect.Map {
			n++
		}
	}
	return n
}
