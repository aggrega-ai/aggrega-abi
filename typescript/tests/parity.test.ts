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
  encodePolicyValidatorInput,
  decodePolicyValidatorInput,
  encodePolicyValidatorOutput,
  decodePolicyValidatorOutput,
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

test("PolicyValidatorInput wire parity", () => {
  const want = golden("PolicyValidatorInput.golden.json");
  assert.equal(encodePolicyValidatorInput(decodePolicyValidatorInput(want)), want);
});

test("PolicyValidatorOutput wire parity", () => {
  const want = golden("PolicyValidatorOutput.golden.json");
  assert.equal(encodePolicyValidatorOutput(decodePolicyValidatorOutput(want)), want);
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
