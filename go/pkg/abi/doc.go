// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

// Package abi is the portable Aggrega Wasm ABI codec.
//
// The reflect-free socket codec — the encode/decode routines for the
// typed payloads that cross the host/guest boundary — is generated
// into this package from the canonical schemas. Each type carries a
// streaming token-walk Marshal/Unmarshal pair: the generated code walks
// the JSON object and its composite containers with the encoding/json
// tokenizer rather than reflective struct-walking, so it references no
// reflect symbol by construction. A guest decodes the ABI through it
// without the generic-dispatch reflection a reflective decode drags in.
// The codec is consumable directly by plugin authors working against the
// raw Extism runtime, and is re-exported by the language Plugin
// Development Kits.
//
// The codec uses a neutral, cycle-free type vocabulary: an opaque
// cross-boundary value (a render-time context, a diagnostic) carries
// through as a verbatim json.RawMessage, and a map the schema marks
// order-preserving is an OrderedMap. A plain map carries no order of its
// own, so both encodings write its keys sorted. An OrderedMap's order is
// part of what it carries, so both encodings write it in the order it
// holds: the text encoding as a JSON object, the binary encoding as a list
// of key and value entries. A guest therefore reads such a map in the
// order the host wrote it and returns one in the order it set it, and a
// decoder refuses an encoded OrderedMap that repeats a key. The wire bytes
// are identical to the host's.
//
// Contract floor: v1beta1.
package abi
