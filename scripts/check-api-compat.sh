#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Cognitio S.r.l.
#
# check-api-compat.sh — refuse a breaking change to a kit's exported API that its
# version does not announce. Run with --help for the contract.

set -uo pipefail

# ---------------------------------------------------------------------------
# Pins. Every tool the comparison runs is pinned here and nowhere else; the
# workflows install them through --tools, so a pin moves in one place.
# ---------------------------------------------------------------------------

# golang.org/x/exp/cmd/apidiff (BSD-3-Clause).
APIDIFF_MODULE_VERSION="v0.0.0-20261109185158-a52e3aef5920"
# cargo-semver-checks (Apache-2.0 OR MIT), the upstream release archives and the
# sha256 digests the release publishes for them.
CSC_VERSION="v0.51.0"
csc_sha256() {
  case "$1" in
    x86_64-unknown-linux-gnu) echo "cacadafaa5cd27ae6d05a74a103266917ebab4c5e43c898360e177d4fbf06a65" ;;
    aarch64-unknown-linux-gnu) echo "58531fa492d237c71b97d113a6d2a36209d96105956569cd0406aa66226ac928" ;;
    aarch64-apple-darwin) echo "3cec5da41eee9a3336d75ad43d1f30a5c9680818f4ba209024786f452cabab5b" ;;
    x86_64-apple-darwin) echo "542f9518fd5360f46c1027a6ac2c9e9ee7431518c42495346df84922a050ce5b" ;;
  esac
}
# The toolchain that generates the rustdoc cargo-semver-checks reads. It is newer
# than the kits' own toolchain floor because the pinned cargo-semver-checks reads
# only the rustdoc format of rustc 1.93 and later; the floor still builds the kits.
RUST_API_TOOLCHAIN="1.98.1"

# The first version of the public version line. A release before it made no
# compatibility promise, so it is not a baseline.
FIRST_VERSION="0.9.0"

PROG="check-api-compat"

usage() {
  cat <<'EOF'
usage: check-api-compat.sh [--sibling DIR] [--tools DIR] [--work DIR]
                           [--languages LIST] REPO

Compares the exported API of the kits in REPO — its go/, rust/ and typescript/
directories — with the API at REPO's newest release tag, and refuses a removed or
changed exported item unless the version being prepared marks a breaking change.
An added item passes.

THE VERSION RULE
  Each repository carries one version for all its languages. The version being
  prepared is read from rust/Cargo.toml and typescript/package.json, which must
  agree; Go carries no version in a manifest and takes the repository's. The
  baseline is REPO's newest tag named vX.Y.Z.
  Below 1.0.0 a breaking change raises the minor version (0.9.x -> 0.10.0) and a
  patch never breaks; from 1.0.0 a breaking change raises the major version.
  Manifests behind the baseline tag (a release tagged without raising them) are
  read as the tag's own version, which marks no breaking change.
  The rule judges the version the manifests prepare. A Go module's version is the
  tag a release cuts, which nothing here compares with the manifests.

WHAT IS COMPARED, PER LANGUAGE
  go          every package of the module in go/ outside internal/, with apidiff,
              on the host platform and on wasip1/wasm (the plugin target).
  rust        the package in rust/Cargo.toml, with cargo-semver-checks.
  typescript  the declaration files the package emits (tsc, comments removed),
              compared line by line: a removed or changed line is a breaking
              change, an added line is not. Bounds of that comparison: a
              declaration moved within or across files reads as removed, and a
              widened parameter type reads as changed, so both are refused
              although neither breaks a caller; a member added to an interface
              reads as an addition and passes, although a required one breaks
              code that constructs or implements that interface.

  A kit directory the baseline has and the working tree does not is a breaking
  change: every item it exported is removed.

NOT COMPARABLE — PASSES, SAYING SO
  the baseline is below the first version of the public line;
  --sibling is given and the baseline's rust/Cargo.toml declares no version on
  its path dependency on the sibling;
  a language directory the baseline does not have.
  A repository with no vX.Y.Z tag is not a pass: a clone without its tags would
  pass every change unjudged, so the check stops with exit 2 (fetch the tags).

OPTIONS
  --sibling DIR    a repository REPO's manifests reach by relative path, checked
                   out beside REPO (DIR's directory name is the one the manifests
                   name). rust/Cargo.toml declares the sibling's version on that
                   path dependency (version = "X.Y.Z"), and the declaration is held
                   equal to the version the sibling's own rust/Cargo.toml carries.
                   The baseline lays the sibling out beside REPO at the tag
                   vX.Y.Z the baseline's rust/Cargo.toml declares.
  --tools DIR      install the pinned tools into DIR/bin when missing there, and
                   use them. Without it the pinned tools must be on PATH.
  --work DIR       build the compared trees under DIR and keep them; by default a
                   temporary directory, removed on exit.
  --languages L    a comma-separated subset of go,rust,typescript; by default
                   every language directory REPO has in its working tree or at
                   the baseline tag.

  The compared trees are copies: the baseline from the tag (git archive), the
  current side from the working trees of REPO and the sibling, untracked files
  included and ignored files excluded. Neither REPO nor the sibling is written.

PAIRING — THE PRECONDITION IT RESTS ON
  With --sibling, the baseline is paired with the sibling tag named by the
  version REPO's rust/Cargo.toml declared at the baseline tag. That tag is the
  sibling release the baseline was built on only when two things hold, and
  nothing here makes them hold:
    each release of either repository is tagged at the version its manifests
    carry; and
    REPO's declared sibling version is raised to the sibling release it builds
    on before REPO is tagged.
  Where they do not, the paired tag can differ from what the baseline used. If
  it lacks something the baseline used, the baseline fails to build; if the
  sibling has no tag of the declared version — never tagged at it, or not tagged
  yet because REPO was tagged before the sibling release its declaration names —
  there is nothing to lay out. Either way the check stops with exit 2, naming
  this as the likely cause. If the baseline still builds, the comparison runs
  against a sibling other than the one that release shipped with, which nothing
  here detects.

REQUIREMENTS
  git, tar, and per language: go 1.26 or later (the pinned apidiff's floor;
  apidiff runs `go list`); cargo, with rustup to select the pinned toolchain, or
  a rustc of at least 1.93 on PATH; node and npm (the dependencies of both sides
  are installed with `npm ci`).

EXIT STATUS
  0  the API is compatible, or the version marks the break, or nothing was
     comparable (each said on its own line)
  1  refused: a breaking change the version does not mark; manifests that
     disagree or declare no X.Y.Z version; or, with --sibling, a declared
     sibling version unequal to the sibling's own
  2  the check could not run: usage, a missing or mismatched tool, a tree that
     does not build, a repository with no release tag, or a sibling missing the
     tag the baseline declares
EOF
}

say() { printf '%s: %s\n' "$PROG" "$*"; }
fail2() { say "cannot run: $*" >&2; exit 2; }

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------
REPO=""; SIBLING=""; TOOLS=""; WORK=""; LANGS=""
while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --sibling) SIBLING="${2:-}"; shift 2 || fail2 "--sibling needs a directory" ;;
    --tools) TOOLS="${2:-}"; shift 2 || fail2 "--tools needs a directory" ;;
    --work) WORK="${2:-}"; shift 2 || fail2 "--work needs a directory" ;;
    --languages) LANGS="${2:-}"; shift 2 || fail2 "--languages needs a list" ;;
    -*) usage >&2; fail2 "unknown option $1" ;;
    *) [ -z "$REPO" ] || fail2 "one REPO only"; REPO="$1"; shift ;;
  esac
done
[ -n "$REPO" ] || { usage >&2; exit 2; }
REPO="$(cd "$REPO" 2>/dev/null && pwd)" || fail2 "no such directory: $REPO"
git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1 || fail2 "$REPO is not a git repository"
REPO_NAME="$(basename "$REPO")"
if [ -n "$SIBLING" ]; then
  SIBLING="$(cd "$SIBLING" 2>/dev/null && pwd)" || fail2 "no such directory: $SIBLING"
  git -C "$SIBLING" rev-parse --git-dir >/dev/null 2>&1 || fail2 "$SIBLING is not a git repository"
  SIBLING_NAME="$(basename "$SIBLING")"
  [ "$(dirname "$SIBLING")" = "$(dirname "$REPO")" ] \
    || fail2 "the sibling must sit beside $REPO_NAME, as its manifests reach it: $SIBLING"
fi

LANG_LIST=()
if [ -n "$LANGS" ]; then
  IFS=',' read -r -a LANG_LIST <<<"$LANGS"
  for l in "${LANG_LIST[@]}"; do
    case "$l" in go|rust|typescript) ;; *) fail2 "unknown language $l" ;; esac
  done
fi
has_lang() { local l; for l in "${LANG_LIST[@]}"; do [ "$l" = "$1" ] && return 0; done; return 1; }

# ---------------------------------------------------------------------------
# Manifests
# ---------------------------------------------------------------------------

# cargo_package_key FILE KEY -> the string value of KEY in FILE's [package] table.
cargo_package_key() {
  awk -v k="$2" '/^\[/ { p = ($0 == "[package]") }
    p && $0 ~ ("^" k "[ \t]*=") { sub(/^[^=]*=[ \t]*"/, ""); sub(/".*/, ""); print; exit }' "$1"
}

# cargo_path_dep_version FILE NAME -> the version FILE's [dependencies] table
# declares on its path dependency reaching ../../NAME/, or nothing.
cargo_path_dep_version() {
  awk -v p="path = \"../../$2/" '/^\[/ { d = ($0 == "[dependencies]") }
    d && index($0, p) {
      if (match($0, /version[ \t]*=[ \t]*"[^"]*"/)) {
        v = substr($0, RSTART, RLENGTH); sub(/^[^"]*"/, "", v); sub(/"$/, "", v); print v
      }
      exit
    }' "$1"
}

# ---------------------------------------------------------------------------
# Versions
# ---------------------------------------------------------------------------

# semver_core VERSION -> "MAJOR MINOR PATCH", or nothing when VERSION is not one.
semver_core() {
  local v="${1#v}"; v="${v%%[-+]*}"
  if [[ "$v" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    printf '%s %s %s\n' "$((10#${BASH_REMATCH[1]}))" "$((10#${BASH_REMATCH[2]}))" "$((10#${BASH_REMATCH[3]}))"
  fi
}

# semver_cmp A B -> prints -1, 0 or 1 comparing the cores.
semver_cmp() {
  local a b; read -r -a a <<<"$(semver_core "$1")"; read -r -a b <<<"$(semver_core "$2")"
  local i; for i in 0 1 2; do
    if [ "${a[$i]}" -lt "${b[$i]}" ]; then echo -1; return; fi
    if [ "${a[$i]}" -gt "${b[$i]}" ]; then echo 1; return; fi
  done
  echo 0
}

# next_breaking BASELINE -> the lowest version that marks a breaking change
# against BASELINE: below 1.0.0 the next minor (0.9.3 -> 0.10.0), from 1.0.0 the
# next major (1.2.3 -> 2.0.0). This is the version rule's one statement.
next_breaking() {
  local b; read -r -a b <<<"$(semver_core "$1")"
  if [ "${b[0]}" -eq 0 ]; then echo "0.$((b[1] + 1)).0"; else echo "$((b[0] + 1)).0.0"; fi
}

# breaking_permitted PREPARED BASELINE -> exit 0 when PREPARED marks a breaking
# change against BASELINE.
breaking_permitted() {
  [ "$(semver_cmp "$1" "$(next_breaking "$2")")" -ge 0 ]
}

# newest_tag DIR -> the newest vX.Y.Z tag of the repository at DIR, by version.
newest_tag() {
  git -C "$1" tag --list 'v*' --sort=-v:refname | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/ { print; exit }'
}

# manifest_version FILE -> the package version a Cargo.toml or package.json declares.
manifest_version() {
  case "$1" in
    *Cargo.toml)
      cargo_package_key "$1" version ;;
    *package.json)
      node -e 'process.stdout.write(String(require(process.argv[1]).version || ""))' "$1" ;;
  esac
}

PREPARED=""; PREPARED_FROM=""
for m in rust/Cargo.toml typescript/package.json; do
  [ -f "$REPO/$m" ] || continue
  v="$(manifest_version "$REPO/$m")"
  [ -n "$(semver_core "$v")" ] || { say "REFUSED: $REPO_NAME/$m declares no X.Y.Z version (\"$v\")"; exit 1; }
  if [ -z "$PREPARED" ]; then
    PREPARED="$v"; PREPARED_FROM="$m"
  elif [ "$v" != "$PREPARED" ]; then
    say "REFUSED: $REPO_NAME carries one version for all its languages, and its manifests disagree: $PREPARED_FROM declares $PREPARED, $m declares $v"
    exit 1
  fi
done

# declared_sibling_version CARGO_TOML -> the X.Y.Z the manifest declares on its
# path dependency on the sibling, a leading requirement operator dropped; nothing
# when it declares none.
declared_sibling_version() {
  local v; v="$(cargo_path_dep_version "$1" "$SIBLING_NAME")"
  v="${v#[=^~]}"; v="${v# }"
  printf '%s' "$v"
}

if [ -n "$SIBLING" ] && [ -f "$REPO/rust/Cargo.toml" ]; then
  # The declaration is what pairs a baseline with its sibling, so the working tree
  # keeps it equal to the sibling version it is built against.
  declared="$(declared_sibling_version "$REPO/rust/Cargo.toml")"
  sib_own=""
  [ -f "$SIBLING/rust/Cargo.toml" ] && sib_own="$(cargo_package_key "$SIBLING/rust/Cargo.toml" version)"
  if [ -z "$(semver_core "$declared")" ]; then
    say "REFUSED: $REPO_NAME/rust/Cargo.toml declares no X.Y.Z version on its path dependency on $SIBLING_NAME (\"$declared\"); declare version = \"$sib_own\""
    exit 1
  fi
  if [ "$declared" != "$sib_own" ]; then
    say "REFUSED: $REPO_NAME/rust/Cargo.toml declares $SIBLING_NAME $declared, and the $SIBLING_NAME it builds against carries $sib_own; set the declared version to $sib_own"
    exit 1
  fi
fi

BASE_TAG="$(newest_tag "$REPO")"
if [ -z "$BASE_TAG" ]; then
  fail2 "$REPO_NAME has no vX.Y.Z tag, so there is no release to compare with; a clone without its tags would pass every change unjudged — fetch the tags (git fetch --tags, or a checkout with full history)"
fi
if [ "$(semver_cmp "$BASE_TAG" "$FIRST_VERSION")" -lt 0 ]; then
  say "$REPO_NAME's newest release $BASE_TAG predates the public version line, which starts at $FIRST_VERSION: not comparable, passes"
  exit 0
fi
if [ -z "$PREPARED" ]; then
  # Only Go present: the repository's version is the next tag, not known yet, so
  # no break is permitted.
  PREPARED="${BASE_TAG#v}"; PREPARED_FROM="the baseline tag (no manifest carries a version)"
fi
if [ "$(semver_cmp "$PREPARED" "$BASE_TAG")" -lt 0 ]; then
  # A release tagged without raising the manifests leaves them behind the tag. The
  # version being prepared is then the tag's own, which marks no break: a breaking
  # change still has to raise the manifests past it.
  say "$REPO_NAME's manifests declare $PREPARED, below its last release $BASE_TAG: the version being prepared is taken as ${BASE_TAG#v}"
  PREPARED="${BASE_TAG#v}"; PREPARED_FROM="the baseline tag (the manifests are behind it)"
fi

SIB_TAG=""
if [ -n "$SIBLING" ]; then
  # The sibling tag paired with the baseline is the one named by the version the
  # baseline's own manifest declares; a tag's date says nothing about it. That tag
  # is the release the baseline was built on only under the precondition PAIRING in
  # --help states.
  base_declared=""
  if git -C "$REPO" cat-file -e "$BASE_TAG:rust/Cargo.toml" 2>/dev/null; then
    base_declared="$(declared_sibling_version <(git -C "$REPO" show "$BASE_TAG:rust/Cargo.toml"))"
  fi
  if [ -z "$(semver_core "$base_declared")" ]; then
    say "$REPO_NAME $BASE_TAG declares no $SIBLING_NAME version in rust/Cargo.toml, so no $SIBLING_NAME tag can be paired with it: not comparable, passes"
    exit 0
  fi
  SIB_TAG="v$base_declared"
  git -C "$SIBLING" rev-parse -q --verify "refs/tags/$SIB_TAG" >/dev/null \
    || fail2 "$REPO_NAME $BASE_TAG declares $SIBLING_NAME $base_declared, and $SIBLING_NAME has no tag $SIB_TAG. Either this clone lacks $SIBLING_NAME's tags (fetch them), or $SIBLING_NAME $base_declared was never tagged — a release of $SIBLING_NAME cut at another version than its manifests carried, so the version $REPO_NAME declared names no release — or it is not tagged yet, because $REPO_NAME was tagged before the $SIBLING_NAME release its declaration names (see PAIRING in --help)"
fi

# pairing_hint -> the likely cause, beside what a baseline build failure says,
# when the baseline is laid out with its sibling: the paired sibling tag lacks
# what the baseline release used.
pairing_hint() {
  [ -n "$SIBLING" ] || return 0
  printf ' If the %s baseline is what failed, the likely cause is the pairing: %s %s declares %s %s, and the %s tag %s lacks what that release used — a release of %s tagged at another version than its manifests carried, or %s'"'"'s declared version not raised before %s was tagged (see PAIRING in --help).' \
    "$BASE_TAG" "$REPO_NAME" "$BASE_TAG" "$SIBLING_NAME" "${SIB_TAG#v}" "$SIBLING_NAME" "$SIB_TAG" "$SIBLING_NAME" "$REPO_NAME" "$REPO_NAME"
}

# The languages compared: those asked for, or every kit directory the working
# tree or the baseline has — a kit the baseline has and the tree lost is a
# removal, not a language to skip.
if [ "${#LANG_LIST[@]}" = 0 ]; then
  for l in go rust typescript; do
    if [ -d "$REPO/$l" ] || [ -n "$(git -C "$REPO" ls-tree --name-only "$BASE_TAG" -- "$l")" ]; then
      LANG_LIST+=("$l")
    fi
  done
fi

if breaking_permitted "$PREPARED" "$BASE_TAG"; then PERMITTED=1; else PERMITTED=0; fi
say "$REPO_NAME: version being prepared $PREPARED (from $PREPARED_FROM), baseline $BASE_TAG${SIB_TAG:+, with $SIBLING_NAME at $SIB_TAG}"
if [ "$PERMITTED" = 1 ]; then
  say "$PREPARED marks a breaking change against $BASE_TAG: incompatible changes are reported and permitted"
else
  say "$PREPARED marks no breaking change against $BASE_TAG: an incompatible change is refused"
fi

# ---------------------------------------------------------------------------
# Tools
# ---------------------------------------------------------------------------
host_triple() {
  local m o; m="$(uname -m)"; o="$(uname -s)"
  case "$m" in arm64) m=aarch64 ;; esac
  case "$o" in Linux) echo "$m-unknown-linux-gnu" ;; Darwin) echo "$m-apple-darwin" ;; *) echo "$m-$o" ;; esac
}

install_tools() {
  local dir="$1" bin="$1/bin"
  mkdir -p "$bin" || fail2 "cannot create $bin"
  if has_lang go && [ ! -x "$bin/apidiff" ]; then
    say "installing apidiff (golang.org/x/exp $APIDIFF_MODULE_VERSION) into $bin"
    GOBIN="$bin" GOFLAGS='' GOWORK=off go install "golang.org/x/exp/cmd/apidiff@$APIDIFF_MODULE_VERSION" \
      || fail2 "go install of apidiff failed"
  fi
  if has_lang rust && [ ! -x "$bin/cargo-semver-checks" ]; then
    local triple sum archive
    triple="$(host_triple)"; sum="$(csc_sha256 "$triple")"
    if [ -n "$sum" ]; then
      say "installing cargo-semver-checks $CSC_VERSION ($triple release archive) into $bin"
      archive="$dir/cargo-semver-checks-$CSC_VERSION-$triple.tar.gz"
      curl -fsSL -o "$archive" \
        "https://github.com/obi1kenobi/cargo-semver-checks/releases/download/$CSC_VERSION/cargo-semver-checks-$triple.tar.gz" \
        || fail2 "download of cargo-semver-checks failed"
      local got
      if command -v sha256sum >/dev/null; then got="$(sha256sum "$archive")"; else got="$(shasum -a 256 "$archive")"; fi
      [ "${got%% *}" = "$sum" ] || fail2 "cargo-semver-checks archive digest ${got%% *} is not the pinned $sum"
      tar -C "$bin" -xzf "$archive" cargo-semver-checks || fail2 "cannot unpack $archive"
    else
      say "installing cargo-semver-checks $CSC_VERSION from source into $dir (no release archive pinned for $triple)"
      cargo "+$RUST_API_TOOLCHAIN" install --locked --root "$dir" cargo-semver-checks --version "${CSC_VERSION#v}" \
        || fail2 "cargo install of cargo-semver-checks failed"
    fi
  fi
  if has_lang rust && command -v rustup >/dev/null; then
    if ! rustup toolchain list | awk -v t="$RUST_API_TOOLCHAIN" 'index($1, t "-") == 1 { found = 1 } END { exit !found }'; then
      say "installing the Rust toolchain $RUST_API_TOOLCHAIN (minimal profile) for the API comparison"
      rustup toolchain install "$RUST_API_TOOLCHAIN" --profile minimal || fail2 "rustup could not install $RUST_API_TOOLCHAIN"
    fi
  fi
  PATH="$bin:$PATH"; export PATH
}

check_tools() {
  if has_lang go; then
    command -v go >/dev/null || fail2 "go is not on PATH"
    local ad; ad="$(command -v apidiff)" || fail2 "apidiff is not on PATH (pass --tools DIR to install the pinned one)"
    go version -m "$ad" | awk -v v="$APIDIFF_MODULE_VERSION" '$1 == "mod" && $2 == "golang.org/x/exp" && $3 == v { ok = 1 } END { exit !ok }' \
      || fail2 "$ad is not apidiff from golang.org/x/exp $APIDIFF_MODULE_VERSION"
  fi
  if has_lang rust; then
    command -v cargo >/dev/null || fail2 "cargo is not on PATH"
    local v; v="$(cargo-semver-checks --version 2>/dev/null)" \
      || fail2 "cargo-semver-checks is not on PATH (pass --tools DIR to install the pinned one)"
    [ "$v" = "cargo-semver-checks ${CSC_VERSION#v}" ] || fail2 "found $v, the pinned one is ${CSC_VERSION#v}"
    if command -v rustup >/dev/null; then
      export RUSTUP_TOOLCHAIN="$RUST_API_TOOLCHAIN"
    fi
  fi
  if has_lang typescript; then
    command -v node >/dev/null || fail2 "node is not on PATH"
    command -v npm >/dev/null || fail2 "npm is not on PATH"
  fi
}

[ -z "$TOOLS" ] || { mkdir -p "$TOOLS" && TOOLS="$(cd "$TOOLS" && pwd)" && install_tools "$TOOLS"; }
check_tools

# ---------------------------------------------------------------------------
# The compared trees
# ---------------------------------------------------------------------------
if [ -z "$WORK" ]; then
  WORK="$(mktemp -d "${TMPDIR:-/tmp}/check-api-compat.XXXXXX")" || fail2 "cannot create a work directory"
  trap 'rm -rf "$WORK"' EXIT
else
  mkdir -p "$WORK" && WORK="$(cd "$WORK" && pwd)" || fail2 "cannot create $WORK"
  rm -rf "$WORK/old" "$WORK/new" "$WORK/out"
fi
OLD="$WORK/old"; NEW="$WORK/new"; OUT="$WORK/out"
mkdir -p "$OLD" "$NEW" "$OUT"

# from_tag DIR TAG DEST — the tree at TAG.
from_tag() {
  mkdir -p "$3" && git -C "$1" archive --format=tar "$2" | tar -x -C "$3"
}
# from_worktree DIR DEST — the working tree: tracked and untracked files, ignored
# ones excluded, a tracked file deleted from the working tree left out.
from_worktree() {
  mkdir -p "$2" && (cd "$1" && git ls-files -z --cached --others --exclude-standard \
    | while IFS= read -r -d '' f; do
        if [ -e "$f" ] || [ -L "$f" ]; then printf '%s\0' "$f"; fi
      done | tar --null -T - -cf -) | tar -x -C "$2"
}
indent() { awk '{ print "    " $0 }' "$@"; }
from_tag "$REPO" "$BASE_TAG" "$OLD/$REPO_NAME" || fail2 "cannot read $REPO_NAME at $BASE_TAG"
from_worktree "$REPO" "$NEW/$REPO_NAME" || fail2 "cannot copy the $REPO_NAME working tree"
if [ -n "$SIBLING" ]; then
  from_tag "$SIBLING" "$SIB_TAG" "$OLD/$SIBLING_NAME" || fail2 "cannot read $SIBLING_NAME at $SIB_TAG"
  from_worktree "$SIBLING" "$NEW/$SIBLING_NAME" || fail2 "cannot copy the $SIBLING_NAME working tree"
fi

BROKE=()      # languages with an incompatible change
NOTCOMP=()    # languages not comparable

# lang_present LANG -> 0 when both sides have it; records the other cases.
lang_present() {
  local l="$1"
  if [ ! -d "$NEW/$REPO_NAME/$l" ] && [ -d "$OLD/$REPO_NAME/$l" ]; then
    say "[$l] the $l kit is gone: every item it exported is removed"
    BROKE+=("$l"); return 1
  fi
  if [ ! -d "$OLD/$REPO_NAME/$l" ]; then
    say "[$l] $BASE_TAG has no $l kit: not comparable"
    NOTCOMP+=("$l"); return 1
  fi
  return 0
}

# ---------------------------------------------------------------------------
# Go — apidiff over the module, on the host platform and on the plugin target.
# apidiff exits 0 whatever it finds, so the verdict is read from what
# -incompatible prints.
# ---------------------------------------------------------------------------
check_go() {
  lang_present go || return 0
  local old="$OLD/$REPO_NAME/go" new="$NEW/$REPO_NAME/go" mod_old mod_new plat goos goarch broke=0
  mod_old="$(cd "$old" && GOWORK=off go list -m 2>"$OUT/go-mod-old.err")" || { cat "$OUT/go-mod-old.err" >&2; fail2 "go list -m failed at $BASE_TAG"; }
  mod_new="$(cd "$new" && GOWORK=off go list -m 2>"$OUT/go-mod-new.err")" || { cat "$OUT/go-mod-new.err" >&2; fail2 "go list -m failed in the working tree"; }
  if [ "$mod_old" != "$mod_new" ]; then
    say "[go] the module path changed from $mod_old to $mod_new: every package is removed for an importer"
    BROKE+=("go"); return 0
  fi
  for plat in "$(go env GOOS)/$(go env GOARCH)" wasip1/wasm; do
    goos="${plat%/*}"; goarch="${plat#*/}"
    local exp="$OUT/go-$goos-$goarch.exp" inc="$OUT/go-$goos-$goarch.incompatible" all="$OUT/go-$goos-$goarch.all"
    (cd "$old" && GOWORK=off GOOS="$goos" GOARCH="$goarch" apidiff -m -w "$exp" "$mod_old") \
      || fail2 "apidiff could not load $mod_old at $BASE_TAG for $plat.$(pairing_hint)"
    (cd "$new" && GOWORK=off GOOS="$goos" GOARCH="$goarch" apidiff -m -incompatible "$exp" "$mod_new" >"$inc") \
      || fail2 "apidiff could not load $mod_new in the working tree for $plat"
    (cd "$new" && GOWORK=off GOOS="$goos" GOARCH="$goarch" apidiff -m "$exp" "$mod_new" >"$all") \
      || fail2 "apidiff could not load $mod_new in the working tree for $plat"
    if [ -s "$inc" ]; then
      say "[go $plat] incompatible changes:"; indent "$inc"; broke=1
    else
      say "[go $plat] no incompatible change ($(awk '/^- / { n++ } END { print n + 0 }' "$all") compatible)"
    fi
  done
  [ "$broke" = 0 ] || BROKE+=("go")
}

# ---------------------------------------------------------------------------
# Rust — cargo-semver-checks against the baseline tree. It always judges the
# change as a minor release — additions pass, breaking changes fail — and the
# version rule above decides whether a breaking change is permitted, as for the
# other languages; the release type it would derive from the manifests compares
# against the baseline's manifest rather than the tag.
# ---------------------------------------------------------------------------
check_rust() {
  lang_present rust || return 0
  local new="$NEW/$REPO_NAME/rust" old="$OLD/$REPO_NAME/rust" pkg log="$OUT/rust.log" rc
  pkg="$(cargo_package_key "$new/Cargo.toml" name)"
  [ -n "$pkg" ] || fail2 "rust/Cargo.toml names no package"
  CARGO_TARGET_DIR="$WORK/target" cargo semver-checks --color never \
    --manifest-path "$new/Cargo.toml" --package "$pkg" --baseline-root "$old" --release-type minor >"$log" 2>&1
  rc=$?
  indent "$log"
  if [ "$rc" = 0 ]; then
    say "[rust] no incompatible change"
  elif awk '/Summary semver requires new/ { f = 1 } END { exit !f }' "$log"; then
    say "[rust] incompatible changes (above)"; BROKE+=("rust")
  else
    fail2 "cargo semver-checks failed (exit $rc) without a verdict; its output is above.$(pairing_hint)"
  fi
}

# ---------------------------------------------------------------------------
# TypeScript — the emitted declarations, compared line by line (bounds in --help).
# ---------------------------------------------------------------------------
# npm_ci DIR LOG — install DIR's locked dependencies, running no package script.
npm_ci() {
  (cd "$1" && npm ci --ignore-scripts --no-audit --no-fund >"$2" 2>&1) \
    || { cat "$2" >&2; fail2 "npm ci failed in $1"; }
}

check_typescript() {
  lang_present typescript || return 0
  local new="$NEW/$REPO_NAME/typescript" old="$OLD/$REPO_NAME/typescript" diff="$OUT/ts.diff" rc removed side dir
  # The current side's compiler emits both sides, so a compiler version
  # difference cannot read as an API change.
  npm_ci "$new" "$OUT/npm-new.log"
  npm_ci "$old" "$OUT/npm-old.log"
  local tsc="$new/node_modules/typescript/bin/tsc"
  [ -f "$tsc" ] || fail2 "the working tree's typescript/ has no typescript compiler among its dependencies"
  # A package reaching the sibling resolves its types from the sibling's built
  # declarations, so the sibling is installed and built on each side first.
  if [ -n "$SIBLING" ]; then
    for side in old new; do
      dir="$WORK/$side/$SIBLING_NAME/typescript"
      [ -d "$dir" ] || continue
      npm_ci "$dir" "$OUT/npm-sibling-$side.log"
      (cd "$dir" && node "$tsc" -p tsconfig.json >"$OUT/tsc-sibling-$side.log" 2>&1) \
        || { cat "$OUT/tsc-sibling-$side.log" >&2; fail2 "the $SIBLING_NAME TypeScript package does not build ($side side)."; }
    done
  fi
  for side in old new; do
    if [ "$side" = old ]; then dir="$old"; else dir="$new"; fi
    (cd "$dir" && node "$tsc" -p tsconfig.json --declaration --emitDeclarationOnly --removeComments \
      --outDir "$OUT/dts-$side" >"$OUT/tsc-$side.log" 2>&1) \
      || { cat "$OUT/tsc-$side.log" >&2; fail2 "tsc could not emit the $side declarations.$( [ "$side" = old ] && pairing_hint )"; }
  done
  git diff --no-index --no-color --no-renames --no-ext-diff -U0 --ignore-blank-lines \
    "$OUT/dts-old" "$OUT/dts-new" >"$diff"
  rc=$?
  [ "$rc" -le 1 ] || fail2 "git diff could not compare the declarations (exit $rc)"
  removed="$(awk '/^-/ && !/^--- / { n++ } END { print n + 0 }' "$diff")"
  if [ "$removed" -gt 0 ]; then
    say "[typescript] removed or changed declarations:"
    awk '/^diff --git/ { f = $NF; sub(/.*\/dts-(old|new)\//, "", f) }
         /^-/ && !/^--- / { print "    " f ": " $0 }
         /^\+/ && !/^\+\+\+ / { print "    " f ": " $0 }' "$diff"
    BROKE+=("typescript")
  else
    say "[typescript] no removed or changed declaration ($(awk '/^\+/ && !/^\+\+\+ / { n++ } END { print n + 0 }' "$diff") added lines)"
  fi
}

has_lang go && check_go
has_lang rust && check_rust
has_lang typescript && check_typescript

# ---------------------------------------------------------------------------
# Verdict
# ---------------------------------------------------------------------------
COMPARED=()
for l in "${LANG_LIST[@]}"; do
  case " ${NOTCOMP[*]} " in *" $l "*) ;; *) COMPARED+=("$l") ;; esac
done
unjudged=""
[ "${#NOTCOMP[@]}" = 0 ] || unjudged="; not compared: ${NOTCOMP[*]}"
if [ "${#BROKE[@]}" = 0 ]; then
  if [ "${#COMPARED[@]}" = 0 ]; then
    say "PASS: no language was comparable with $BASE_TAG$unjudged"
  else
    say "PASS: $REPO_NAME's exported API keeps every item $BASE_TAG exported, in ${COMPARED[*]}$unjudged"
  fi
  exit 0
fi
if [ "$PERMITTED" = 1 ]; then
  say "PASS: incompatible changes in ${BROKE[*]}, marked breaking by $PREPARED against $BASE_TAG$unjudged"
  exit 0
fi
say "REFUSED: $REPO_NAME's exported API changed incompatibly (${BROKE[*]}) and $PREPARED marks no breaking change against $BASE_TAG; restore the API, or raise the version in every manifest to $(next_breaking "$BASE_TAG")$unjudged"
exit 1
