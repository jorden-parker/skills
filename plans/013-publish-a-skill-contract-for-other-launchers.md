# Plan 013: Publish a skill contract file that other launchers (claude-ollama) can filter on

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- skills/ upstream-skills.json README.md docs/adr main.go`
> New or removed skill directories since this plan was written are expected
> drift. Step 2's data table must then gain or lose rows; that is not a STOP.
> Any other mismatch with "Current state" is a STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (adds a data file and a test; changes no runtime behaviour)
- **Depends on**: none. It pairs with 010, which stops link clobbering.
- **Category**: direction (cross-repo alignment)
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Why this matters

This repo's skills reach a second consumer it knows nothing about.
`~/src/claude-custom` ships `claude-ollama`, which runs Claude Code on small
local models. That session has **no Agent tool and no web** (its tool list is
`Bash,Read,Edit,Write,Skill,TaskStop`, `claude-custom/bin/claude-ollama:16`).
At launch it copies `~/.claude/skills` with `cp -RL`
(`bin/claude-ollama:519-523`). It then adds its own vendored skills only where
a name is still free (`:525-531`). claude-custom deliberately leaves out
skills that "need subagents or the web" (`code-review`, `research`, …;
`claude-custom/CLAUDE.md:965-968`), but only by **name**, in prose. Any of
this repo's skills linked into `~/.claude/skills` gets past that list. On
this machine, `skills sync` would link about 30 of them at once.

The two repos should stay separate. Neither should import the other's code or
read its internals. The seam is a **published fact sheet**: this repo states
what each skill needs (subagents, web, gh, …) and which other skills it calls.
claude-custom keeps its own policy and decides what to drop. Each repo then
owns what it knows best: this repo knows its skills, and claude-custom knows
its runtime.

This plan builds this repo's half: the file, its schema, a test that keeps it
complete, and an ADR. It also adds the name/description frontmatter checks
this repo lacks; claude-custom has them in `bin/check-config`. The consumer
side is a separate change in claude-custom (see Maintenance notes).

## Current state

- `skills/<name>/SKILL.md`: 34 skills, each with YAML frontmatter. `name:` equals the folder name for all of them today, and every `description:` is one line of at most 506 characters. Claude Code cuts descriptions at 1536 characters.
- `upstream-skills.json`: records where vendored skills came from (`sources[].repository`, `.commit`, and `.skills` mapping name → upstream path). It is the existing pattern for a repo-root JSON manifest. Read it and match its 2-space indentation and key style.
- Nothing reads `upstream-skills.json` today except the README.
- `main.go` is `package main` at the repo root. A `_test.go` file in that package runs with the repo root as its working directory under `go test ./...`, so it can read `skills/` and the manifests by relative path.
- No skill-content validation exists in Go. `internal/skills/repo.go` `Describe` (line 76) reads the first `description:` line naively.
- The YAML library is not a dependency. Check with `grep -n yaml go.mod`; `go.sum` may list `gopkg.in/yaml.v3` as indirect. **Do not add a YAML dependency.** Parse frontmatter the way `Describe` does: the lines between the first two `---` lines, split on the first `:`, value trimmed, surrounding `'` or `"` quotes removed.

Vocabulary (`CONTEXT.md`): **Skill**, **Skills Directory**, **Repo**. Don't introduce "plugin" or "command" for a skill.

Conventions: there are no code comments (user hooks reject them), apart from doc comments on exported Go identifiers. Tests use plain `t.Fatalf`. JSON files are formatted by prettier (`.lintstagedrc` runs `prettier --ignore-unknown --write` on every staged file).

## The contract (implement exactly this)

Create `skill-requirements.json` at the repo root:

```json
{
  "vocabulary": {
    "subagents": "Starts sub-agents or background agents (the Agent/Task tool)",
    "web": "Searches or fetches the live web",
    "gh": "Needs the GitHub CLI",
    "network": "Installs packages or otherwise needs network access",
    "python": "Runs a bundled Python 3 script",
    "node-modules": "Needs this repo's pnpm install (marked, dompurify)",
    "browser": "Opens a page in a browser"
  },
  "skills": {
    "code-review": { "requires": ["subagents"], "uses": [] }
  }
}
```

Rules, enforced by the test in step 3:

- `skills` has exactly one entry per directory in `skills/` that has a `SKILL.md`. No extras and none missing.
- Each `requires` item is a key of `vocabulary`.
- Each `uses` item is the name of another skill in `skills/`.
- `requires` lists **hard** needs only: the skill's main path can't work without them. A skill that says "if you can, use subagents" has no `subagents` entry.

## Commands you will need

| Purpose       | Command                                              | Expected                                                                         |
| ------------- | ---------------------------------------------------- | -------------------------------------------------------------------------------- |
| Tests         | `go test ./...`                                      | all `ok`                                                                         |
| Contract test | `go test . -run TestSkillContract -v`                | `PASS`                                                                           |
| JSON format   | `pnpm exec prettier --check skill-requirements.json` | exit 0 (run `pnpm install --frozen-lockfile` first if `node_modules` is missing) |
| Vet / format  | `go vet ./... && gofmt -l .`                         | exit 0, no output                                                                |

## Scope

**In scope**:

- `skill-requirements.json` (create)
- `contract_test.go` (create, `package main`, repo root)
- `docs/adr/0006-publish-skill-requirements-for-other-launchers.md` (create; use 0005 if plan 010 has not created 0005, and adjust the link in step 4)
- `README.md` (one short section)

**Out of scope**:

- Any `SKILL.md` or other file under `skills/`. Do not add frontmatter keys to vendored skills; that would create upstream drift.
- `upstream-skills.json`. Don't merge the two files; one records provenance, the other records needs.
- A new `skills` subcommand. The JSON file is the contract; YAGNI.
- Anything in `~/src/claude-custom`.

## Git workflow

Branch `advisor/013-skill-contract`. Conventional Commits, e.g. `feat: publish skill requirements for other launchers`. End each message with the operator's attribution trailer. Don't push or open a PR unless told to.

## Steps

### Step 1: Write the failing contract test

Create `contract_test.go` (`package main`) with `TestSkillContract`:

1. List `skills/*/SKILL.md` and collect the folder names.
2. For each skill, parse its frontmatter (see "Current state" for how) and check two things. `name` must equal the folder name. `description` must be non-empty, contain no newline, and be at most 1536 characters. Report every failure with the skill name; don't stop at the first.
3. Read `skill-requirements.json` into a struct `{Vocabulary map[string]string; Skills map[string]struct{Requires, Uses []string}}`. Use JSON tags `vocabulary`, `skills`, `requires`, `uses`. Check the rules above, and report every violation.

**Verify**: `go test . -run TestSkillContract -v` → FAIL, because `skill-requirements.json` doesn't exist yet. The frontmatter part should already pass; if it doesn't, STOP and report the failing skills.

### Step 2: Write the data

Fill `skill-requirements.json` from this table. It came from reading each skill; evidence is given for every non-empty row. Check each evidence line with `sed -n '<line>p' skills/<path>` before you trust it. If a line no longer says what's claimed, drop that requirement and note it in your report.

| Skill                         | requires                           | uses                      | Evidence                                                                                                  |
| ----------------------------- | ---------------------------------- | ------------------------- | --------------------------------------------------------------------------------------------------------- |
| ask-matt                      |                                    |                           |                                                                                                           |
| code-review                   | subagents                          |                           | `code-review/SKILL.md:11` "parallel sub-agents"                                                           |
| codebase-design               |                                    |                           | the sub-agents are only in an optional sub-flow, `DESIGN-IT-TWICE.md:21`                                  |
| diagnosing-bugs               |                                    |                           |                                                                                                           |
| domain-modeling               |                                    |                           |                                                                                                           |
| geist                         |                                    |                           |                                                                                                           |
| git-guardrails-claude-code    |                                    |                           |                                                                                                           |
| grill-me                      |                                    | grilling                  | `grill-me/SKILL.md:7`                                                                                     |
| grill-with-docs               |                                    | grilling, domain-modeling | `grill-with-docs/SKILL.md:7`                                                                              |
| grilling                      |                                    |                           | a sub-agent is suggested for facts (`:26`), but it isn't required                                         |
| handoff                       |                                    |                           |                                                                                                           |
| implement                     |                                    | tdd, code-review          | `implement/SKILL.md:9,13`                                                                                 |
| improve                       |                                    |                           | it falls back to auditing directly without subagents (`improve/SKILL.md:43`, see the next sentence there) |
| improve-codebase-architecture | subagents, browser                 | codebase-design, grilling | `:27` "spawn a sub-agent"; HTML report opened in a browser (`HTML-REPORT.md`)                             |
| migrate-to-shoehorn           | network                            |                           | package install                                                                                           |
| new-project                   | network                            |                           | `new-project/SKILL.md:94,108` `pnpm add`/`create`                                                         |
| pr-draft                      | python, node-modules, browser      |                           | `pr-draft/preview` script, `preview:51-52`                                                                |
| prototype                     |                                    |                           |                                                                                                           |
| research                      | subagents, web                     |                           | `research/SKILL.md:6` "background agent"                                                                  |
| resolving-merge-conflicts     |                                    |                           |                                                                                                           |
| scaffold-exercises            | network                            |                           |                                                                                                           |
| setup-matt-pocock-skills      |                                    |                           |                                                                                                           |
| setup-pre-commit              | network                            |                           | package install                                                                                           |
| tdd                           |                                    |                           |                                                                                                           |
| teach                         |                                    |                           |                                                                                                           |
| to-questionnaire              |                                    |                           |                                                                                                           |
| to-spec                       |                                    |                           |                                                                                                           |
| to-tickets                    |                                    |                           |                                                                                                           |
| triage                        |                                    |                           |                                                                                                           |
| wait-what                     |                                    |                           |                                                                                                           |
| wayfinder                     | subagents                          | research                  | `wayfinder/SKILL.md:115`                                                                                  |
| web-research                  | web, python, node-modules, browser |                           | `web-research/SKILL.md` steps 2–6; `artifact` → `../../artifact/render`                                   |
| wizard                        | gh, browser                        |                           |                                                                                                           |
| writing-for-agents            |                                    |                           |                                                                                                           |

If `ls skills` shows a skill that isn't in this table, add it with empty arrays. Then read its `SKILL.md` for the words "sub-agent", "subagent", "Agent tool", "WebFetch", "WebSearch", "gh " and "pnpm add", and add the matching requirements. List each such skill in your report.

Order the `skills` keys alphabetically. Format with `pnpm exec prettier --write skill-requirements.json`.

**Verify**: `go test . -run TestSkillContract -v` → PASS. `pnpm exec prettier --check skill-requirements.json` → exit 0.

### Step 3: Prove the test guards the rules

Make three temporary edits to `skill-requirements.json`, one at a time, running the test after each and expecting FAIL:

1. Remove the `geist` entry.
2. Add `"cloud"` to `geist.requires`.
3. Add `"nope"` to `geist.uses`.

Then restore the file with `git checkout -- skill-requirements.json`, or rewrite it if it isn't committed yet.

**Verify**: each of the three runs FAILs with a message naming `geist`. The final run PASSes.

### Step 4: Record the decision and document the file

Create the ADR (number per Scope), in the prose style of `docs/adr/0004-port-install-and-refresh-to-go.md`:

- Context: other launchers (claude-custom's `claude-ollama`) copy installed skills into sessions that lack subagents or the web. Filtering by name in the other repo doesn't scale and has already leaked.
- Decision: this repo publishes `skill-requirements.json`, a fact sheet per skill. Consumers apply their own policy. No code is shared, and nothing in `SKILL.md` changes, so vendored skills stay byte-identical to upstream.
- Consequences: adding a skill now means adding a row, and the contract test enforces it. The vocabulary is small on purpose; add a word only when a consumer needs it.

In `README.md`, after the "Upstream skills" section, add a `### Requirements` section of 3–5 sentences. It should say what the file is, show the vocabulary, say that `go test .` checks it, and say that other launchers may read it from `$SKILLS_REPO/skill-requirements.json` (default `~/src/skills`).

**Verify**: `pnpm exec prettier --check README.md docs/adr/` → exit 0. `go test ./...` → all `ok`.

## Test plan

Add one test, `TestSkillContract` in `contract_test.go`. It covers the name/folder match, the description shape and length, full coverage of the requirements file, the vocabulary, and the `uses` references. Step 3 shows it fails when it should.

## Done criteria

- [ ] `skill-requirements.json` exists and has one entry per skill directory
- [ ] `go test ./...` all `ok`, including `TestSkillContract`
- [ ] `pnpm exec prettier --check skill-requirements.json README.md docs/adr/` exit 0
- [ ] `git diff --stat -- skills/` is empty: no skill content changed
- [ ] The ADR exists and the README has a `### Requirements` section
- [ ] `plans/README.md` row 013 updated

## STOP conditions

- A skill's frontmatter fails the name or description check in step 1. Report it; don't edit vendored `SKILL.md` files to make the test pass.
- More than 3 evidence lines in step 2 don't say what the table claims. The table has drifted; report it rather than re-deriving the whole table.
- You feel the need to add a vocabulary word not listed here. Report the skill and the need instead.

## Maintenance notes

- **Consumer side, in `~/src/claude-custom`, not part of this plan.** In `bin/claude-ollama` `copy_user_skills`, after the copy, read `${SKILLS_REPO:-$HOME/src/skills}/skill-requirements.json` if it exists. For every copied skill whose original link resolved into that repo and whose `requires` contains `subagents` or `web`, delete it from the session copy. Then let `copy_vendored_skills` fill the name, as it already does for absent names. `bin/claude-ollama-doctor` can then size the listing for the real session set. That repo needs its own plan, written against its own conventions.
- When vendoring a new upstream skill, add its row. The contract test fails until you do.
- `requires` describes facts, not policy. Don't encode "unsafe for small models" here; that judgement belongs to the consumer.
