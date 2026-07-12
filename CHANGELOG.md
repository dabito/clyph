# Changelog

All notable changes to `clyph` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-07-12

### Added
- `clyph semantic <concept> [--all] [--json]` — resolve concepts like `success`, `warning`, `upload`, and `branch` to curated glyphs.
- `data/semantic.json` — embedded concept seed used by `semantic`, with `CLYPH_SEMANTIC_PATH` override for custom seeds.

### Changed
- Catalog loading now uses a disposable gob cache (`catalog.cache.gob`) when fresh, keeping JSON as canonical while reducing repeated lookup latency.
## [0.3.0] - 2026-07-09

### Added
- `clyph identify <glyph...> [--json]` — reverse lookup: glyph character -> name, codepoint, family. Reads glyphs from stdin when none are given.
- `clyph fmt <name> [--format html|css|unicode|js|hex|octal|all] [--json]` — format a glyph's codepoint for HTML, CSS, JS, etc. Default `all` prints a labeled block; a single `--format` prints the bare value.
- `clyph families [--limit N] [--json]` — per-family glyph counts, sorted by count desc.
- `clyph stats [--json]` — catalog summary: total records, family count, labeled and aliased counts.
- `ROADMAP.md`.
- Terminal demo: `docs/demo/clyph.cast` + `docs/demo/clyph.gif`, with a `## Demo` section in the README.

## [0.2.1] - 2026-07-03

### Added
- `--limit`, `--offset`, and `--source` now accept `--flag=value` in addition to `--flag value`.

### Changed
- Record JSON now omits `label`/`aliases` entirely when unset, instead of emitting `"label": ""` / `"aliases": []`.

## [0.2.0] - 2026-07-02

### Added
- `clyph search --pretty` for space-aligned columns in a terminal. Default tab-separated output relies on fixed terminal tab stops, which drift out of alignment once a Nerd Font name is longer than one tab stop.
- `clyph search --offset N` to page past `--limit`.
- `--help`/`-h` on every subcommand (e.g. `clyph label --help`) for a one-line usage reminder, not just the top-level command.
- `search` JSON output now includes `total` (match count before `--limit`/`--offset`) and `offset` fields alongside `matches`.

### Changed
- `search` default `--limit` raised from 10 to 100.
- Truncated `search` results are no longer silent: plain output prints `showing START-END of TOTAL matches; use --offset/--limit to see more` to stderr whenever the page doesn't cover every match.

### Fixed
- `--limit 0` now returns zero matches (previously returned one, an off-by-one in the truncation check).

## [0.1.0-beta.5] - 2026-07-02

### Added
- `clyph label <name> <text>` and `clyph label <name> --clear` to set/clear a record's label.
- `clyph alias <name> add|rm <value>` to manage a record's aliases.

### Changed
- `--json` failures now emit `{"error": "..."}` on stderr instead of plain text, on every command.
- `search` now normalizes underscores and spaces interchangeably, so `clyph search "arrow circle"` matches `arrow_circle_down`.
- Missing-catalog errors now name the expected path and point at `clyph update` instead of surfacing the raw filesystem error.
- README shared-family sentence now uses the canonical family text verbatim.
- README Requirements now note the Nerd Font terminal requirement for rendering glyph output.

### Fixed
- Removed an unreachable "match everything" branch in `searchRecords` (empty query can never reach the CLI; `parseSearchArgs` already rejects it).
- Removed a `Related packages` link to `pi-cake`, which isn't a published package (Lab-only, unpublished) and should never have been cross-linked from a public README.

## [0.1.0-beta.4] - 2026-07-01

### Changed
- Split core code into `catalog.go`, `css.go`, and `cli.go`.
- Rejected negative `--limit` values.
- README now has shared-family, Requirements, Behavior notes, and Related packages sections.

## [0.1.0-beta.3] - 2026-06-30

### Changed
- Removed `CLYPH_DATA_DIR` env override. Use `CLYPH_CATALOG_PATH` to point at an exact catalog file. Single-file data model keeps the door open for a future multi-data merge.

## [0.1.0-beta.2] - 2026-06-30

### Added
- README failure-modes section.
- README sample output block.
- README repo and issues links.
- Parser-fixture test for malformed / shifted CSS block shapes.
- `CHANGELOG.md`.

## [0.1.0-beta.1] - 2026-06-28

### Added
- `clyph search <query> [--limit N] [--json]` — substring search over glyph names, labels, and aliases.
- `clyph get <name> [--json]` — return one glyph record by exact Nerd Font name.
- `clyph glyph <name> [--json]` — print only the rendered glyph.
- `clyph codepoint <name> [--json]` — print only the codepoint.
- `clyph update [--source <file-or-url>] [--json]` — refresh the local catalog from Nerd Fonts CSS.
- `clyph version` — print version.
- Local catalog at `~/.clyph/data/catalog.json` with `CLYPH_CATALOG_PATH` override.
- CSS parser for Nerd Fonts `webfont.css`, with comment stripping and CSS-escape decoding.
- Atomic catalog writes with rollback on failed updates.
- Tab-separated plain output and stable JSON output.
- Test suite covering CLI paths, CSS parser, env overrides, update rollback, and empty-source rejection.

[Unreleased]: https://github.com/dabito/clyph/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/dabito/clyph/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/dabito/clyph/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/dabito/clyph/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/dabito/clyph/compare/v0.1.0-beta.5...v0.2.0
[0.1.0-beta.5]: https://github.com/dabito/clyph/compare/v0.1.0-beta.4...v0.1.0-beta.5
[0.1.0-beta.4]: https://github.com/dabito/clyph/compare/v0.1.0-beta.3...v0.1.0-beta.4
[0.1.0-beta.3]: https://github.com/dabito/clyph/compare/v0.1.0-beta.2...v0.1.0-beta.3
[0.1.0-beta.2]: https://github.com/dabito/clyph/compare/v0.1.0-beta.1...v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/dabito/clyph/releases/tag/v0.1.0-beta.1
