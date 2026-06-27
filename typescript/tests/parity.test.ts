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
  encodeMutatorInput,
  decodeMutatorInput,
  encodeMutatorOutput,
  decodeMutatorOutput,
  encodeLinterInput,
  decodeLinterInput,
  encodeLinterOutput,
  decodeLinterOutput,
} from "../src/index.ts";

const here = dirname(fileURLToPath(import.meta.url));

function golden(name: string): string {
  return readFileSync(join(here, "..", "testdata", name), "utf8").trim();
}

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
