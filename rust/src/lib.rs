// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

//! Portable codec for the Aggrega Wasm plugin ABI.
//!
//! The reflect-free socket codec — the encode/decode routines for the
//! typed payloads that cross the host/guest boundary — is generated
//! into this crate from the canonical schemas. The codec is consumable
//! directly by plugin authors working against the raw Extism runtime,
//! and is re-exported by the Rust Plugin Development Kit.
//!
//! Contract floor: `v1alpha1`.
