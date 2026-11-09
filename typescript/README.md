# aggrega-abi (TypeScript)

TypeScript codec for the Aggrega Wasm plugin ABI.

Package name: `@aggrega-ai/aggrega-abi`.

The package encodes and decodes the Wasm socket payloads to and from
the JSON wire form. It is reflect-free and generated from the canonical
schemas, so the wire shape stays byte-stable.

Depend on this package directly when authoring a plugin against the raw
Extism runtime. The TypeScript Plugin Development Kit
(`@aggrega-ai/aggrega-pdk-typescript`) depends on this package and
re-exports it, adding the Extism plumbing and the Aggrega
host-function bindings.

Contract floor: `v1beta1`.

This package follows semantic versioning from `0.9.0`: code built
against a version keeps building against every later `0.9.x` release,
and a breaking change ships only as a new minor version below `1.0.0`
(`0.10.0`), or a new major version from `1.0.0`. The
[repository overview](../README.md#versioning-and-compatibility) states
the rule and the check that holds it.
