# Changelog

All notable changes to `clyph` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- Local catalog at `~/.clyph/data/catalog.json` with `CLYPH_DATA_DIR` and `CLYPH_CATALOG_PATH` overrides.
- CSS parser for Nerd Fonts `webfont.css`, with comment stripping and CSS-escape decoding.
- Atomic catalog writes with rollback on failed updates.
- Tab-separated plain output and stable JSON output.
- Test suite covering CLI paths, CSS parser, env overrides, update rollback, and empty-source rejection.

[Unreleased]: https://github.com/dabito/clyph/compare/v0.1.0-beta.2...HEAD
[0.1.0-beta.2]: https://github.com/dabito/clyph/compare/v0.1.0-beta.1...v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/dabito/clyph/releases/tag/v0.1.0-beta.1
