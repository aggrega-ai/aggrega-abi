// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

// The member-order JSON reader and writer behind a Map<string, V>.
//
// The authored order `d, 8080, bbbb, aaaaaaaa, cc` is neither lexicographic, nor
// length-first, nor the order a plain object gives an array-index key — which is
// the order JSON.parse produces, "8080" first. A reader that went through
// JSON.parse, or a writer that sorted, therefore fails on the first key rather
// than agreeing by coincidence. The first test calibrates that: it shows the
// runtime really does reorder the object, so a green here is not a runtime that
// happens to keep insertion order for every key.

import { test } from "node:test";
import assert from "node:assert/strict";

import { readJSONMembers, readOrderedObject, writeOrderedObject } from "../src/orderedmap_gen.ts";

const authoredProbeKeys = ["d", "8080", "bbbb", "aaaaaaaa", "cc"];
const authoredProbeObject = '{"d":"1","8080":"2","bbbb":"3","aaaaaaaa":"4","cc":"5"}';

const readString = (raw: string): string => JSON.parse(raw) as string;
const writeString = (v: string): string => JSON.stringify(v);

test("the reader keeps an array-index key in place where JSON.parse moves it first", () => {
  assert.deepEqual(Object.keys(JSON.parse(authoredProbeObject)), ["8080", "d", "bbbb", "aaaaaaaa", "cc"]);
  const m = readOrderedObject(authoredProbeObject, readString);
  assert.deepEqual([...m.keys()], authoredProbeKeys);
  assert.deepEqual([...m.values()], ["1", "2", "3", "4", "5"]);
  assert.equal(m.get("8080"), "2");
  assert.equal(m.size, authoredProbeKeys.length);
});

test("decode then encode returns the authored object byte for byte", () => {
  const m = readOrderedObject(authoredProbeObject, readString);
  assert.equal(writeOrderedObject(m, writeString), authoredProbeObject);
});

test("an empty object reads as an empty Map and an empty Map writes as {}", () => {
  for (const doc of ["{}", " { } ", "\n{\t}\r\n"]) {
    const m = readOrderedObject(doc, readString);
    assert.equal(m.size, 0, doc);
  }
  assert.equal(writeOrderedObject(new Map<string, string>(), writeString), "{}");
});

test("a repeated key is refused, naming the key", () => {
  assert.throws(() => readOrderedObject('{"a":"1","b":"2","a":"3"}', readString), /duplicate key "a"/);
  assert.throws(() => readOrderedObject('{"8080":"1","8080":"2"}', readString), /duplicate key "8080"/);
});

test("setting a present key replaces its value and keeps its position", () => {
  const m = readOrderedObject(authoredProbeObject, readString);
  m.set("8080", "x");
  m.set("0", "6");
  assert.deepEqual([...m.keys()], [...authoredProbeKeys, "0"]);
  assert.equal(
    writeOrderedObject(m, writeString),
    '{"d":"1","8080":"x","bbbb":"3","aaaaaaaa":"4","cc":"5","0":"6"}',
  );
});

test("a nested map keeps its order at both levels", () => {
  const doc = '{"zz":' + authoredProbeObject + ',"8080":{},"a":{"y":"1","x":"2"}}';
  const m = readOrderedObject(doc, (raw) => readOrderedObject(raw, readString));
  assert.deepEqual([...m.keys()], ["zz", "8080", "a"]);
  assert.deepEqual([...m.get("zz")!.keys()], authoredProbeKeys);
  assert.equal(m.get("8080")!.size, 0);
  assert.deepEqual([...m.get("a")!.keys()], ["y", "x"]);
  assert.equal(writeOrderedObject(m, (inner) => writeOrderedObject(inner, writeString)), doc);
});

test("a key a plain object treats specially is an ordinary key", () => {
  const m = readOrderedObject('{"__proto__":"x","constructor":"y"}', readString);
  assert.deepEqual([...m.keys()], ["__proto__", "constructor"]);
  assert.equal(m.get("__proto__"), "x");
});

test("members are visited in order, each with the raw text of its value", () => {
  const seen: Array<[string, string]> = [];
  readJSONMembers(' {"s" : "}{\\"" , "a":[1,{"c":"]"}],"n":-1.5e3,"t":true,"z":null,"o":{"k":[]}} ', (key, raw) => {
    seen.push([key, raw]);
  });
  assert.deepEqual(seen, [
    ["s", '"}{\\""'],
    ["a", '[1,{"c":"]"}]'],
    ["n", "-1.5e3"],
    ["t", "true"],
    ["z", "null"],
    ["o", '{"k":[]}'],
  ]);
  const escaped: string[] = [];
  readJSONMembers('{"a\\"b":1,"\\u0041":2}', (key) => {
    escaped.push(key);
  });
  assert.deepEqual(escaped, ['a"b', "A"]);
});

test("text that is not exactly one JSON object is refused", () => {
  for (const doc of [
    "",
    "null",
    "[]",
    '"d"',
    "1",
    "{",
    '{"a"}',
    '{"a":}',
    '{"a":1,}',
    '{"a":1 2}',
    '{"a":1} x',
    '{"a":"unterminated}',
    "{a:1}",
    '{"a":[1,2}',
  ]) {
    assert.throws(() => readJSONMembers(doc, () => {}), Error, JSON.stringify(doc));
  }
});

test("a malformed value is refused when it is decoded", () => {
  assert.throws(() => readOrderedObject('{"a":tru}', readString));
  assert.throws(() => readOrderedObject('{"a":[1,2}', (raw) => JSON.parse(raw) as unknown));
});
