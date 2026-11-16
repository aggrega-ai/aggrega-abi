# aggrega-abi

Portable codec for the [Aggrega](https://aggrega.ai) Wasm plugin ABI.
Three siblings live here, one per supported source language:

- **`go/`** — Go codec, published as the Go module
  `github.com/aggrega-ai/aggrega-abi/go`.
- **`rust/`** — Rust codec, published to crates.io as
  `aggrega-abi`.
- **`typescript/`** — TypeScript codec, published to npm as
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

Contract floor: `v1beta1`. The three codecs export a common version
constant and stay coordinated at every socket-shape change.

## Layout

```text
aggrega-abi/
├── README.md           (this file)
├── .github/workflows/  (the API compatibility check on dev, the codec freeze on main)
├── go/                 (Go codec)
├── rust/               (Rust codec)
├── scripts/            (check-api-compat.sh, the API compatibility check)
└── typescript/         (TypeScript codec)
```

Per-language entry points live in `<language>/README.md`. Start there
for the language you author plugins in.

## Versioning and compatibility

The codecs and the Plugin Development Kits follow semantic versioning
from `0.9.0`, the version they take at Aggrega's first public release.
The three languages here share one version number.

- **Code built against a version keeps building against every later
  version of the same line.** Within `0.9.x` a release may add exported
  functions, types and constants; it never removes or changes one. A
  field added to a Go struct is an addition for struct literals written
  with field names, as Go's own compatibility promise assumes; in Rust
  a public field added to a struct code can build with a struct literal
  is a change, and in TypeScript a required member added to an exported
  interface is a change, which the check's comparison does not see — its
  `--help` names each such limit.
- **A breaking change starts a new line.** Below `1.0.0` it raises the
  minor number (`0.9.x` → `0.10.0`) and a patch release never breaks;
  from `1.0.0` it raises the major number. The codecs move to `1.0.0`
  together with Aggrega's own `1.0.0`.

The promise covers the exported API. The wire shape the codecs encode
is held by the plugin ABI version they export: within one ABI version a
socket gains only descriptions and optional fields.

`scripts/check-api-compat.sh` holds the rule. It compares the exported
API of each codec with the newest release tag and refuses a removed or
changed item unless the version being prepared marks a breaking change
— Go with `apidiff`, Rust with `cargo-semver-checks`, TypeScript by
comparing the declaration files the package emits. A change that has to
break raises the version in `rust/Cargo.toml` and
`typescript/package.json` in the same change. The workflow
`api-compat.yml` runs it on every push to `dev` and every pull request
into it, and the Plugin Development Kits' workflow runs it over the
kits. To run it locally:

```sh
scripts/check-api-compat.sh --tools .api-tools .
```

Its `--help` states the rule, the tools it pins, and what each
language's comparison cannot see.

## License

The Aggrega plugin ABI codecs are released under the Apache License, Version 2.0; the full text is in the
[`LICENSE`](LICENSE) file.
