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
// stay green. And it is the only one that can observe a plain map's key sort being
// lost from the JSON encoder: the two wire forms order a plain map's keys by
// different rules — JSON lexicographically, CBOR by the encoded key bytes, whose
// length head leads — so a value arriving from the CBOR wire presents its keys in
// an order the JSON wire must actively re-sort. A value decoded from the JSON
// golden instead arrives already in JSON order, which is why the first contract
// stays green whether the encoder sorts or not. A map the schema marks
// order-preserving is a Map here, read in member order and written in the order it
// holds on both wires, so for it the same contract proves the order survives the
// binary form.

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
  // where dropping either moves the bytes. The Component map is order-preserving
  // and holds "agent" before "db", which is not the length-first order a sorted
  // binary map would take, and each document keeps its keys out of key order; this
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
  // A source render input carrying POPULATED omit entries, one with a namespace
  // and one without, listed out of kind and name order, and the output carrying
  // one count per entry, a non-zero count before a zero. Both lists are omitted
  // when empty, so these are the vectors where dropping either moves the bytes,
  // and a codec that sorted a list, or dropped a zero count, fails them.
  ["RenderNodeInputOmits", "RenderNodeInput"],
  ["RenderNodeOutputOmitted", "RenderNodeOutput"],
  // A source render input whose first omit entry carries the sensitivity mark and
  // whose second does not. The mark is omitted when false, so this is the vector
  // where dropping a true mark, or writing a false one, moves the bytes.
  ["RenderNodeInputOmitsSensitive", "RenderNodeInput"],
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
  // The populated payloads of the render family's remaining exports, each with its
  // members populated: validate_node_input's diagnostics, carve's and reassemble's
  // file bytes and units — a structured unit whose tree keeps its keys out of key
  // order, and an opaque unit carrying none — name_bound's proposal and verdicts,
  // one carrying a bound and one not, rendered_name's identities and answer, and
  // boundary_env_overrides' edges, its author-ordered infra and its per-consumer
  // overrides.
  ["ValidateNodeOutput", "ValidateNodeOutput"],
  ["CarveInput", "CarveInput"],
  ["CarveOutput", "CarveOutput"],
  ["ReassembleInput", "ReassembleInput"],
  ["ReassembleOutput", "ReassembleOutput"],
  ["NameBoundInput", "NameBoundInput"],
  ["NameBoundOutput", "NameBoundOutput"],
  ["RenderedNameInput", "RenderedNameInput"],
  ["RenderedNameOutput", "RenderedNameOutput"],
  ["BoundaryEnvOverridesInput", "BoundaryEnvOverridesInput"],
  ["BoundaryEnvOverridesOutput", "BoundaryEnvOverridesOutput"],
  ["PluginDescriptorCLI", "PluginDescriptor"],
  ["PluginDescriptorProvider", "PluginDescriptor"],
  // A provider contract claiming to honour omit entries: the claim is omitted
  // when false, so this is the descriptor vector where dropping it moves the bytes.
  ["PluginDescriptorProviderOmits", "PluginDescriptor"],
  // An importer's recognition contract populated in every field: omitted when
  // absent, so this is the descriptor vector where dropping it, or reordering a
  // list it carries, moves the bytes.
  ["PluginDescriptorImporter", "PluginDescriptor"],
  ["PluginDescriptorMinimal", "PluginDescriptor"],
  ["PluginDescriptorMoo", "PluginDescriptor"],
  ["ParseInput", "ParseInput"],
  // A parse input whose source bytes are not valid UTF-8 — a bare 0xff 0xfe pair,
  // a NUL, and a lead byte followed by a non-continuation byte — and which omits
  // the options document. This package holds the bytes as a base64 string, so the
  // vector pins that the binary half writes them as the raw byte string the other
  // two languages emit, and reads them back to the same base64.
  ["ParseInputNonUTF8Source", "ParseInput"],
  ["ParseOutput", "ParseOutput"],
  ["ParseOutputBare", "ParseOutput"],
  ["ImportInput", "ImportInput"],
  // An import input carrying a POPULATED catalog view: the field is omitted when
  // absent, so this is the vector where dropping it moves the bytes. One
  // Component declares sockets and one declares none, the sockets are listed out
  // of name order, and the label keys sort differently in the two wire forms, so
  // a codec that sorted a list or lost a map-key sort fails it.
  ["ImportInputCatalog", "ImportInput"],
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
  // The fetched bytes ride a byte field: this package holds them as a base64
  // string, so the vector pins that the binary half writes them as the raw byte
  // string the other two languages emit, and reads them back to the same base64.
  ["FetchExternalInputResponseContent", "FetchExternalInputResponse"],
  // A zero-length fetched payload: the content field is omitted when empty, so
  // the reply carries no field at all, and it stays distinguishable from a failure
  // only because a failure always carries its error fields.
  ["FetchExternalInputResponseEmptyContent", "FetchExternalInputResponse"],
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
  // carry); the output omits the failure member. The annotation map is
  // order-preserving, so neither wire sorts it — see the dedicated test below.
  // MutatorInputSources is
  // the label-transform payload WITH its pinned sources and settings populated —
  // the mutator vector above leaves both absent, so the file list a governance
  // attachment hands its guest crossed the wire with no parity coverage.
  ["RenderPolicyInput", "RenderPolicyInput"],
  ["AnnotationTransformInput", "AnnotationTransformInput"],
  ["AnnotationTransformOutput", "AnnotationTransformOutput"],
  ["MutatorInputSources", "MutatorInput"],
  // The map-order probe vectors carry one golden pair per map-owning payload, with
  // every map that payload declares populated by a probe key set. Map-key ordering
  // is a determinism contract — identical input must produce byte-identical output —
  // and it reads differently for the two kinds of map. A plain map carries no order
  // of its own, so every encoder sorts it, and its probe keys' lexicographic order is
  // the exact reverse of their canonical CBOR order: the fourth contract sources the
  // value from the committed CBOR bytes, so the JSON encoder receives the keys in
  // the reverse of the order it must emit and has to re-sort to match. A map the
  // schema marks order-preserving carries its order as content, so every encoder
  // writes it in the order it holds, and its probe order is neither sorted order and
  // holds "8080", a key a plain object would move to the front — so a decoder going
  // through JSON.parse, or an encoder sorting in either wire's rule, fails it. A
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
  ["MapOrderLinterInput", "LinterInput"],
  ["MapOrderMutatorGraph", "MutatorGraph"],
  ["MapOrderMutatorNode", "MutatorNode"],
  // The policy payload declares no map of its own; it carries the graph, whose nodes
  // and a node's labels are order-preserving maps, held here in a populated payload.
  ["MapOrderPolicyInput", "PolicyInput"],
  ["MapOrderRenderNodeOutput", "RenderNodeOutput"],
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

// The annotation map's order is pinned by ENCODING a constructed value whose order
// is neither of the two sorted orders: not the lexicographic one a JSON map sort
// would give, not the length-first one a CBOR map sort would give, and with a key
// that is an array index, which a plain object would move to the front. The map is
// order-preserving, so both wire forms carry the order it holds — the JSON form as
// an object written in that order, the binary form as an array of entries in that
// order — and the test asserts each, literally. Decoding each wire back is asserted
// too, because a decoder going through JSON.parse would hoist "8080" on the way in.
test("AnnotationTarget keeps the annotation order on both wire forms", () => {
  const ordered = ["tier", "aggrega.ai/owner", "8080", "cost-center"];
  const values: Record<string, string> = { tier: "frontend", "aggrega.ai/owner": "checkout", "8080": "port", "cost-center": "42" };
  const annotations = new Map<string, string>();
  for (const k of ordered) annotations.set(k, values[k]);
  const v = { boundary_path: "prod/edge", component_name: "web", annotations };

  const json = codec("encodeAnnotationTarget")(v) as string;
  assert.equal(
    json,
    '{"boundary_path":"prod/edge","component_name":"web",' +
      '"annotations":{"tier":"frontend","aggrega.ai/owner":"checkout","8080":"port","cost-center":"42"}}',
    "JSON annotation keys not in the order the map holds",
  );
  const fromJson = codec("decodeAnnotationTarget")(json) as { annotations: Map<string, string> };
  assert.deepEqual([...fromJson.annotations.keys()], ordered, "the JSON decode did not keep the order");

  const cbor = codec("encodeCBORAnnotationTarget")(v) as Uint8Array;
  const hex = toHex(cbor);
  const at = ordered.map((k) => hex.indexOf(toHex(new TextEncoder().encode(k))));
  for (let i = 0; i < ordered.length; i++) {
    assert.ok(at[i] >= 0, `CBOR wire does not carry annotation key ${ordered[i]}: ${hex}`);
  }
  for (let i = 1; i < at.length; i++) {
    assert.ok(
      at[i - 1] < at[i],
      `CBOR annotation entries not in the order the map holds ` +
        `(${ordered[i]}@${at[i]} before ${ordered[i - 1]}@${at[i - 1]}): ${hex}`,
    );
  }
  const fromCbor = codec("decodeCBORAnnotationTarget")(cbor) as { annotations: Map<string, string> };
  assert.deepEqual([...fromCbor.annotations.keys()], ordered, "the binary decode did not keep the order");
});

// A non-required order-preserving field is omitted when it is empty and emitted
// when it is populated, on both wires. The two fixtures are asserted together
// because each hides the other's failure: an encoder emitting the key
// unconditionally passes the populated one, and one dropping it unconditionally
// passes the empty one. Each form's binary bytes are read back too, because the
// map head declares the pair count and a head counting an omitted key fails there.
test("an empty order-preserving field is omitted and a populated one is emitted", () => {
  const encodeJson = codec("encodeRenderNodeOutput");
  const encodeCbor = codec("encodeCBORRenderNodeOutput");
  const decodeCbor = codec("decodeCBORRenderNodeOutput");
  const derived = new TextEncoder().encode("derived_keys");
  const carries = (b: Uint8Array): boolean => toHex(b).includes(toHex(derived));

  for (const empty of [{ files: [] }, { files: [], derived_keys: new Map<string, string[]>() }]) {
    const json = encodeJson(empty) as string;
    assert.ok(!json.includes("derived_keys"), `an empty order-preserving field is on the JSON wire: ${json}`);
    const cbor = encodeCbor(empty) as Uint8Array;
    assert.ok(!carries(cbor), `an empty order-preserving field is on the binary wire: ${toHex(cbor)}`);
    decodeCbor(cbor);
  }

  const populated = { files: [], derived_keys: new Map([["resource-names", ["web"]]]) };
  const json = encodeJson(populated) as string;
  assert.ok(
    json.includes('"derived_keys":{"resource-names":["web"]}'),
    `a populated order-preserving field is missing from the JSON wire: ${json}`,
  );
  const cbor = encodeCbor(populated) as Uint8Array;
  assert.ok(carries(cbor), `a populated order-preserving field is missing from the binary wire: ${toHex(cbor)}`);
  const back = decodeCbor(cbor) as { derived_keys: Map<string, string[]> };
  assert.deepEqual([...back.derived_keys], [["resource-names", ["web"]]], "the populated field did not survive the binary form");
});

// A refused vector is canonical CBOR every decoder of its shape must REFUSE: a
// document no encoder of the shape can write, malformed in exactly one way. A
// vector pair holds the decoder to what it must read; a refused vector holds it to
// what it must not, and the two cannot stand in for each other — a decoder that
// kept one of two repeated keys would read every well-formed vector correctly.
// These pin the order-preserving map's array form; the same vectors bind the Go and
// Rust decoders, and the compiler's independent byte-parity oracle confirms each is
// well-formed CBOR whose only fault is the one its name states.
const refusedSuffix = ".refused-cbor.hex";

// The refused vectors, each with the decoder that must refuse it and a fragment
// the refusal must carry.
const refusedVectors: Array<[string, string, string]> = [
  ["MapOrderMutatorNodeRepeatedKey", "MutatorNode", 'duplicate key "a"'],
  ["MapOrderMutatorNodeEntryArity", "MutatorNode", "holds 3 items, want 2"],
  ["MapOrderMutatorNodeEntryKeyNotText", "MutatorNode", "expected text string"],
];

function refusedCbor(name: string): Uint8Array {
  const raw = readFileSync(join(testdata, `${name}${refusedSuffix}`), "utf8");
  if (!raw.endsWith("\n") || /[^0-9a-f]/.test(raw.slice(0, -1))) {
    throw new Error(`refused vector ${name} is not lowercase hex followed by one newline: ${JSON.stringify(raw)}`);
  }
  return fromHex(raw.slice(0, -1));
}

for (const [file, type, want] of refusedVectors) {
  test(`${file} is refused`, () => {
    const decodeCbor = codec(`decodeCBOR${type}`);
    assert.throws(
      () => decodeCbor(refusedCbor(file)),
      (err: unknown) => err instanceof Error && err.message.includes(want),
      `${file} must be refused with an error carrying ${JSON.stringify(want)}`,
    );
  });
}

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
    // A refused vector is a single file rather than a pair, held to its own census.
    if (name.endsWith(refusedSuffix)) continue;
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

test("every refused vector is asserted", () => {
  const refused = readdirSync(testdata)
    .filter((name) => name.endsWith(refusedSuffix))
    .map((name) => name.slice(0, -refusedSuffix.length));
  assert.ok(refused.length > 0, `found no ${refusedSuffix} vectors in the shared corpus, so this census certifies nothing`);
  const asserted = new Set(refusedVectors.map(([file]) => file));
  assert.deepEqual(
    refused.filter((name) => !asserted.has(name)).sort(),
    [],
    "these refused vectors are committed to the shared corpus but no assertion in this harness reads them, so " +
      "this language's decoder is not held to refusing them",
  );
  assert.deepEqual(
    [...asserted].filter((name) => !refused.includes(name)).sort(),
    [],
    "these refused vectors are asserted by this harness but the shared corpus does not carry them",
  );
});
