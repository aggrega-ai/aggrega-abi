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
// Two contracts are proven per vector: the JSON wire round-trips byte-identical
// through the generated codec, and the generated CBOR encoder reproduces the
// committed canonical bytes and decodes back to a byte-stable value.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import * as abi from "../src/index.ts";

const here = dirname(fileURLToPath(import.meta.url));
const testdata = join(here, "..", "..", "testdata");

function goldenJson(name: string): string {
  return readFileSync(join(testdata, `${name}.golden.json`), "utf8").trim();
}

function goldenCbor(name: string): string {
  return readFileSync(join(testdata, `${name}.cbor.hex`), "utf8").trim();
}

function toHex(u8: Uint8Array): string {
  let s = "";
  for (const b of u8) s += b.toString(16).padStart(2, "0");
  return s;
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
  ["LinterOutput", "LinterOutput"],
  ["IRGraph", "IRGraph"],
  ["PolicyInput", "PolicyInput"],
  ["PolicyOutput", "PolicyOutput"],
  ["RenderNodeInput", "RenderNodeInput"],
  ["RenderNodeOutput", "RenderNodeOutput"],
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
  ["FetchExternalInputResponse", "FetchExternalInputResponse"],
  ["FetchExternalInputResponseContent", "FetchExternalInputResponse"],
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
];

const m = abi as Record<string, (arg: unknown) => unknown>;

for (const [file, type] of vectors) {
  test(`${file} JSON+CBOR parity`, () => {
    const wantJson = goldenJson(file);
    const decodeJson = m[`decode${type}`];
    const encodeJson = m[`encode${type}`];
    const v = decodeJson(wantJson);
    assert.equal(encodeJson(v), wantJson, `${file} JSON wire`);

    const wantCbor = goldenCbor(file);
    const encodeCbor = m[`encodeCBOR${type}`];
    const decodeCbor = m[`decodeCBOR${type}`];
    const enc = encodeCbor(v) as Uint8Array;
    assert.equal(toHex(enc), wantCbor, `${file} CBOR encode`);
    const rt = encodeCbor(decodeCbor(enc)) as Uint8Array;
    assert.equal(toHex(rt), wantCbor, `${file} CBOR round-trip`);
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
    m["encodeAnnotationTarget"](v),
    '{"boundary_path":"prod/edge","component_name":"web",' +
      '"annotations":{"aggrega.ai/owner":"checkout","cost-center":"42","tier":"frontend"}}',
    "JSON annotation keys not in lexicographic order",
  );

  // Shortest key first: "tier" (4), "cost-center" (11), "aggrega.ai/owner" (16).
  const hex = toHex(m["encodeCBORAnnotationTarget"](v) as Uint8Array);
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
  const encodeCbor = m["encodeCBORCandidateSignals"];
  const decodeCbor = m["decodeCBORCandidateSignals"];
  for (const fv of vecs) {
    const f = f64FromBits(fv.bits);
    const v = { table: f, category: f, domainLabels: f, shape: f, name: f };
    const enc = encodeCbor(v) as Uint8Array;
    assert.equal(toHex(enc), fv.cbor, `${fv.label} CBOR float`);
    const rt = encodeCbor(decodeCbor(enc)) as Uint8Array;
    assert.equal(toHex(rt), fv.cbor, `${fv.label} float round-trip`);
  }
});
