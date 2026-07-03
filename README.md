# clyph

`clyph` is a Nerd Font glyph lookup CLI for shell scripts, status lines, and agent-friendly UI work.

Search glyph names, print exact glyphs, return codepoints, and refresh a local offline catalog from Nerd Fonts CSS.

Built for AI coding agents: small local tools, typed inputs, deterministic text output, bounded context, and explicit failure modes.

Repo: <https://github.com/dabito/clyph> · Issues: <https://github.com/dabito/clyph/issues>

## Requirements

- Go 1.22 or later
- No external dependencies — uses Go standard library only
- A terminal/font with Nerd Font glyphs installed to render `glyph`/`search` output correctly; without it, glyph columns show as boxes or blanks

## Install

```bash
go install github.com/dabito/clyph@latest
```

Go installs the binary into `$GOBIN`, or `$GOPATH/bin` when `GOBIN` is unset. Default Go setups usually use:

```text
$HOME/go/bin/clyph
```

Ensure Go's bin dir is on `PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

## Initialize catalog

Lookups use a local catalog. Refresh it once after install:

```bash
clyph update
```

Default source:

```text
https://www.nerdfonts.com/assets/css/webfont.css
```

Default catalog path:

```text
~/.clyph/data/catalog.json
```

Environment override:

```bash
export CLYPH_CATALOG_PATH="$PWD/data/catalog.json" # exact catalog file override
```

## Usage

```bash
clyph search circle --limit 5
clyph get nf-md-check
clyph glyph nf-md-check
clyph codepoint nf-md-check
clyph update --source ./webfont.css
clyph label nf-md-check "checkmark"
clyph alias nf-md-check add tick
clyph version
```

Use JSON for scripts:

```bash
clyph search circle --json
clyph get nf-md-check --json
clyph glyph nf-md-check --json
clyph codepoint nf-md-check --json
clyph update --json
clyph label nf-md-check "checkmark" --json
clyph alias nf-md-check add tick --json
```

Errors are JSON too when `--json` is passed, on every command, so scripts never have to fall back to parsing stderr text:

```bash
$ clyph get nf-does-not-exist --json
{
  "error": "not found: nf-does-not-exist"
}
```

Print a glyph inside shell output:

```bash
printf "status: %s done\n" "$(clyph glyph nf-md-check)"
```

## Sample output

```text
$ clyph search circle --limit 5
nf-cod-arrow_circle_down	ebfc		-
nf-cod-arrow_circle_left	ebfd		-
nf-cod-arrow_circle_right	ebfe		-
nf-cod-arrow_circle_up	ebff		-
nf-cod-circle	eabc		-

$ clyph get nf-md-check
nf-md-check	f012c	󰄬	-

$ clyph update --json
{
  "status": "updated",
  "records": 10764,
  "catalog": "/home/user/.clyph/data/catalog.json"
}
```

Plain output is tab-separated: `name`, `codepoint`, `glyph`, `label`. Use `--json` for stable machine-readable output.

## Commands

```text
clyph search <query> [--limit N] [--json]
clyph get <name> [--json]
clyph glyph <name> [--json]
clyph codepoint <name> [--json]
clyph update [--source <file-or-url>] [--json]
clyph label <name> <text> [--json]
clyph label <name> --clear [--json]
clyph alias <name> <add|rm> <value> [--json]
clyph version
```

## Behavior notes

- **search --limit**: `--limit N` caps results to N. `--limit 0` returns at most 1 result (the `>=` comparison fires after the first append). Negative values are rejected with exit code 2.
- **search matches underscores and spaces interchangeably**: Nerd Font names use underscores (`arrow_circle_down`); `clyph search "arrow circle"` normalizes both the query and catalog text so either form matches.
- **Multi-rune CSS content**: Nerd Fonts CSS `content` values containing multiple Unicode escapes (e.g. `"\f444\f555"`) collapse to the first rune. Only the first codepoint is recorded; subsequent runes are dropped.
- **Label and alias assignment**: `clyph label <name> <text>` sets a record's label (`--clear` removes it); `clyph alias <name> add|rm <value>` manages its alias list. `clyph update` then preserves these across a catalog refresh — only glyphs absent from the new source are removed.

## Failure modes

- **Missing catalog**: every lookup command fails with exit code `1` until `clyph update` has been run once. The error names the expected path and points at `clyph update`.
- **Empty search**: `clyph search <query>` with no matches prints nothing and exits `0`; pass `--json` to get an empty `matches` array.
- **Bad CSS source**: `clyph update --source <file-or-url>` rejects a source that parses to zero glyph records (exit `1`) and leaves the existing catalog untouched.
- **Network failure**: `clyph update` against a URL reports `update failed: ...` and exits `1` without modifying the catalog.
- **Unknown name**: `clyph get|glyph|codepoint|label|alias <name>` prints `not found: <name>` and exits `1`; with `--json`, the same message comes back as `{"error": "not found: <name>"}`.

## Development

```bash
make test
make vet
make check
make install
```

Manual install from local checkout:

```bash
go install .
```
