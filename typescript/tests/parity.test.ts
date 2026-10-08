// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

// Cross-language wire-parity tests for the generated socket codecs.
//
// The goldens under the shared ../../testdata tree are the canonical wire bytes —
// the same tree the Go and Rust harnesses validate against, so a match here is a
// cross-language parity assertion: the three codecs are generated from one schema
// and produce one wire. Each vector carries a JSON document (<Name>.golden.json)
// and the canonical CBOR bytes (<Name>.cbor.hex).
//
// Four contracts are proven per vector. The JSON wire round-trips byte-identical
// through the generated codec. The generated CBOR encoder reproduces the committed
// canonical bytes. Those bytes decode back to a byte-stable value. And the
// COMMITTED CBOR bytes decode into a value whose JSON encoding is the committed
// JSON golden.
//
// That fourth contract carries two assertions the other three structurally cannot
// make. It is the only one that feeds the CBOR decoder the committed canonical
// bytes — the other CBOR legs decode this codec's own fresh output, so an encoder
// and decoder that shared a compensating defect would agree with each other and
// stay green. And it is the only one that can observe a map-key sort being lost
// from the JSON encoder: the two wire forms order map keys by different rules —
// JSON lexicographically, CBOR by the encoded key bytes, whose length head leads —
// so a value arriving from the CBOR wire presents its keys in an order the JSON
// wire must actively re-sort. A value decoded from the JSON golden instead arrives
// already in JSON order, because JSON.parse preserves the key order of the
// document it reads, which is why the first contract stays green whether the
// encoder sorts or not.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import * as abi from "../src/index.ts";

const here = dirname(fileURLToPath(import.meta.url));
const testdata = join(here, "..", "..", "testdata");

// goldenJson reads a JSON golden VERBATIM. Its committed bytes are the wire bytes,
// carrying no terminating newline, so the byte-compare against the encoder output
// is itself the strictest possible check on the file's shape and a separate
// assertion would only restate it. Reading it verbatim is what makes that compare
// mean anything: a reader that trimmed surrounding whitespace could not
// distinguish the committed file from one whose bytes had drifted, and a stray
// newline would be absorbed here while a stricter harness rejected it — leaving the
// same file meaning two different things in two languages, which is exactly what a
// shared corpus cannot afford.
function goldenJson(name: string): string {
  return readFileSync(join(testdata, `${name}.golden.json`), "utf8");
}

// goldenCbor reads a canonical-CBOR golden and returns its hex body, holding the
// file to an exact on-disk shape: the lowercase hex digits, then exactly one
// terminating newline, and nothing else anywhere. Admitting exactly one form is
// what keeps the corpus single-valued across the three harnesses — a whitespace
// variant one of them silently accepted would be a fixture the languages read
// differently.
function goldenCbor(name: string): string {
  const raw = readFileSync(join(testdata, `${name}.cbor.hex`), "utf8");
  if (!raw.endsWith("\n")) {
    throw new Error(
      `golden ${name}.cbor.hex does not end in a newline, so its committed shape is not the canonical one: ${JSON.stringify(raw)}`,
    );
  }
  const body = raw.slice(0, -1);
  const stray = body.search(/[^0-9a-f]/);
  if (stray >= 0) {
    throw new Error(
      `golden ${name}.cbor.hex carries ${JSON.stringify(body[stray])} at offset ${stray}, so it is not ` +
        `lowercase hex followed by one newline: ${JSON.stringify(raw)}`,
    );
  }
  return body;
}

function toHex(u8: Uint8Array): string {
  let s = "";
  for (const b of u8) s += b.toString(16).padStart(2, "0");
  return s;
}

function fromHex(h: string): Uint8Array {
  const out = new Uint8Array(h.length / 2);
  for (let i = 0; i < out.length; i++) out[i] = parseInt(h.substr(i * 2, 2), 16);
  return out;
}

// The vectors, mapping each golden file to the interface type whose codec it
// exercises. The codec functions are resolved by name off the module namespace.
const vectors: Array<[string, string]> = [
  ["CLIRunInput", "CLIRunInput"],
  ["FRPStepInput", "FRPStepInput"],
  ["FRPStepOutput", "FRPStepOutput"],
  ["MutatorInput", "MutatorInput"],
  ["MutatorOutput", "MutatorOutput"],
  ["LinterInput", "LinterInput"],
  // The input of a rule that reads a whole Release, with the Release and the
  // Component map POPULATED: both are omitted when empty, so this is the vector
  // where dropping either moves the bytes. The two instance names sort
  // differently in the two wire forms ("agent" first in the JSON, "db" first in
  // the canonical CBOR), and each document keeps its keys out of key order; this
  // package holds each document PARSED and re-serialises it, so the vector pins
  // that the re-serialisation keeps the order it was given.
  ["LinterInputRelease", "LinterInput"],
  ["LinterOutput", "LinterOutput"],
  ["IRGraph", "IRGraph"],
  ["PolicyInput", "PolicyInput"],
  ["PolicyOutput", "PolicyOutput"],
  ["RenderNodeInput", "RenderNodeInput"],
  ["RenderNodeOutput", "RenderNodeOutput"],
  // A render output carrying a POPULATED fact list: the field is omitted when
  // empty, so this is the vector where dropping it moves the bytes. Its facts are
  // listed out of name order and its object value's keys out of key order; this
  // package holds each value PARSED and re-serialises it, so the vector pins that
  // the re-serialisation keeps the guest's own order rather than sorting it.
  ["RenderNodeOutputRenderFacts", "RenderNodeOutput"],
  ["IRNodeProjectionSensitiveRenderedKeys", "IRNodeProjection"],
  ["IRFeatureEntrySensitivePayloadLeaves", "IRFeatureEntry"],
  // An attachment entry carrying the connection it was written on, beside one of the
  // node's own carrying none: the field is omitted when absent, so this is the vector
  // where dropping it, or emitting it unconditionally, moves the bytes. The two share
  // one node's bucket, so the vector is the graph's spec, with the edge the
  // connection names.
  ["IRSpecConnectionAttachment", "IRSpec"],
  // The realization an attachment carries, on a node's entries and on a Release
  // boundary's: the field is omitted when absent, so this is the vector where
  // dropping it moves the bytes, and the one entry carrying none is where emitting
  // it unconditionally would.
  ["IRSpecAttachmentRealization", "IRSpec"],
  ["IRNodeSensitivePayloadLeaves", "IRNode"],
  ["IRBoundaryFeatureSensitivePayloadLeaves", "IRBoundaryFeature"],
  ["AssembleBoundaryInput", "AssembleBoundaryInput"],
  ["AssembleBoundaryOutput", "AssembleBoundaryOutput"],
  ["LinkPublishInput", "LinkPublishInput"],
  ["LinkPublishOutput", "LinkPublishOutput"],
  ["LinkResolveInput", "LinkResolveInput"],
  ["LinkResolveOutput", "LinkResolveOutput"],
  ["PluginDescriptorCLI", "PluginDescriptor"],
  ["PluginDescriptorProvider", "PluginDescriptor"],
  ["PluginDescriptorMinimal", "PluginDescriptor"],
  ["PluginDescriptorMoo", "PluginDescriptor"],
  ["ParseInput", "ParseInput"],
  ["ParseOutput", "ParseOutput"],
  ["ParseOutputBare", "ParseOutput"],
  ["ImportInput", "ImportInput"],
  ["ImportOutput", "ImportOutput"],
  ["ParseSourceResponse", "ParseSourceResponse"],
  // The parse-source reply's SUCCESS shape, the half its failure envelope cannot
  // stand in for: the two are the two directions of one omission property, and an
  // encoder emitting an error key unconditionally passes any test built only from
  // success fixtures while one dropping it unconditionally passes any test built
  // only from failure fixtures. It also carries the opaque tree, which crosses as
  // a byte string of the JSON document's verbatim bytes; the failure envelope
  // reaches no opaque field at all, so the tree member's bytes are pinned here.
  // This package is the one that holds the tree as a PARSED value and
  // re-serialises it, where the other two carry the bytes verbatim, so the vector
  // pins that the re-serialisation reproduces the committed document exactly. The
  // document is long enough that its byte-string length needs a two-byte head.
  ["ParseSourceResponseTree", "ParseSourceResponse"],
  ["FetchExternalInputResponse", "FetchExternalInputResponse"],
  ["FetchExternalInputResponseContent", "FetchExternalInputResponse"],
  // The pack-source and read-source-file request and SUCCESS reply vectors. The
  // success reply is the half a failure envelope cannot stand in for: the two are the
  // two directions of one omission property, and an encoder emitting an error key
  // unconditionally passes any test built only from success fixtures while one
  // dropping it unconditionally passes any test built only from failure fixtures.
  // Each reply type's failure envelope is listed among the error envelopes below;
  // the pair is the assertion.
  //
  // The request vectors declare no omittable member, so their map head is a constant
  // and there is no omission direction to prove. They pin head and key order on the
  // payload the guest builds and the host decodes.
  //
  // The packed reply carries three files in the host's byte-order walk — Chart.yaml
  // before charts/ before templates/, byte order rather than case-insensitive order —
  // and one file's content is deliberately not valid UTF-8, proving the file bytes
  // cross as an opaque byte string rather than as text. This package declares those
  // bytes as a base64 string, so the vector also pins that the CBOR half decodes to
  // the same byte string the other two languages emit; its base64 is the only place
  // in the corpus exercising both non-alphanumeric alphabet characters, which this
  // package's hand-written base64 codec reaches on no other vector.
  ["SourcePackRequest", "SourcePackRequest"],
  ["SourcePackResponse", "SourcePackResponse"],
  ["SourceFileRequest", "SourceFileRequest"],
  ["SourceFileResponse", "SourceFileResponse"],
  // The governance vectors cover the payload shapes reached only by a governance
  // attachment. RenderPolicyInput is the payload a policy attachment receives at
  // the render stage — rendered artifacts instead of the IR graph — carrying one
  // artifact with both optional members (the parsed tree and the declared
  // format) and one that omits both. The two annotation-transform vectors are
  // the wire an annotation-transform attachment receives and returns: the input
  // carries an empty annotation map (pinning that an empty object is emitted
  // rather than dropped), a binding-key list whose entries are NOT in sorted
  // order (pinning that a list is carried in author order rather than sorted),
  // and an opaque settings object with out-of-order keys (pinning the verbatim
  // carry); the output omits the failure member. The annotation map's key sort is
  // pinned here for CBOR but not for JSON — see the dedicated test below.
  // MutatorInputSources is
  // the label-transform payload WITH its pinned sources and settings populated —
  // the mutator vector above leaves both absent, so the file list a governance
  // attachment hands its guest crossed the wire with no parity coverage.
  ["RenderPolicyInput", "RenderPolicyInput"],
  ["AnnotationTransformInput", "AnnotationTransformInput"],
  ["AnnotationTransformOutput", "AnnotationTransformOutput"],
  ["MutatorInputSources", "MutatorInput"],
  // The map-order probe vectors carry one golden pair per map-owning payload, with
  // every map that payload declares populated by a probe key set whose lexicographic
  // order is the exact reverse of its canonical CBOR order. Map-key ordering is a
  // determinism contract — identical input must produce byte-identical output — and
  // a map is the one container with no inherent order, so every encoder sorts before
  // emitting. These vectors are what pins that sort on this side: the fourth contract
  // sources the value from the committed CBOR bytes, so the JSON encoder receives the
  // keys in the reverse of the order it must emit and has to re-sort to match. A
  // constructed-value test cannot stand in for them here, because a TypeScript
  // interface is erased before it runs and the payload types cannot be walked.
  ["MapOrderAnnotationTarget", "AnnotationTarget"],
  ["MapOrderArtifactProjectionEntry", "ArtifactProjectionEntry"],
  ["MapOrderBoundaryEnvOverridesInput", "BoundaryEnvOverridesInput"],
  ["MapOrderBoundaryEnvOverridesOutput", "BoundaryEnvOverridesOutput"],
  ["MapOrderDocumentSet", "DocumentSet"],
  ["MapOrderEmittedEntity", "EmittedEntity"],
  ["MapOrderFRPStepInput", "FRPStepInput"],
  ["MapOrderFRPStepOutput", "FRPStepOutput"],
  ["MapOrderIRFeatureEntry", "IRFeatureEntry"],
  ["MapOrderIRNode", "IRNode"],
  ["MapOrderIRNodeProjection", "IRNodeProjection"],
  ["MapOrderIRSpec", "IRSpec"],
  ["MapOrderLinkEdge", "LinkEdge"],
  ["MapOrderLinkPublishOutput", "LinkPublishOutput"],
  ["MapOrderLinkResolveInput", "LinkResolveInput"],
  ["MapOrderMutatorGraph", "MutatorGraph"],
  ["MapOrderMutatorNode", "MutatorNode"],
  ["MapOrderRenderedNode", "RenderedNode"],
  ["MapOrderSubChart", "SubChart"],
  ["MapOrderWorkloadProjection", "WorkloadProjection"],
  // The failure-envelope vectors pin the error member every socket output type
  // carries — the path that reports "I ran but could not complete", which the host
  // halts the build on. It is the path that carries bad news, so a codec defect
  // there surfaces only once something has already gone wrong and what gets
  // corrupted is the diagnostic itself.
  //
  // Each sets the error message with its success members left at their zero value,
  // because the host abandons the payload the moment it reads a non-empty error:
  // an error beside substantively populated success content is a state nothing
  // ever acts on. Where the schema makes a success member required it stays
  // present but empty, which is what a failing guest must emit to satisfy the
  // contract. The two host-function replies above already carried a failure
  // golden and are not duplicated here.
  ["ErrorEnvelopeAnnotationTransformOutput", "AnnotationTransformOutput"],
  ["ErrorEnvelopeAssembleBoundaryOutput", "AssembleBoundaryOutput"],
  ["ErrorEnvelopeFRPStepOutput", "FRPStepOutput"],
  ["ErrorEnvelopeImportOutput", "ImportOutput"],
  ["ErrorEnvelopeLinkPublishOutput", "LinkPublishOutput"],
  ["ErrorEnvelopeLinkResolveOutput", "LinkResolveOutput"],
  ["ErrorEnvelopeLinterOutput", "LinterOutput"],
  ["ErrorEnvelopeMutatorOutput", "MutatorOutput"],
  ["ErrorEnvelopeNameBoundOutput", "NameBoundOutput"],
  ["ErrorEnvelopeParseOutput", "ParseOutput"],
  ["ErrorEnvelopePolicyOutput", "PolicyOutput"],
  ["ErrorEnvelopeRenderNodeOutput", "RenderNodeOutput"],
  ["ErrorEnvelopeRenderedNameOutput", "RenderedNameOutput"],
  ["ErrorEnvelopeSourceFileResponse", "SourceFileResponse"],
  ["ErrorEnvelopeSourcePackResponse", "SourcePackResponse"],
  ["ErrorEnvelopeValidateNodeOutput", "ValidateNodeOutput"],
];

// The codecs are resolved by NAME because the vector list above is data and a
// TypeScript interface is erased before it runs, so a per-type codec cannot be
// reached through the typed namespace. The namespace is viewed as a bag of unknown
// values rather than as a bag of one-argument functions: it also exports
// multi-argument append helpers this harness never calls, so the narrower shape is
// not true of the module as a whole and describing it that way is rejected. Each
// name is checked to be callable as it is resolved, so a codec that stops being
// exported reports its own name instead of failing as "not a function" inside an
// assertion several lines further on.
const exported = abi as Record<string, unknown>;

function codec(name: string): (arg: unknown) => unknown {
  const fn = exported[name];
  if (typeof fn !== "function") {
    throw new Error(`the ABI module exports no codec named ${name}`);
  }
  return fn as (arg: unknown) => unknown;
}

for (const [file, type] of vectors) {
  test(`${file} JSON+CBOR parity`, () => {
    const wantJson = goldenJson(file);
    const decodeJson = codec(`decode${type}`);
    const encodeJson = codec(`encode${type}`);
    const v = decodeJson(wantJson);
    assert.equal(encodeJson(v), wantJson, `${file} JSON wire`);

    const wantCbor = goldenCbor(file);
    const encodeCbor = codec(`encodeCBOR${type}`);
    const decodeCbor = codec(`decodeCBOR${type}`);
    const enc = encodeCbor(v) as Uint8Array;
    assert.equal(toHex(enc), wantCbor, `${file} CBOR encode`);
    const rt = encodeCbor(decodeCbor(enc)) as Uint8Array;
    assert.equal(toHex(rt), wantCbor, `${file} CBOR round-trip`);

    assert.equal(
      encodeJson(decodeCbor(fromHex(wantCbor))),
      wantJson,
      `${file} JSON wire encoded from the committed CBOR bytes is not the JSON golden`,
    );
  });
}

// The annotation map's JSON key sort is pinned by ENCODING a constructed value,
// the one path on which a lost sort is observable. A value decoded from a golden
// already carries the golden's lexicographic order — JSON.parse preserves the key
// order of the document it reads — so the vector round-trips green whether the
// JSON encoder sorts or not. Constructing the object with its keys INSERTED out
// of order is what makes that sort visible.
//
// The two wire forms order the same three keys differently, and the test asserts
// both. JSON sorts lexicographically. CBOR sorts by the encoded key bytes, whose
// length head leads, so a shorter key always precedes a longer one and the three
// keys emit in the exact reverse of their JSON order. The CBOR half is already
// pinned by the vector above — the golden's JSON carries the lexicographic order,
// so the CBOR encoder has to re-sort actively — and is asserted again here
// because a single constructed value makes both rules legible side by side.
test("AnnotationTarget orders annotation map keys per wire form", () => {
  const v = {
    boundary_path: "prod/edge",
    component_name: "web",
    annotations: { tier: "frontend", "aggrega.ai/owner": "checkout", "cost-center": "42" },
  };

  assert.equal(
    codec("encodeAnnotationTarget")(v),
    '{"boundary_path":"prod/edge","component_name":"web",' +
      '"annotations":{"aggrega.ai/owner":"checkout","cost-center":"42","tier":"frontend"}}',
    "JSON annotation keys not in lexicographic order",
  );

  // Shortest key first: "tier" (4), "cost-center" (11), "aggrega.ai/owner" (16).
  const hex = toHex(codec("encodeCBORAnnotationTarget")(v) as Uint8Array);
  const ordered = ["tier", "cost-center", "aggrega.ai/owner"];
  const at = ordered.map((k) => hex.indexOf(toHex(new TextEncoder().encode(k))));
  for (let i = 0; i < ordered.length; i++) {
    assert.ok(at[i] >= 0, `CBOR wire does not carry annotation key ${ordered[i]}: ${hex}`);
  }
  for (let i = 1; i < at.length; i++) {
    assert.ok(
      at[i - 1] < at[i],
      `CBOR annotation keys not in canonical length-first order ` +
        `(${ordered[i]}@${at[i]} before ${ordered[i - 1]}@${at[i - 1]}): ${hex}`,
    );
  }
});

// The float-boundary suite carries values a JSON number cannot express (NaN,
// ±Inf, −0.0) as the raw f64 bits, so the CBOR CDE §4.2 float rule is proven
// cross-language on exactly those cases. Each entry's five-field CandidateSignals
// must encode to the committed canonical bytes.
function f64FromBits(hexBits: string): number {
  const dv = new DataView(new ArrayBuffer(8));
  for (let i = 0; i < 8; i++) dv.setUint8(i, parseInt(hexBits.substr(i * 2, 2), 16));
  return dv.getFloat64(0);
}

test("float-boundary vectors", () => {
  const raw = readFileSync(join(testdata, "float_vectors.json"), "utf8");
  const vecs = JSON.parse(raw) as Array<{ label: string; bits: string; cbor: string }>;
  const encodeCbor = codec("encodeCBORCandidateSignals");
  const decodeCbor = codec("decodeCBORCandidateSignals");
  for (const fv of vecs) {
    const f = f64FromBits(fv.bits);
    const v = { table: f, category: f, domainLabels: f, shape: f, name: f };
    const enc = encodeCbor(v) as Uint8Array;
    assert.equal(toHex(enc), fv.cbor, `${fv.label} CBOR float`);
    const rt = encodeCbor(decodeCbor(enc)) as Uint8Array;
    assert.equal(toHex(rt), fv.cbor, `${fv.label} float round-trip`);
  }
});

// The vector corpus is ONE golden read by every harness that enumerates it, and the
// value of that arrangement is entirely in the word "one". Codecs generated from a
// single schema are only proven to produce a single wire if all of them are held to
// the same bytes; a vector exercised in one language and not another proves that
// language's codec self-consistent and says nothing about parity.
//
// Nothing structural keeps the harnesses aligned. Each enumerates its vectors in its
// own idiom — the array above, call sites in Go, macro invocations in Rust — so
// agreement across them is a hand-maintained property, and a vector pair committed
// to the shared tree is wired by whoever remembers to wire it everywhere. The
// failure mode is silent in the worst way: the new vector passes wherever it was
// wired, every existing vector keeps passing, and the harness that never received it
// reports nothing at all. Coverage does not go red when it is missing, because
// missing coverage has no assertion to fail.
//
// The census below closes that on this side by deriving coverage FROM THE CORPUS
// rather than from a list: the directory is the authority, so a pair is covered the
// moment it lands or this test names it. Other harnesses carry the same guard against
// the same directory, and NOT ALL OF THEM LIVE IN THIS REPOSITORY — the compiler's
// independent byte-parity oracle carries one too. So a guard added or removed
// elsewhere is invisible from here, which is why this says what the property is
// rather than how many harnesses hold it. Here the comparison is exact in both
// directions, because the vector list is data this test can read directly.

// standaloneFixtures are the corpus files that are deliberately NOT vector pairs,
// each consumed by its own dedicated test in every harness that reads this tree.
// They are named so that a fixture arriving in the corpus and wired nowhere cannot
// pass as one of these.
const standaloneFixtures = ["float_vectors.json"];

// corpusVectors returns the vector names committed to the shared tree, having first
// asserted that every name carries BOTH halves of its pair and that every file
// present is either half of a pair or a named standalone fixture. An unrecognized
// file fails rather than being skipped, because a fixture nothing consumes is the
// same coverage hole this census exists to close.
function corpusVectors(): Set<string> {
  const json = new Set<string>();
  const cbor = new Set<string>();
  for (const name of readdirSync(testdata)) {
    if (standaloneFixtures.includes(name)) continue;
    if (name.endsWith(".golden.json")) json.add(name.slice(0, -".golden.json".length));
    else if (name.endsWith(".cbor.hex")) cbor.add(name.slice(0, -".cbor.hex".length));
    else {
      throw new Error(
        `the shared vector corpus carries ${JSON.stringify(name)}, which is neither half of a vector ` +
          `pair nor a fixture any harness is known to consume, so nothing asserts anything about it`,
      );
    }
  }
  const halfOnly = [...json].filter((n) => !cbor.has(n)).concat([...cbor].filter((n) => !json.has(n)));
  assert.deepEqual(
    halfOnly.sort(),
    [],
    "a vector is the JSON and CBOR pair, and a name carrying one half leaves the other codec unpinned",
  );
  assert.ok(json.size > 0, "found no vector pairs in the shared corpus, so this census certifies nothing");
  return json;
}

test("every golden vector is exercised", () => {
  const corpus = corpusVectors();
  const exercised = new Set(vectors.map(([file]) => file));

  assert.deepEqual(
    [...corpus].filter((name) => !exercised.has(name)).sort(),
    [],
    "these vectors are committed to the shared corpus but no assertion in this harness reads them, so " +
      "this language's codec is unpinned on a wire the others prove",
  );
  assert.deepEqual(
    [...exercised].filter((name) => !corpus.has(name)).sort(),
    [],
    "these vectors are asserted by this harness but the shared corpus does not carry them",
  );
});
