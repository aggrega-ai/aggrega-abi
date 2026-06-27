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
// through as a verbatim json.RawMessage, and an order-preserving host
// container surfaces as a plain map. The wire bytes are identical to the
// host's; the container's key order is the emitting host's concern, and
// a guest that selects by content reads a map.
//
// Contract floor: v1alpha1.
package abi
