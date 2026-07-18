// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

// Cross-language wire-parity tests for the generated socket codecs.
//
// The goldens under testdata/ are the canonical wire bytes — the same goldens
// the Go and Rust codecs validate against, so a byte-match here is a
// cross-language parity assertion: the three codecs are generated from one
// schema and produce one wire. Each golden carries the determinism-sensitive
// cases (sorted map keys, verbatim opaque carry, omitted empty fields), so a
// codec regression breaks the byte-compare loudly.
//
// The contract proven per payload: decode the golden through the generated
// codec, re-encode, and assert byte-identical output.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import {
  encodeCLIRunInput,
  decodeCLIRunInput,
  encodeFRPStepInput,
  decodeFRPStepInput,
  encodeFRPStepOutput,
  decodeFRPStepOutput,
  encodeMutatorInput,
  decodeMutatorInput,
  encodeMutatorOutput,
  decodeMutatorOutput,
  encodeLinterInput,
  decodeLinterInput,
  encodeLinterOutput,
  decodeLinterOutput,
  encodeIRGraph,
  decodeIRGraph,
  encodePolicyInput,
  decodePolicyInput,
  encodePolicyOutput,
  decodePolicyOutput,
  encodeRenderNodeInput,
  decodeRenderNodeInput,
  encodeRenderNodeOutput,
  decodeRenderNodeOutput,
  encodeAssembleBoundaryInput,
  decodeAssembleBoundaryInput,
  encodeAssembleBoundaryOutput,
  decodeAssembleBoundaryOutput,
  encodeLinkPublishInput,
  decodeLinkPublishInput,
  encodeLinkPublishOutput,
  decodeLinkPublishOutput,
  encodeLinkResolveInput,
  decodeLinkResolveInput,
  encodeLinkResolveOutput,
  decodeLinkResolveOutput,
  encodePluginDescriptor,
  decodePluginDescriptor,
  encodeParseInput,
  decodeParseInput,
  encodeParseOutput,
  decodeParseOutput,
} from "../src/index.ts";

const here = dirname(fileURLToPath(import.meta.url));

function golden(name: string): string {
  return readFileSync(join(here, "..", "testdata", name), "utf8").trim();
}

test("CLIRunInput wire parity", () => {
  const want = golden("CLIRunInput.golden.json");
  assert.equal(encodeCLIRunInput(decodeCLIRunInput(want)), want);
});

test("FRPStepInput wire parity", () => {
  const want = golden("FRPStepInput.golden.json");
  assert.equal(encodeFRPStepInput(decodeFRPStepInput(want)), want);
});

test("FRPStepOutput wire parity", () => {
  const want = golden("FRPStepOutput.golden.json");
  assert.equal(encodeFRPStepOutput(decodeFRPStepOutput(want)), want);
});

test("MutatorInput wire parity", () => {
  const want = golden("MutatorInput.golden.json");
  assert.equal(encodeMutatorInput(decodeMutatorInput(want)), want);
});

test("MutatorOutput wire parity", () => {
  const want = golden("MutatorOutput.golden.json");
  assert.equal(encodeMutatorOutput(decodeMutatorOutput(want)), want);
});

test("LinterInput wire parity", () => {
  const want = golden("LinterInput.golden.json");
  assert.equal(encodeLinterInput(decodeLinterInput(want)), want);
});

test("LinterOutput wire parity", () => {
  const want = golden("LinterOutput.golden.json");
  assert.equal(encodeLinterOutput(decodeLinterOutput(want)), want);
});

test("IRGraph wire parity", () => {
  const want = golden("IRGraph.golden.json");
  assert.equal(encodeIRGraph(decodeIRGraph(want)), want);
});

test("PolicyInput wire parity", () => {
  const want = golden("PolicyInput.golden.json");
  assert.equal(encodePolicyInput(decodePolicyInput(want)), want);
});

test("PolicyOutput wire parity", () => {
  const want = golden("PolicyOutput.golden.json");
  assert.equal(encodePolicyOutput(decodePolicyOutput(want)), want);
});

test("RenderNodeInput wire parity", () => {
  const want = golden("RenderNodeInput.golden.json");
  assert.equal(encodeRenderNodeInput(decodeRenderNodeInput(want)), want);
});

test("RenderNodeOutput wire parity", () => {
  const want = golden("RenderNodeOutput.golden.json");
  assert.equal(encodeRenderNodeOutput(decodeRenderNodeOutput(want)), want);
});

test("AssembleBoundaryInput wire parity", () => {
  const want = golden("AssembleBoundaryInput.golden.json");
  assert.equal(encodeAssembleBoundaryInput(decodeAssembleBoundaryInput(want)), want);
});

test("AssembleBoundaryOutput wire parity", () => {
  const want = golden("AssembleBoundaryOutput.golden.json");
  assert.equal(encodeAssembleBoundaryOutput(decodeAssembleBoundaryOutput(want)), want);
});

test("LinkPublishInput wire parity", () => {
  const want = golden("LinkPublishInput.golden.json");
  assert.equal(encodeLinkPublishInput(decodeLinkPublishInput(want)), want);
});

test("LinkPublishOutput wire parity", () => {
  const want = golden("LinkPublishOutput.golden.json");
  assert.equal(encodeLinkPublishOutput(decodeLinkPublishOutput(want)), want);
});

test("LinkResolveInput wire parity", () => {
  const want = golden("LinkResolveInput.golden.json");
  assert.equal(encodeLinkResolveInput(decodeLinkResolveInput(want)), want);
});

test("LinkResolveOutput wire parity", () => {
  const want = golden("LinkResolveOutput.golden.json");
  assert.equal(encodeLinkResolveOutput(decodeLinkResolveOutput(want)), want);
});

test("PluginDescriptor CLI wire parity", () => {
  const want = golden("PluginDescriptorCLI.golden.json");
  assert.equal(encodePluginDescriptor(decodePluginDescriptor(want)), want);
});

test("PluginDescriptor provider wire parity", () => {
  const want = golden("PluginDescriptorProvider.golden.json");
  assert.equal(encodePluginDescriptor(decodePluginDescriptor(want)), want);
});

// A descriptor declaring none of the optional nested objects: the codec must omit
// every absent optional rather than emit an empty placeholder.
test("PluginDescriptor minimal wire parity", () => {
  const want = golden("PluginDescriptorMinimal.golden.json");
  assert.equal(encodePluginDescriptor(decodePluginDescriptor(want)), want);
});

// The wire a real CLI plugin (moo) produces: no contract, metadata present.
test("PluginDescriptor moo wire parity", () => {
  const want = golden("PluginDescriptorMoo.golden.json");
  assert.equal(encodePluginDescriptor(decodePluginDescriptor(want)), want);
});

test("ParseInput wire parity", () => {
  const want = golden("ParseInput.golden.json");
  assert.equal(encodeParseInput(decodeParseInput(want)), want);
});

test("ParseOutput wire parity", () => {
  const want = golden("ParseOutput.golden.json");
  assert.equal(encodeParseOutput(decodeParseOutput(want)), want);
});

// A parser output that omits every optional (no diagnostics, no spans) and
// carries a top-level-array tree: the codec must omit absent optionals and
// carry a non-object tree verbatim.
test("ParseOutput bare wire parity", () => {
  const want = golden("ParseOutputBare.golden.json");
  assert.equal(encodeParseOutput(decodeParseOutput(want)), want);
});
