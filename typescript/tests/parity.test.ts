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
