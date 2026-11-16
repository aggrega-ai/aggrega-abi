#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Cognitio s.r.l.
#
# check-release-version.sh — refuse to publish when a package's own version is not
# the release version. Run with --help for the contract.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/check-release-version.sh VERSION

Confirms that both packages this repository publishes carry VERSION as their own
version: the crate in rust/Cargo.toml and the npm package in
typescript/package.json. VERSION is a final release version, X.Y.Z with no
pre-release or build suffix (a release tag vX.Y.Z without its leading v).

Run from anywhere; paths resolve against the repository root. Needs cargo and
node on PATH.

Exit status:
  0  both packages carry VERSION
  1  a package carries another version (each one is named)
  2  usage error, or VERSION is not X.Y.Z
EOF
}

case "${1:-}" in
  -h | --help) usage; exit 0 ;;
esac
if [ "$#" -ne 1 ]; then
  usage >&2
  exit 2
fi

version="$1"
if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: '$version' is not a final release version X.Y.Z" >&2
  exit 2
fi

cd "$(dirname "$0")/.."

crate_version="$(cargo metadata --no-deps --format-version 1 --manifest-path rust/Cargo.toml |
  node -e 'const m = JSON.parse(require("fs").readFileSync(0, "utf8"));
const p = m.packages.find((p) => p.name === "aggrega-abi");
if (!p) { console.error("error: rust/Cargo.toml declares no aggrega-abi package"); process.exit(1); }
console.log(p.version);')"
npm_version="$(node -p 'require("./typescript/package.json").version')"

status=0
if [ "$crate_version" != "$version" ]; then
  echo "error: rust/Cargo.toml carries version $crate_version, not the release version $version" >&2
  status=1
fi
if [ "$npm_version" != "$version" ]; then
  echo "error: typescript/package.json carries version $npm_version, not the release version $version" >&2
  status=1
fi
if [ "$status" -eq 0 ]; then
  echo "both packages carry the release version $version"
fi
exit "$status"
