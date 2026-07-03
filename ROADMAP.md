# Roadmap

`clyph` is intentionally small: one Go binary, stdlib-only, local-first catalog.

## Known issues (from 2026-07-02 hands-on critique)

Found by building the binary and running every command/failure path directly, not just reading the code.

### Should fix (done 2026-07-02)

- ~~`--json` errors aren't JSON.~~ Fixed via `printError` in `cli.go` — every command now emits `{"error": "..."}` on stderr when `--json` is set.
- ~~Unreachable "match everything" branch.~~ Removed; `parseSearchArgs` already guarantees a non-empty query.
- ~~Space vs. underscore mismatch fails silently.~~ `searchRecords` now normalizes underscores/spaces on both the query and catalog text (`normalizeSearchText` in `catalog.go`).
- ~~`label`/`aliases` have no assignment path.~~ Added `clyph label <name> <text>` / `clyph label <name> --clear` and `clyph alias <name> add|rm <value>`.
- ~~Missing-catalog error leaks a raw Go error.~~ `loadRecords` now returns `catalog not found at <path> — run 'clyph update' first`.

### Fixed since (2026-07-02)

- ~~Tab-separated `search` output drifts out of alignment for long/irregular names.~~ Added `clyph search --pretty` for space-padded columns (`formatRowsPretty` in `catalog.go`); default tab-separated output is unchanged for scripts.
- ~~`search` truncates matches silently.~~ Default `--limit` raised from 10 to 100; `searchRecords` now also returns the pre-limit total, plain output prints a `showing N of TOTAL matches` notice on stderr when truncated, and `--json` carries a `total` field.
- ~~`--limit 0` returns 1 result instead of 0.~~ Fixed as a side effect of the truncation-total refactor — `len(matches) < limit` is now the append condition instead of `>=`-after-append.
- ~~No way to page past `--limit`.~~ Added `clyph search --offset N`; `searchRecords` now takes `(limit, offset)` and the truncation notice reports the shown page range.
- ~~Support `--help`/`-h` on subcommands.~~ `run()` checks every subcommand's `rest` args for `-h`/`--help` against a `commandUsage` map before dispatch.

### Minor polish

- Support `--limit=5` (equals form), not just `--limit 5` (space form).
- Omit empty `label`/`aliases: []` from JSON output (`omitempty`) to cut noise/tokens on the common case.

## Deferred: GoReleaser

Not adding yet — single binary, `go install` covers the install path, and there's no cross-platform artifact need yet. Revisit if we want GitHub Releases with prebuilt binaries/checksums for non-Go-toolchain users.
