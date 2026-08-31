// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// The failure envelope is the path that carries bad news: a socket output's error
// member is what a guest sets when it ran but could not complete, and the host
// halts the build on it. A codec defect there surfaces only once something has
// already gone wrong, and what gets corrupted is the diagnostic itself — so the
// envelope is worth pinning at least as much as the success payload.
//
// Shape of each vector, and why it is this shape rather than a fuller one: the
// error message is set and the success members are left at their zero value. The
// host abandons the payload the moment it reads a non-empty error — it returns
// without reading the patches, the files, or even the diagnostics — so an error
// set beside substantively populated success content is a state nothing ever acts
// on, and pinning it would pin fiction. Where the schema makes a success member
// required, that member stays present but empty, which is what a failing guest
// has to emit to satisfy the contract at all: an empty list, or an empty object
// for a verbatim-carried tree.
//
// The two host-function replies that already carried a failure golden before this
// file existed are deliberately not duplicated here; the census below names them
// and asserts their goldens still populate the member.

// errorEnvelopeVectors are the socket output types whose error member this file's
// goldens populate, paired with the vector name.
var errorEnvelopeVectors = []struct {
	vector string
	proto  any
}{
	{"ErrorEnvelopeAnnotationTransformOutput", abi.AnnotationTransformOutput{}},
	{"ErrorEnvelopeAssembleBoundaryOutput", abi.AssembleBoundaryOutput{}},
	{"ErrorEnvelopeFRPStepOutput", abi.FRPStepOutput{}},
	{"ErrorEnvelopeImportOutput", abi.ImportOutput{}},
	{"ErrorEnvelopeLinterOutput", abi.LinterOutput{}},
	{"ErrorEnvelopeMutatorOutput", abi.MutatorOutput{}},
	{"ErrorEnvelopeNameBoundOutput", abi.NameBoundOutput{}},
	{"ErrorEnvelopeParseOutput", abi.ParseOutput{}},
	{"ErrorEnvelopePolicyOutput", abi.PolicyOutput{}},
	{"ErrorEnvelopeRenderNodeOutput", abi.RenderNodeOutput{}},
	{"ErrorEnvelopeSourceFileResponse", abi.SourceFileResponse{}},
	{"ErrorEnvelopeSourcePackResponse", abi.SourcePackResponse{}},
	{"ErrorEnvelopeValidateNodeOutput", abi.ValidateNodeOutput{}},
}

// preExistingErrorGoldens are the failure envelopes the suite already carried.
// They are the host-function replies, whose whole populated form IS the failure
// case, which is why they were the only error members with any coverage.
var preExistingErrorGoldens = map[string]any{
	"ParseSourceResponse":        abi.ParseSourceResponse{},
	"FetchExternalInputResponse": abi.FetchExternalInputResponse{},
}

func TestSocketParity_ErrorEnvelopes(t *testing.T) {
	t.Parallel()

	t.Run("AnnotationTransformOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.AnnotationTransformOutput](t, "ErrorEnvelopeAnnotationTransformOutput")
	})
	t.Run("AssembleBoundaryOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.AssembleBoundaryOutput](t, "ErrorEnvelopeAssembleBoundaryOutput")
	})
	t.Run("FRPStepOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.FRPStepOutput](t, "ErrorEnvelopeFRPStepOutput")
	})
	t.Run("ImportOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.ImportOutput](t, "ErrorEnvelopeImportOutput")
	})
	t.Run("LinterOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.LinterOutput](t, "ErrorEnvelopeLinterOutput")
	})
	t.Run("MutatorOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.MutatorOutput](t, "ErrorEnvelopeMutatorOutput")
	})
	t.Run("NameBoundOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.NameBoundOutput](t, "ErrorEnvelopeNameBoundOutput")
	})
	t.Run("ParseOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.ParseOutput](t, "ErrorEnvelopeParseOutput")
	})
	t.Run("PolicyOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.PolicyOutput](t, "ErrorEnvelopePolicyOutput")
	})
	t.Run("RenderNodeOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.RenderNodeOutput](t, "ErrorEnvelopeRenderNodeOutput")
	})
	t.Run("SourceFileResponse", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.SourceFileResponse](t, "ErrorEnvelopeSourceFileResponse")
	})
	t.Run("SourcePackResponse", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.SourcePackResponse](t, "ErrorEnvelopeSourcePackResponse")
	})
	t.Run("ValidateNodeOutput", func(t *testing.T) {
		t.Parallel()
		roundTripBothGolden[abi.ValidateNodeOutput](t, "ErrorEnvelopeValidateNodeOutput")
	})
}

// TestEveryErrorMemberIsPopulatedBySomeGolden is the guard that makes the census
// above self-checking. A byte-compare vector proves the wire round-trips, but it
// cannot notice that the member under test was empty in the golden all along —
// which is exactly how this whole class stayed invisible while the suite was
// green. So each vector is decoded and its error member asserted non-empty.
func TestEveryErrorMemberIsPopulatedBySomeGolden(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, vector string, proto any) {
		t.Helper()
		golden := readGolden(t, vector+".golden.json")
		v := reflect.New(reflect.TypeOf(proto))
		if err := json.Unmarshal(golden, v.Interface()); err != nil {
			t.Fatalf("decode %s: %v", vector, err)
		}
		f := v.Elem().FieldByName("Error")
		if !f.IsValid() {
			t.Fatalf("%s declares no error member, so it does not belong in the error-envelope census",
				reflect.TypeOf(proto).Name())
		}
		if strings.TrimSpace(f.String()) == "" {
			t.Errorf("%s carries an empty error member, so the vector proves nothing about the failure envelope:\n %s",
				vector, golden)
		}
	}

	for _, c := range errorEnvelopeVectors {
		t.Run(c.vector, func(t *testing.T) {
			t.Parallel()
			check(t, c.vector, c.proto)
		})
	}
	for vector, proto := range preExistingErrorGoldens {
		t.Run(vector, func(t *testing.T) {
			t.Parallel()
			check(t, vector, proto)
		})
	}
}

// TestErrorBearingCensusIsComplete counts the generated types that declare an
// error member and compares that to the census this file covers. A new socket
// output type ships with the member as shared boilerplate, so without this guard
// its failure envelope would go unpinned exactly as the twelve above did, and
// nothing would say so.
func TestErrorBearingCensusIsComplete(t *testing.T) {
	t.Parallel()

	// The number of error members is counted from the generated sources rather
	// than written down beside the census. Comparing the census to a constant a
	// maintainer keeps in step with it proves only that two hand-written numbers
	// agree; counting the declarations is what makes a newly generated member
	// fail here.
	declared := countErrorMemberDeclarations(t)
	covered := len(errorEnvelopeVectors) + len(preExistingErrorGoldens)
	if covered != declared {
		t.Errorf("error members declared by the generated codec = %d; covered by a vector = %d.\n"+
			"A socket output type gained or lost its error member. Extend the census so every failure "+
			"envelope keeps a vector that populates it.", declared, covered)
	}

	// Each censused type must actually declare the member, so a rename cannot
	// leave the census pointing at a type that no longer carries one.
	all := make([]any, 0, covered)
	for _, c := range errorEnvelopeVectors {
		all = append(all, c.proto)
	}
	for _, p := range preExistingErrorGoldens {
		all = append(all, p)
	}
	for _, proto := range all {
		rt := reflect.TypeOf(proto)
		if _, ok := rt.FieldByName("Error"); !ok {
			t.Errorf("%s is in the error-envelope census but declares no error member", rt.Name())
		}
	}
}

// countErrorMemberDeclarations counts the error members the generated codec
// declares, by reading the generated sources in this package directory. Reading
// the declarations is deliberate: reflection cannot enumerate a package's types,
// and a hand-maintained constant would only ever agree with the hand-maintained
// census it is supposed to check.
func countErrorMemberDeclarations(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	const tag = `json:"error,omitempty"`
	n := 0
	generated := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_gen.go") {
			continue
		}
		generated++
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		n += strings.Count(string(b), tag)
	}
	if generated == 0 {
		t.Fatal("found no generated sources in the package directory, so the count certifies nothing")
	}
	return n
}
