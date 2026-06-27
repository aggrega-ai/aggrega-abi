# aggrega-abi (Rust)

Rust codec for the Aggrega Wasm plugin ABI.

Crate name: `aggrega-abi`.

The crate encodes and decodes the Wasm socket payloads to and from the
JSON wire form. It is reflect-free and generated from the canonical
schemas, so the wire shape stays byte-stable.

Depend on this crate directly when authoring a plugin against the raw
Extism runtime. The Rust Plugin Development Kit (`aggrega-pdk-rust`)
depends on this crate and re-exports it, adding the Extism plumbing and
the Aggrega host-function bindings.

Contract floor: `v1alpha1`.
