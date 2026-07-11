#!/usr/bin/env bash
set -euo pipefail

# Demo script for clyph. Record with:
#   asciinema rec --command "env CLYPH_BIN=/tmp/clyph_bin bash docs/demo/clyph-demo.sh" \
#     docs/demo/clyph.cast
# Render with (Nerd Font family is required so glyphs render, not tofu):
#   agg docs/demo/clyph.cast docs/demo/clyph.gif \
#     --font-size 22 --cols 80 --theme monokai \
#     --text-font-family "0xProto Nerd Font Mono"

CLYPH_BIN="${CLYPH_BIN:-clyph}"

# Isolate catalog writes (alias/label) in a temp copy so the shipped
# data/catalog.json is never mutated by the demo. Also expose the binary as
# `clyph` on PATH so recorded commands read like real usage.
SRC_CATALOG="${CLYPH_CATALOG_PATH:-$PWD/data/catalog.json}"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
export CLYPH_CATALOG_PATH="$WORKDIR/catalog.json"
cp "$SRC_CATALOG" "$CLYPH_CATALOG_PATH"
mkdir -p "$WORKDIR/bin"
ln -sf "$(command -v "$CLYPH_BIN")" "$WORKDIR/bin/clyph"
export PATH="$WORKDIR/bin:$PATH"
cd "$WORKDIR"

say() {
  printf '\n\033[1;36m# %s\033[0m\n' "$*"
  sleep 1.3
}

run() {
  printf '\n\033[1;32m$ %s\033[0m\n' "$*"
  sleep 0.7
  eval "$@"
  sleep 1.3
}

clear
say "clyph: offline Nerd Font glyph lookup"

say "Scale of the local catalog"
run "clyph stats"
run "clyph families --limit 5"

say "Search by name — no web browser needed"
run "clyph search check --limit 5"

say "Print the glyph itself"
run "clyph glyph nf-md-check"

say "One name, every code format"
run "clyph fmt nf-md-check --format css"
run "clyph fmt nf-md-check"

say "Reverse lookup: paste a glyph, get its name back"
GLYPH="$(clyph glyph nf-md-check)"
run "clyph identify $GLYPH"

say "Use it in scripts: print a status message"
run 'printf "%s deploy succeeded\n" "$(clyph glyph nf-md-check)"'

say "Compose a status line from several glyphs"
run 'printf "%s main  %s clean  %s v1.2.0\n" "$(clyph glyph nf-md-git)" "$(clyph glyph nf-md-check)" "$(clyph glyph nf-md-tag)"'

say "Tag a glyph with a stable alias, then resolve it by name"
run "clyph alias nf-md-check add success"
run "clyph search success"

say "Takeaway: search, resolve, format, identify, curate, and script — fully offline"
sleep 2.0
