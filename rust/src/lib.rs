// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

//! Portable codec for the Aggrega Wasm plugin ABI.
//!
//! The socket codec — the typed payloads that cross the host/guest
//! boundary plus their JSON serialization — is generated into this crate
//! from the canonical schemas. serde_json is the wire codec; the
//! generated types are shaped (BTreeMap for sorted map keys, RawValue for
//! verbatim opaque carry) so the wire is byte-identical to the Go and
//! TypeScript codecs generated from the same schema. The codec is
//! consumable directly by plugin authors working against the raw Extism
//! runtime, and is re-exported by the Rust Plugin Development Kit.
//!
//! Contract floor: `v1alpha1`.

mod cli_gen;
mod describe_gen;
mod frp_gen;
mod hostfunc_error_enums_gen;
mod import_gen;
mod ir_gen;
mod linter_gen;
mod mutator_gen;
mod parse_gen;
mod policy_gen;
mod providerrender_gen;

pub use cli_gen::*;
pub use describe_gen::*;
pub use frp_gen::*;
pub use hostfunc_error_enums_gen::*;
pub use import_gen::*;
pub use ir_gen::*;
pub use linter_gen::*;
pub use mutator_gen::*;
pub use parse_gen::*;
pub use policy_gen::*;
pub use providerrender_gen::*;
