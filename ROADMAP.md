# Roadmap

`clyph` is intentionally small: one Go binary, stdlib-only, local-first catalog.

## Known issues (from 2026-07-03 hands-on critique)

Found by building the binary and running every command/failure path directly, not just reading the code.

### Should fix (done 2026-07-03)

- ~~`--json` errors aren't JSON.~~ Fixed via `printError` in `cli.go` — every command now emits `{"error": "..."}` on stderr when `--json` is set.
- ~~Unreachable "match everything" branch.~~ Removed; `parseSearchArgs` already guarantees a non-empty query.
- ~~Space vs. underscore mismatch fails silently.~~ `searchRecords` now normalizes underscores/spaces on both the query and catalog text (`normalizeSearchText` in `catalog.go`).
- ~~`label`/`aliases` have no assignment path.~~ Added `clyph label <name> <text>` / `clyph label <name> --clear` and `clyph alias <name> add|rm <value>`.
- ~~Missing-catalog error leaks a raw Go error.~~ `loadRecords` now returns `catalog not found at <path> — run 'clyph update' first`.

### Minor polish

- Support `--limit=5` (equals form), not just `--limit 5` (space form).
- Support `--help`/`-h` on subcommands (`clyph search --help`), not just top-level.
- Omit empty `label`/`aliases: []` from JSON output (`omitempty`) to cut noise/tokens on the common case.
- `--limit 0` currently returns 1 result instead of 0 (documented in README as intentional off-by-one, but worth just fixing instead of documenting around).

## Deferred: GoReleaser

Not adding yet — single binary, `go install` covers the install path, and there's no cross-platform artifact need yet. Revisit if we want GitHub Releases with prebuilt binaries/checksums for non-Go-toolchain users.
