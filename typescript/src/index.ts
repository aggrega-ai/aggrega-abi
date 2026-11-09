// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

/**
 * @aggrega-ai/aggrega-abi
 *
 * Portable codec for the Aggrega Wasm plugin ABI. The socket codec — the
 * typed payloads that cross the host/guest boundary plus their JSON
 * encode/decode — is generated into this package from the canonical
 * schemas. The encoders write object fields in schema order, sort a plain
 * map's keys, write a map the schema marks order-preserving — a Map — in
 * the order it holds, and carry opaque values verbatim, so the wire is
 * byte-identical to the Go and Rust codecs generated from the same schema.
 * A type holding such a Map decodes its JSON member by member, so a key
 * such as "8080", which JSON.parse would move to the front, keeps its
 * place. The codec is
 * consumable directly by plugin authors working against the raw Extism
 * runtime, and is re-exported by the TypeScript Plugin Development Kit.
 *
 * Contract floor: v1beta1.
 */

export * from "./abi_version_enums_gen.ts";
export * from "./cli_gen.ts";
export * from "./describe_gen.ts";
export * from "./diagnostic_enums_gen.ts";
export * from "./diagnostic_gen.ts";
export * from "./hostfunc_error_enums_gen.ts";
export * from "./frp_gen.ts";
export * from "./mutator_gen.ts";
export * from "./linter_gen.ts";
export * from "./ir_gen.ts";
export * from "./policy_gen.ts";
export * from "./annotationtransform_gen.ts";
export * from "./providerrender_gen.ts";
export * from "./parse_gen.ts";
export * from "./import_gen.ts";
