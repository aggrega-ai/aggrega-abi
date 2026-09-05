// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The vector corpus under testdata/ is ONE golden read by every harness that
// enumerates it, and the value of that arrangement is entirely in the word "one".
// Codecs generated from a single schema are only proven to produce a single wire
// if all of them are held to the same bytes; a vector exercised in one language
// and not another proves that language's codec self-consistent and says nothing
// about parity.
//
// Nothing structural keeps the harnesses aligned. Each enumerates its vectors in
// its own idiom — call sites here, macro invocations in Rust, an array in
// TypeScript — so agreement across them is a hand-maintained property, and a
// vector pair committed to testdata/ is wired by whoever remembers to wire it
// everywhere. The failure mode is silent in the worst way: the new vector passes
// wherever it was wired, every existing vector keeps passing, and the harness that
// never received it reports nothing at all. Coverage does not go red when it is
// missing, because missing coverage has no assertion to fail.
//
// This file closes that gap on the Go side by deriving coverage FROM THE CORPUS
// rather than from a list. The directory is the authority: whatever pairs are
// committed there are what the harness owes, so a new pair is covered the moment
// it lands or this test names it. Other harnesses carry the same guard against the
// same directory, and NOT ALL OF THEM LIVE IN THIS REPOSITORY — the compiler's
// independent byte-parity oracle carries one too, holding these vectors against an
// encoder this schema did not generate. So a guard added or removed elsewhere is
// invisible from here, which is why this says what the property is rather than how
// many harnesses hold it: a count here answers to nothing and goes stale the moment
// one is added.
//
// Deriving the exercised set from the parity call sites — reading this package's
// own test sources — is deliberate, and follows the same reasoning as the
// error-member census in this package: a hand-written list of vector names
// checked against the directory would only move the drift one step, leaving the
// list and the call sites free to disagree. Reading the call sites means there is
// no second copy of the vector set anywhere in the harness.

// goldenPairSuffixes are the two files a vector pair is made of. A vector is the
// pair, not either half: the JSON and CBOR codecs are separate generated methods,
// so a name carrying only one of the two leaves the other codec unpinned.
var goldenPairSuffixes = []string{".golden.json", ".cbor.hex"}

// standaloneFixtures are the testdata files that are deliberately NOT vector
// pairs, each consumed by its own dedicated test in every harness that reads this
// tree. They are named so that a fixture arriving in the corpus and wired nowhere
// cannot pass as one of these.
var standaloneFixtures = []string{"float_vectors.json"}

// parityCallSite matches an invocation of the shared parity helper and captures
// the vector name it asserts.
var parityCallSite = regexp.MustCompile(`roundTripBothGolden\[[^\]]*\]\(\s*t,\s*"([^"]+)"\s*\)`)

func testdataDir() string {
	return filepath.Join("..", "..", "..", "testdata")
}

// goldenCorpus returns the vector names committed to testdata/, having first
// asserted that every name carries BOTH halves of its pair and that every file in
// the directory is either half of a pair or a named standalone fixture. An
// unrecognized file is a failure rather than something to skip, because a fixture
// nothing consumes is the same coverage hole this file exists to close.
func goldenCorpus(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(testdataDir())
	if err != nil {
		t.Fatalf("read the shared vector corpus: %v", err)
	}

	halves := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("the shared vector corpus carries a directory %q; it holds vector files only", e.Name())
			continue
		}
		name := e.Name()
		if slices.Contains(standaloneFixtures, name) {
			continue
		}
		matched := false
		for _, suffix := range goldenPairSuffixes {
			if base, ok := strings.CutSuffix(name, suffix); ok {
				halves[base] = append(halves[base], suffix)
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("the shared vector corpus carries %q, which is neither half of a vector pair nor a "+
				"fixture any harness is known to consume, so nothing asserts anything about it", name)
		}
	}

	var corpus []string
	for base, got := range halves {
		slices.Sort(got)
		if !slices.Equal(got, []string{".cbor.hex", ".golden.json"}) {
			t.Errorf("vector %q carries only %v; a vector is the JSON and CBOR pair, and a name carrying "+
				"one half leaves the other codec unpinned", base, got)
			continue
		}
		corpus = append(corpus, base)
	}
	slices.Sort(corpus)

	if len(corpus) == 0 {
		t.Fatal("found no vector pairs in the shared corpus, so this census certifies nothing")
	}
	return corpus
}

// TestEveryGoldenVectorIsExercised asserts that the vectors this harness asserts
// are exactly the vector pairs committed to the shared corpus — no pair left
// unexercised, and no vector asserted against a pair that is not there.
func TestEveryGoldenVectorIsExercised(t *testing.T) {
	t.Parallel()

	corpus := goldenCorpus(t)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	exercised := map[string]bool{}
	sources := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		sources++
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, m := range parityCallSite.FindAllStringSubmatch(string(b), -1) {
			exercised[m[1]] = true
		}
	}
	if sources == 0 {
		t.Fatal("found no test sources in the package directory, so the coverage census certifies nothing")
	}
	if len(exercised) == 0 {
		t.Fatal("found no parity assertions in the package's test sources, so the coverage census " +
			"certifies nothing — the call-site pattern has stopped matching the code it reads")
	}

	for _, name := range corpus {
		if !exercised[name] {
			t.Errorf("vector %q is committed to the shared corpus but no parity assertion in this harness "+
				"reads it, so this language's codec is unpinned on a wire the others prove", name)
		}
	}

	var orphans []string
	for name := range exercised {
		if !slices.Contains(corpus, name) {
			orphans = append(orphans, name)
		}
	}
	slices.Sort(orphans)
	for _, name := range orphans {
		t.Errorf("a parity assertion reads vector %q, which the shared corpus does not carry", name)
	}
}
