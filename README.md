# aggrega-abi

Portable codec for the [Aggrega](https://aggrega.ai) Wasm plugin ABI.
Three siblings live here, one per supported source language:

- **`go/`** — Go codec; targets publication to `pkg.go.dev` as
  `github.com/aggrega-ai/aggrega-abi-go`.
- **`rust/`** — Rust codec; targets publication to `crates.io` as
  `aggrega-abi`.
- **`typescript/`** — TypeScript codec; targets publication to npm as
  `@aggrega-ai/aggrega-abi`.

Each codec encodes and decodes the Wasm socket payloads — the typed
documents that cross the host/guest boundary — to and from both wire
forms, JSON text and canonical CBOR, whichever a socket carries. The
codec is reflect-free and generated from the canonical schemas, so the
wire shape stays byte-stable across every language.

The codec is consumable on its own: a plugin author working directly
against the raw Extism runtime depends on `aggrega-abi` alone. The
language Plugin Development Kits depend on `aggrega-abi` and re-export
it, adding the Extism input/output plumbing and the Aggrega
host-function bindings on top.

Contract floor: `v1alpha1`. The three codecs export a common version
constant and stay coordinated at every socket-shape change.

## Layout

```text
aggrega-abi/
├── README.md           (this file)
├── go/                 (Go codec)
├── rust/               (Rust codec)
└── typescript/         (TypeScript codec)
```

Per-language entry points live in `<language>/README.md`. Start there
for the language you author plugins in.

## License

The Aggrega plugin ABI codecs are released under the Apache License, Version 2.0; the full text is in the
[`LICENSE`](LICENSE) file.
