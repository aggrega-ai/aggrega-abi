// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

/**
 * @aggrega-ai/aggrega-abi
 *
 * Portable codec for the Aggrega Wasm plugin ABI. The socket codec — the
 * typed payloads that cross the host/guest boundary plus their JSON
 * encode/decode — is generated into this package from the canonical
 * schemas. The encoders write object fields in schema order, sort map
 * keys, and carry opaque values verbatim, so the wire is byte-identical
 * to the Go and Rust codecs generated from the same schema. The codec is
 * consumable directly by plugin authors working against the raw Extism
 * runtime, and is re-exported by the TypeScript Plugin Development Kit.
 *
 * Contract floor: v1alpha1.
 */

export * from "./cli_gen.ts";
export * from "./describe_gen.ts";
export * from "./frp_gen.ts";
export * from "./mutator_gen.ts";
export * from "./linter_gen.ts";
export * from "./ir_gen.ts";
export * from "./policy_gen.ts";
export * from "./providerrender_gen.ts";
