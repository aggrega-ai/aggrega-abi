# aggrega-abi-go

Go codec for the Aggrega Wasm plugin ABI.

Import path: `github.com/aggrega-ai/aggrega-abi-go/pkg/abi`.

The package encodes and decodes the Wasm socket payloads to and from
the JSON wire form. It is reflect-free and generated from the canonical
schemas, so the wire shape stays byte-stable.

Depend on this module directly when authoring a plugin against the raw
Extism runtime. The Go Plugin Development Kit (`aggrega-pdk-go`)
depends on this module and re-exports it, adding the Extism plumbing
and the Aggrega host-function bindings.

Contract floor: `v1alpha1`.
