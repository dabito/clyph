# clyph

`clyph` is a Nerd Font glyph lookup CLI for shell scripts, status lines, and agent-friendly UI work.

Search glyph names, print exact glyphs, return codepoints, and refresh a local offline catalog from Nerd Fonts CSS.

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

Environment overrides:

```bash
export CLYPH_DATA_DIR="$HOME/.clyph/data"          # uses $CLYPH_DATA_DIR/catalog.json
export CLYPH_CATALOG_PATH="$PWD/data/catalog.json" # exact file override, takes precedence
```

## Usage

```bash
clyph search circle --limit 5
clyph get nf-md-check
clyph glyph nf-md-check
clyph codepoint nf-md-check
clyph update --source ./webfont.css
```

Use JSON for scripts:

```bash
clyph search circle --json
clyph get nf-md-check --json
clyph glyph nf-md-check --json
clyph codepoint nf-md-check --json
clyph update --json
```

Print a glyph inside shell output:

```bash
printf "status: %s done\n" "$(clyph glyph nf-md-check)"
```

## Commands

```text
clyph search <query> [--limit N] [--json]
clyph get <name> [--json]
clyph glyph <name> [--json]
clyph codepoint <name> [--json]
clyph update [--source <file-or-url>] [--json]
```

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
