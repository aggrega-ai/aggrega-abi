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
// has to produce byte-identical output. A payload carries two kinds of map, and
// the contract reads differently for each. A plain map carries no order of its
// own, so every generated encoder sorts its keys before emitting them. A map the
// schema marks order-preserving carries its order as part of its content, so every
// generated encoder emits it in the order it holds and never sorts it. This file
// pins both rules across the whole payload surface rather than wherever a golden
// happens to populate a map.
//
// A golden round-trip cannot do that job, for a reason worth stating because it is
// invisible: the value under test is decoded from a golden whose keys are ALREADY
// in the emitted order, so an encoder that dropped its sort — or one that sorted
// where it must not — would reproduce that order anyway and the byte-compare would
// stay green. Either fault becomes observable only when the value reaches the
// encoder with its keys in some other order, which is what constructing the value
// here achieves.
//
// The plain-map probe keys are chosen so the two wire forms disagree about their
// order in every position: descending length with ascending first letter means the
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
// The order-preserving probe keys are set in an order that is neither sorted
// order and that holds a key which is an array index, so a sorting encoder in
// either wire's rule, and an encoder that hoists such a key as a plain JavaScript
// object does, each fail. Five rather than four, because the array-index key is the
// one the other four could not test. The order is deterministic by construction,
// so it needs no repetition to expose a fault.
//
// Scope: this file asserts the JSON wire. That is the leg a golden round-trip is
// blind to. The CBOR leg is pinned by the golden vectors themselves: a plain map's
// canonical CBOR order differs from the JSON order the goldens carry, and an
// order-preserving map's probe order is neither sorted order, so a CBOR encoder
// that dropped its sort, or sorted where it must not, emits different bytes and
// the vector fails.
var probeKeysInJSONOrder = []string{"aaaaaaaa", "bbbb", "cc", "d"}

// probeKeysInHeldOrder is the order the order-preserving probe containers are set
// in, which every wire must reproduce.
var probeKeysInHeldOrder = []string{"d", "8080", "bbbb", "aaaaaaaa", "cc"}

// stabilityRepeats is how many times a constructed payload is re-encoded when
// asserting that identical input yields byte-identical output. Each repeat is an
// independent draw of map iteration order, so a dropped sort survives all of them
// with probability that falls off geometrically.
const stabilityRepeats = 24

// mapBearingPayloads is EVERY generated payload that declares a map field of either
// kind, taken from the census of map field declarations across the generated codec
// rather than from a sample. TestMapBearingCensusIsComplete guards it against
// drift.
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
	abi.LinterInput{},
	abi.MutatorGraph{},
	abi.MutatorNode{},
	abi.RenderNodeOutput{},
	abi.RenderedNode{},
	abi.SubChart{},
	abi.WorkloadProjection{},
}

var rawMessageType = reflect.TypeOf(json.RawMessage(nil))

// abiPackagePath is the import path of the package whose order-keeping container
// the generated payloads use.
var abiPackagePath = reflect.TypeOf(abi.OrderedMap[string]{}).PkgPath()

// isOrderedMap reports whether t is an instantiation of the package's
// order-keeping container. A generic type's instantiations carry the generic
// name with their type arguments, which is what is matched.
func isOrderedMap(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t.PkgPath() == abiPackagePath && strings.HasPrefix(t.Name(), "OrderedMap[")
}

// probeEntryValue returns a value of type t usable as a map entry: a composite
// gets an empty-but-present value so the encoder walks it without dereferencing a
// nil, and a nested map of either kind is itself populated with its probe keys so
// the inner level's order is pinned too.
func probeEntryValue(t reflect.Type) reflect.Value {
	switch {
	case isOrderedMap(t):
		return probePopulatedOrderedMap(t)
	case t.Kind() == reflect.Map:
		return probePopulatedMap(t)
	case t.Kind() == reflect.Slice:
		// A raw-message field is carried verbatim, so it has to hold valid JSON.
		if t == rawMessageType {
			return reflect.ValueOf(json.RawMessage(`{}`))
		}
		return reflect.MakeSlice(t, 0, 0)
	case t.Kind() == reflect.Pointer:
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

// probePopulatedOrderedMap returns an order-keeping container of type t set with
// the order-preserving probe keys in their held order. The container is filled
// through its own Set, the API a guest fills one through, and the value type is
// read from that method's signature.
func probePopulatedOrderedMap(t reflect.Type) reflect.Value {
	m := reflect.New(t)
	set := m.MethodByName("Set")
	elem := set.Type().In(1)
	for _, k := range probeKeysInHeldOrder {
		set.Call([]reflect.Value{reflect.ValueOf(k), probeEntryValue(elem)})
	}
	return m.Elem()
}

// populateMapFields fills every map field of struct v, of either kind, with its
// probe keys and reports how many it filled, descending into nested struct fields
// so a map on an inlined payload is reached as well.
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
		case isOrderedMap(f.Type()):
			f.Set(probePopulatedOrderedMap(f.Type()))
			n++
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

// probeMaps are the emitted key orders of the probe maps found on one wire,
// split by kind: an object whose key set is the plain probe set, and one whose key
// set is the order-preserving probe set. The two sets differ, so an object is one
// kind or the other and never both.
type probeMaps struct {
	plain   [][]string
	ordered [][]string
}

// probeMapOrders walks the JSON token stream and returns the emitted key order of
// every object whose key set is exactly one of the probe sets. Reading the token
// stream rather than pattern-matching the text gets nesting exactly right, and
// keying on the probe sets is what separates a map from a struct: no generated
// struct declares a field by any probe name, and struct fields emit in declaration
// order rather than in a probe order.
func probeMapOrders(t *testing.T, doc []byte) probeMaps {
	t.Helper()

	type frame struct {
		isObject  bool
		expectKey bool
		keys      []string
	}
	var stack []*frame
	var found probeMaps

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
				if top.isObject {
					switch {
					case sameKeySet(top.keys, probeKeysInJSONOrder):
						found.plain = append(found.plain, top.keys)
					case sameKeySet(top.keys, probeKeysInHeldOrder):
						found.ordered = append(found.ordered, top.keys)
					}
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

// sameKeySet reports whether keys holds exactly the members of probe, in any
// order, which is what identifies an emitted object as one of the maps this test
// populated.
func sameKeySet(keys, probe []string) bool {
	if len(keys) != len(probe) {
		return false
	}
	for _, want := range probe {
		if !slices.Contains(keys, want) {
			return false
		}
	}
	return true
}

// TestJSONMapKeyOrderAcrossEveryMapBearingPayload constructs each map-bearing
// payload with its probe keys in every map it declares and asserts the JSON wire
// emits a plain map's keys lexicographically and an order-preserving map's keys in
// the order the map holds. Because each map is built rather than decoded, a lost
// sort — and a sort where there must be none — is observable here in a way no
// golden round-trip can reproduce.
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

			found := probeMapOrders(t, doc)
			if len(found.plain)+len(found.ordered) == 0 {
				t.Fatalf("%s emitted no probe map onto the JSON wire, so the ordering assertion proved nothing:\n wire: %s",
					name, doc)
			}
			for _, keys := range found.plain {
				if !slices.Equal(keys, probeKeysInJSONOrder) {
					t.Errorf("%s JSON plain-map keys not in lexicographic order: emitted %v, want %v\n wire: %s",
						name, keys, probeKeysInJSONOrder, doc)
				}
			}
			for _, keys := range found.ordered {
				if !slices.Equal(keys, probeKeysInHeldOrder) {
					t.Errorf("%s JSON order-preserving map keys not in the order the map holds: emitted %v, want %v\n wire: %s",
						name, keys, probeKeysInHeldOrder, doc)
				}
			}

			// Identical input has to produce byte-identical output. Each repeat
			// draws a fresh map iteration order, so an encoder that emitted a plain
			// map's keys in iteration order rather than sorted order diverges here
			// even when a single draw happens to come out sorted.
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
// payload, with every map that payload declares populated by its probe key set.
// They exist so the Rust and TypeScript harnesses reach the same ordering surface
// this file covers here: neither can walk the payload types the way the assertions
// above do — a TypeScript interface is erased before it runs — so the coverage has
// to arrive as committed fixture content rather than as constructed values.
//
// Running them through the shared parity helper is what makes them work. For a
// plain map, the helper's fourth contract sources the value from the committed CBOR
// bytes, whose key order is the reverse of the JSON golden's, so each language's
// JSON encoder has to re-sort to reproduce the golden. For an order-preserving map,
// the golden's order is neither sorted order, so an encoder that sorted the map in
// either wire's rule fails the byte-compare of the wire it sorted.
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
	t.Run("LinterInput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.LinterInput](t, "MapOrderLinterInput")
	})
	t.Run("MutatorGraph", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.MutatorGraph](t, "MapOrderMutatorGraph")
	})
	t.Run("MutatorNode", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.MutatorNode](t, "MapOrderMutatorNode")
	})
	// The policy payload declares no map of its own; it carries the graph, whose
	// nodes and a node's labels are order-preserving maps. This vector holds them in
	// a populated payload, as the ir-verdict role hands it to a guest, so the order
	// is pinned where a policy reads it and not only on the graph in isolation.
	t.Run("PolicyInput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.PolicyInput](t, "MapOrderPolicyInput")
	})
	t.Run("RenderNodeOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.RenderNodeOutput](t, "MapOrderRenderNodeOutput")
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
// it does, and against a field moving between the two kinds. Without it, a new
// map field would simply never be asserted and nothing would say so — and a
// marked field whose binding had stopped being order-preserving would be asserted
// against the sorting rule instead of failing.
func TestMapBearingCensusIsComplete(t *testing.T) {
	t.Parallel()

	// The declared map fields across the generated codec, by kind. Counted from
	// the field declarations, so a nested map counts once as the field that
	// declares it, under the kind of its outermost container: a plain map of
	// order-preserving ones counts as plain.
	const (
		wantPlainMapFields   = 12
		wantOrderedMapFields = 31
	)

	plain, ordered := 0, 0
	for _, proto := range mapBearingPayloads {
		p, o := countDeclaredMapFields(reflect.TypeOf(proto))
		plain += p
		ordered += o
	}
	if plain != wantPlainMapFields || ordered != wantOrderedMapFields {
		t.Errorf("map fields reachable from the census = %d plain and %d order-preserving; want %d and %d.\n"+
			"A payload gained or lost a map field, or a field changed kind. Update mapBearingPayloads and these "+
			"counts together so the ordering assertion keeps covering every map that reaches the wire under the "+
			"rule its kind carries.", plain, ordered, wantPlainMapFields, wantOrderedMapFields)
	}
}

// countDeclaredMapFields counts the map fields a type declares itself, by kind. It
// does NOT descend into struct fields, because a payload that embeds another
// map-bearing payload would otherwise count that payload's maps twice — once
// through the embedding field and again through the embedded type's own census
// entry.
func countDeclaredMapFields(t reflect.Type) (plain, ordered int) {
	for i := 0; i < t.NumField(); i++ {
		switch ft := t.Field(i).Type; {
		case isOrderedMap(ft):
			ordered++
		case ft.Kind() == reflect.Map:
			plain++
		}
	}
	return plain, ordered
}
