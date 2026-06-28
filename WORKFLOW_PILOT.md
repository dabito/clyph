# Workflow Pilot: clyph

## Why clyph first

`clyph` is the smallest safe pressure test for the agentic workflow:

- standalone supporting CLI, not a pi extension,
- deterministic shell-first behavior,
- small PRD and narrow command menu,
- useful immediately for visual language work,
- low risk if the workflow reveals friction.

## Coordinator task packet

```text
goal: implement clyph MVP as a deterministic Nerd Font glyph lookup CLI
repo: /Users/dabito/Workspace/Lab
project: clyph
worktree root: ~/Workspace/Lab/worktrees/clyph
branch: agent/clyph/001-cli-mvp
acceptance:
  - clyph search <query> returns concise matches
  - clyph get <name> returns metadata for one glyph
  - clyph glyph <name> prints only the glyph
  - clyph codepoint <name> prints only the codepoint
  - --json returns stable JSON
  - catalog can be generated/refreshed from Nerd Fonts webfont.css
constraints:
  - shell-first and deterministic
  - no pi- prefix
  - no pi extension in this slice
  - no network dependency for normal lookup after catalog exists
  - concise human output; structured JSON for agents/extensions
```

## Pilot workflow

### 1. Coordinator frames task

Coordinator creates the task packet and worktree/branch.

Suggested setup:

```bash
mkdir -p ~/Workspace/Lab/worktrees/clyph
git worktree add ~/Workspace/Lab/worktrees/clyph/001-cli-mvp -b agent/clyph/001-cli-mvp
```

If the Lab folder is not a git repo yet, initialize or move clyph into the repo where implementation should live before spawning agents.

### 2. Background investigator

Prompt:

```text
You are the investigator for clyph. Do not code.
Read clyph/PRD.md and inspect the repository shape.
Find the smallest implementation approach for a deterministic CLI that parses Nerd Fonts webfont.css into a local catalog and supports search/get/glyph/codepoint with --json.
Return relevant files, proposed project layout, dependencies, risks, and diagnostics.
```

Expected deliverable:

```text
summary:
recommended language/runtime:
project layout:
catalog extraction approach:
CLI command plan:
risks:
diagnostics:
```

### 3. Background adversary reviews investigation

Prompt:

```text
You are the adversary reviewer for the clyph investigation.
Challenge the proposed implementation before coding.
Focus on determinism, dependency risk, CSS parsing correctness, output stability, and whether the MVP is too large.
Return blockers, simpler alternatives, and recommendations for the coder.
```

### 4. Background coder implements

Prompt:

```text
You are the coder for clyph.
Work only in the assigned worktree and branch.
Implement the smallest MVP from clyph/PRD.md and the reviewed investigation.
Favor simple code, deterministic shell behavior, stable JSON, and concise output.
Do not implement a pi extension.
Return changed files, commands run, and remaining concerns.
```

### 5. Diagnostics

Because clyph is a supporting CLI, diagnostics should be ordinary shell commands.

Minimum expected checks depend on chosen runtime, but should include:

```text
format
lint or static check
tests or golden command-output checks
sample CLI invocations
```

Once `taste` exists, this becomes:

```bash
taste gate --changed --json
```

### 6. Implementation adversary review

Prompt:

```text
You are the adversary reviewer for clyph implementation.
Review the diff, CLI behavior, tests, and output shape.
Block on nondeterminism, excessive dependencies, unstable JSON, noisy output, missing fallback catalog behavior, or scope creep.
Return merge/revise/block with concrete issues.
```

### 7. Coordinator review and merge

Coordinator checks:

- diff is small,
- CLI matches PRD,
- output is concise,
- JSON is stable,
- diagnostics passed,
- adversary has no blockers.

## Council need?

For clyph MVP, skip full planning council unless investigation reveals ambiguity. This is a coding task, not an architecture task.

Use a council only if we need to decide between implementation languages, catalog sources, or package boundaries.

## Suggested first implementation slice

Prefer a tiny catalog fixture first, then real CSS extraction:

1. Create CLI skeleton.
2. Add static fixture catalog with 5-10 glyphs.
3. Implement `search`, `get`, `glyph`, `codepoint`, `--json` against fixture.
4. Add parser/update command for `webfont.css`.
5. Add tests/golden outputs.

This reduces risk and gives reviewers something deterministic early.

## What this pressure-tests

- Whether coordinator packets are specific enough.
- Whether investigator output saves coder time.
- Whether adversary review catches scope creep.
- Whether worktrees keep background work isolated.
- Whether our human/agent surface principles produce clearer CLI output.
- Whether a supporting non-`pi-` CLI plus future pi wrapper feels right.
