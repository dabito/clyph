# clyph roadmap

Status: v0.3.0 + unreleased `semantic`. clyph is the offline, scriptable, agent-friendly Nerd Fonts companion.
Goal: grow from a lookup CLI into the **visual-language layer** + a **build tool** for Nerd Fonts.

Legend: ✓ shipped · → next · ○ later · ? optional

## Pillar 1 — discover & resolve

| Feature | Status | Effort | Pitch |
|---|---|---|---|
| `search` (substring, paginated, normalized) | ✓ | — | find glyphs by name |
| `get` / `glyph` / `codepoint` | ✓ | — | exact lookup |
| `label` / `alias` curation | ✓ | — | annotate catalog |
| `identify` (reverse: glyph char → record) | ✓ | low | paste a box, learn its name |
| `fmt` (html / css / unicode / js / hex) | ✓ | low | one name → code for any context |
| `semantic` (concept → ranked glyphs) | → | med | ask for "success", get the icon |
| fuzzy search | ○ | med | typo-tolerant discovery |

## Pillar 2 — build tool

| Feature | Status | Effort | Pitch |
|---|---|---|---|
| `export` (ts / go / css / json / html sheet) | ○ | med | generate typed icon modules |
| `set` (curated, versioned collections) | ○ | med | ship a status-bar / file-type kit |
| `check` (validate glyph refs in configs) | ○ | med | catch dead refs in CI |
| `diff` (catalog changes vs last update) | ○ | med | track upstream Nerd Fonts changes |
| `subset` (strip font to used glyphs) | ? | high | shrink Nerd Font 90% for web/app |

## Pillar 3 — browse

| Feature | Status | Effort | Pitch |
|---|---|---|---|
| `sheet` / `preview` (terminal grid) | ○ | low | browse glyphs offline |
| `browse` (fzf-style interactive) | ○ | low | visual picker |

## Phasing

- **Phase A (0.3.0)** — `identify` + `fmt` + `families` + `stats`. Quick wins, self-contained, high punch.
- **Phase B (next)** — `semantic` + curated concept seed. Builds on `label`/`alias`.
- **Phase C** — `export` + `set`. Turns clyph into a build tool.
- **Phase D** — `check` + `diff`. CI / observability.
- **Phase E (later)** — `sheet`/`browse`; evaluate `subset` on demand.

## Constraints

- `fmt` covers codepoint-derived formats only. SVG / glyph-shape export needs font
  outline data (not in the CSS catalog) — deferred until `subset` or a font-file source lands.
- `semantic` depends on a curated seed shipped under `data/`; user `label`/`alias` extend it.
- All features must keep deterministic text + `--json` output for agent use.
