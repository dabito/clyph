# PRD: clyph

## Status

Draft v0. `clyph` is a supporting shell/CLI tool, not a pi extension. It intentionally does not use the `pi-` prefix.

## One-liner

`clyph` is a Nerd Font glyph lookup CLI: search by name or query string and return the glyph, unicode/codepoint, and metadata in deterministic shell-friendly formats.

## Problem

Our human surfaces assume Nerd Fonts, but choosing glyphs is currently manual and annoying. We need a small deterministic tool that can answer questions like:

```text
what glyph is nf-md-check?
what circle glyphs exist?
what icons match offline?
```

This supports visual language work for bash output, widgets, status lines, and pi extension renderers.

## Product goal

Make Nerd Font glyph discovery shell-first, searchable, scriptable, and easy for pi extensions to wrap.

## Users

### Primary user: human designer/developer

Needs to quickly find glyphs for status, hierarchy, and UI language.

### Secondary user: agent

Needs a discoverable command that returns concise glyph candidates without browsing CSS manually.

### Wrapper user: pi extension

May call `clyph` to render glyph dictionaries, validate configs, or help generate docs.

## Source catalog

The initial catalog can be extracted from:

```text
https://www.nerdfonts.com/assets/css/webfont.css
```

The tool should parse class/name definitions and codepoints from this CSS into a local cache or generated catalog.
Default catalog storage:

```text
~/.clyph/data/catalog.json
```

Environment override:

- `CLYPH_CATALOG_PATH` overrides the exact catalog file path. (A single-file data model is intentional: it keeps the door open for a future multi-data merge.)
## CLI menu

### `clyph search <query>`

Use `clyph search` to find Nerd Font glyphs matching a name or keyword.

Example:

```bash
clyph search circle
```

### `clyph get <name>`

Use `clyph get` to return one glyph by exact Nerd Font name.

Example:

```bash
clyph get nf-md-check
```

### `clyph codepoint <name>`

Use `clyph codepoint` to return only the unicode codepoint for a glyph.

Example:

```bash
clyph codepoint nf-md-check
```

### `clyph glyph <name>`

Use `clyph glyph` to return only the rendered glyph.

Example:

```bash
clyph glyph nf-md-check
```

### `clyph update`

Use `clyph update` to refresh the local glyph catalog from the Nerd Fonts CSS source.

## Output formats

Support concise human text and stable JSON.

```bash
clyph search circle
clyph search circle --json
```

Human output example:

```text
nf-fa-circle        f111    circle
nf-fa-circle_o      f10c    circle outline
nf-md-circle_half   f1395 󱎕  circle half
```

JSON output example:

```json
{
  "query": "circle",
  "matches": [
    {
      "name": "nf-fa-circle",
      "codepoint": "f111",
      "unicode": "\\uf111",
      "glyph": ""
    }
  ]
}
```

## MVP requirements

### P0

- Parse a local or fetched Nerd Fonts `webfont.css` catalog.
- Search by substring over glyph names.
- Return exact glyph by name.
- Return codepoint by name.
- Return rendered glyph by name.
- Support `--json`.
- Keep output deterministic and concise.

### P1

- Local cache with explicit `update`.
- Aliases/tags for common UI states, such as `done`, `failure`, `offline`, `progress`.
- Fuzzy search.
- Limit and sort flags.

### P2

- Generate TOML glyph dictionaries for `pi-cake`.
- Validate glyph names in project configs.
- Preview glyphs with surrounding text/styles.

## State alias candidates

The visual language currently needs glyphs for:

| Alias | Meaning | Desired visual |
| --- | --- | --- |
| `offline` | offline/unreachable | empty circle |
| `progress` | in progress | half-filled circle outline |
| `done` | completed | checkmark |
| `failure` | failed | cross |

Until exact Nerd Font names are chosen, fallbacks are:

```text
offline: ○
progress: ◐
done: ✓
failure: ✗
```

## Agent surface wording

Tool/help descriptions should be short, discoverable, and imperative.

Examples:

```text
Use clyph search to find Nerd Font glyphs by name or keyword.
Use clyph get to return one glyph with codepoint metadata.
Use clyph glyph to print only the rendered glyph for a Nerd Font name.
```

## Human surface principles

- Keep tabular output narrow.
- Show name, codepoint, glyph, and short label.
- Do not emit huge match lists by default.
- Use `--limit` for predictable result counts.
- Use plain output by default; JSON for automation.

## Non-goals

- Build a graphical glyph browser.
- Require a pi extension to use the tool.
- Depend on network access for every lookup.
- Solve all icon taxonomy problems in v0.

## Acceptance criteria

1. `clyph search circle` returns a short list of matching glyphs.
2. `clyph get <exact-name>` returns name, codepoint, unicode escape, and rendered glyph.
3. `clyph glyph <exact-name>` returns only the glyph.
4. `clyph search <query> --json` returns stable machine-readable results.
5. The catalog can be refreshed from Nerd Fonts `webfont.css`.
6. The CLI can be wrapped by a pi extension without parsing noisy human output.
