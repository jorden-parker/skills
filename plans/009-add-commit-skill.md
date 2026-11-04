# Plan 009: Add a `commit` skill that plans Conventional Commits as an HTML page

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update this plan's row in
> `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- skills/pr-draft/ skills/geist/ CONTEXT.md package.json`.
> Changes under `skills/pr-draft/` mean the "Reuse" notes below may be stale;
> re-read `skills/pr-draft/preview` and `assets/preview.html` before step 3.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (the skill never writes to the repository's history or index)
- **Depends on**: none
- **Category**: feature
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Goal

`/commit` reads the working tree, sorts every change into an ordered list of
Conventional Commits, and opens a local HTML page. Each commit appears as a
card with copyable code blocks. The user runs the commands. The skill
**never** stages, commits or pushes; this matches `pr-draft` and the global
rule "commit only when the user asks".

## Design decisions

| Decision                          | Choice                                                                                                          | Why                                                                                                              |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Who executes                      | The user copies the commands                                                                                    | Read-only like `pr-draft`. The page _is_ the review step.                                                        |
| Agent → renderer interface        | The agent writes `plan.json`; a bundled `render` script turns it into HTML                                      | Structured input can be validated. Markdown cannot.                                                              |
| Commit messages in commands       | Each message goes to `msg-NN.txt`, and the command is `git commit -F <path>`                                    | Avoids shell quoting entirely. The user's login shell is fish, which has no heredocs and different `'` escaping. |
| Splitting one file across commits | File-level grouping by default. A hunk-level split writes `patch-NN.diff` and uses `git apply --cached <path>`  | Most changes split cleanly by file. Hunk splits stay possible without interactive `git add -p`.                  |
| Correctness check                 | `render` replays the whole plan in a **temporary index** and checks that the final tree equals the working tree | Catches files left out, paths listed twice, and patches that don't apply, before the user sees the page.         |
| Styling                           | Geist tokens from `skills/geist/` and the copy-button pattern from `pr-draft/assets/preview.html`               | Matches the sibling preview. No Markdown, so no marked/DOMPurify: every value is `html.escape`d.                 |
| Files that shouldn't be tracked   | The first commit adds them to `.gitignore` (and untracks any already tracked) before any other commit           | Every later `git add -A` then skips them, and the history never contains them, even briefly.                     |
| Browser                           | Opens by default; `--no-open` for tests                                                                         | Plan 006 gives `pr-draft`'s preview the same flag.                                                               |
| Storage                           | `$(git rev-parse --absolute-git-dir)/commit-plan/`                                                              | Per worktree, outside the tree, never committed. Overwritten on each run.                                        |

## Files

```
skills/commit/
  SKILL.md                 agent instructions (step 2)
  render                   python3 executable: validate + dry-run + write HTML + open (step 3)
  assets/page.html         HTML template with {{PLACEHOLDERS}} (step 3)
test/commit/
  test_render.py           unittest suite, run against temp git repos (step 1)
```

Tests live outside `skills/commit/` because installs link the skill directory
and the tests shouldn't ship with it.

### `plan.json` schema

```json
{
  "base": "HEAD sha the plan was made against",
  "ignore": {
    "patterns": [".env.local", ".playwright-mcp/"],
    "untrack": [],
    "reasons": {
      ".env.local": "local secrets",
      ".playwright-mcp/": "browser tool output"
    },
    "subject": "ignore local env files and browser tool output"
  },
  "commits": [
    {
      "type": "feat",
      "scope": "cli",
      "breaking": false,
      "subject": "add --dry-run flag",
      "body": "Why the change was made, wrapped at 72.",
      "footers": ["Refs: #42"],
      "paths": ["internal/cli/run.go", "internal/cli/run_test.go"],
      "patch": null,
      "rationale": "One line shown on the card: why these files belong together."
    }
  ],
  "excluded": [{ "path": ".env.local", "reason": "looks like a secret" }],
  "notes": [
    "Index had staged changes; step 0 unstages them (working tree untouched)."
  ]
}
```

`patch` is a path to a diff file, used in place of `paths` when the commit
takes only some hunks of a file.

`ignore` is optional. When present, `render` turns it into the first commit,
`chore: <subject>`, before every entry in `commits`. `untrack` lists paths
that are tracked today and match a new pattern; they get
`git rm -r --cached`, which keeps the files on disk. `excluded` is now only
for changes that shouldn't be ignored but also shouldn't be committed yet,
such as a token pasted into a tracked config file.

## Steps

### 1. Tests first (TDD; tdd-guard requires it)

Write `test/commit/test_render.py` using `unittest`. Each test builds a
throwaway repo in `tempfile.TemporaryDirectory()` and runs
`skills/commit/render --no-open`. Cases:

1. **Happy path**: two files in two commits. Exit 0. The HTML has two cards
   with headers `feat(cli): …` and `test: …`, and `msg-01.txt` holds the full
   message.
2. **Coverage gap**: a modified file appears in no commit and not in
   `excluded`. Non-zero exit; stderr names the file.
3. **Duplicate path**: the same file in two commits without `patch`.
   Non-zero exit.
4. **Bad header**: an unknown type, a subject over 72 chars or ending in `.`,
   or `breaking: true` without `!`/`BREAKING CHANGE:` rendering. Rejected,
   or rendered correctly (for `!`).
5. **Hunk split**: one file, two patches applied in sequence. The dry-run
   passes and the final tree matches.
6. **Patch fails**: a patch doesn't apply to the temporary index. Non-zero
   exit; the real `.git/index` is byte-identical before and after.
7. **Escaping**: a subject containing `<script>` comes out escaped.
8. **Stale base**: `base` ≠ current `HEAD`. Non-zero exit.
9. **Untracked + deleted + renamed** files are covered correctly
   (`git add -A -- <paths>` semantics).
10. **Read-only guarantee**: after any run, `git status --porcelain`, `HEAD`,
    the index checksum and the list of files under `.git/objects` are unchanged.
11. **Ignore step comes first**: with `ignore` set, commit 1 is
    `chore: ignore …`, its commands append the patterns and stage only
    `.gitignore`, and the ignored untracked files need no entry in `commits`.
12. **No trailing newline**: when `.gitignore` doesn't end in a newline, the
    append command starts with an empty line so the last pattern isn't joined
    to the first new one. When `.gitignore` doesn't exist, the command
    creates it.
13. **Already tracked**: a tracked file matching a new pattern is in
    `untrack`, gets `git rm -r --cached`, and the dry-run's final tree no
    longer contains it. A tracked match missing from `untrack` is an error.
14. **Duplicate pattern**: a pattern already in `.gitignore` is rejected.

Add `"test": "python3 -m unittest discover -s test -p 'test_*.py'"` to
`package.json` scripts. Do not overwrite a `test` script plan 006 may have
added; chain onto it instead.

**Verify**: `pnpm test`. Every test fails because `render` doesn't exist yet.

### 2. `SKILL.md`

Frontmatter:

```yaml
---
name: commit
description: Organise the working tree's changes into Conventional Commits and open them as copyable commands on a local HTML page. Makes no Git changes.
argument-hint: "optional: hints on grouping, scope names, or an issue to reference"
disable-model-invocation: true
---
```

Sections (imperative; follow `writing-for-agents` style; no attribution lines
in messages):

1. **Gather.** `git status --porcelain=v2 --untracked-files=all`,
   `git diff HEAD --stat`, `git diff HEAD`, then read untracked files. If
   nothing changed, say so and stop. Record `git rev-parse HEAD` as `base`.
2. **Learn the repo's convention.** Look for commitlint config
   (`commitlint.config.*`, `.commitlintrc*`, `package.json#commitlint`) and
   read its `type-enum`, `scope-enum` and `header-max-length`. Read
   `git log --format=%s -30` for scope habits and casing. With nothing to go
   on, use the Conventional Commits defaults: `feat fix refactor perf test
docs build ci chore style revert`.
3. **Ignore first.** Before grouping, look for untracked or changed files
   that no repository should track: build output, caches, dependency folders,
   logs, editor and OS files (`.DS_Store`, `.idea/`), local env files, and
   output from tools run in this session (`.playwright-mcp/`). Read the
   existing `.gitignore` and don't repeat a pattern it already has. Prefer a
   folder or glob pattern (`dist/`, `*.log`) over listing single files. Put
   the patterns, with a short reason each, in `ignore`. If a tracked file
   matches, add it to `untrack` and say so in the report, because the user's
   next push removes it from the remote. Leave out anything a teammate might
   rely on, such as a shared editor config that is already committed.
4. **Group.** One logical change per commit. Rules:
   - Tests go in the same commit as the code they test (TDD repo norm), not a
     separate `test:` commit, unless the change is test-only.
   - Lockfiles go with the manifest change that caused them.
   - Generated files go with their source.
   - Pure formatting or renames come before behaviour changes, as their own
     `style:` or `refactor:` commit.
   - Order commits so every prefix of the list builds: dependencies first.
   - Split within a file only when it clearly holds two unrelated changes.
     Then write `patch-NN.diff` with `git diff` restricted to the hunks,
     relative to the state after the previous commits.
   - Prefer fewer, coherent commits over many tiny ones. If unsure, merge.
5. **Exclude.** A change that shouldn't be committed but also shouldn't be
   ignored, such as a secret pasted into a tracked file. List each in
   `excluded` with a reason; never put a secret's content in the plan.
6. **Write `plan.json`** to the plan directory, replacing the old one.
7. **Render**: `<skill-directory>/render "$PLAN_DIR/plan.json"`. If it fails
   validation, fix the plan and re-run. Never hand the user a plan that
   failed the dry-run.
8. **Report**: the HTML path, the number of commits with their headers, the
   new ignore patterns, any files being untracked, the excluded files, and any notes (e.g. staged changes). Stop. The user runs
   the commands.

### 3. `render` and `assets/page.html`

`render` (python3, standard library only; small single-purpose functions per
the preferences):

- `load_plan(path)`: parse the JSON and check the schema.
- `validate_headers(commits, convention)`: Conventional Commits regex
  `^(?P<type>[a-z]+)(\((?P<scope>[^)]+)\))?(?P<bang>!)?: (?P<subject>.+)$`,
  plus the commitlint limits if a config is found. Only read
  JSON/`package.json` configs; for JS configs fall back to the defaults and
  say so in stderr.
- `changed_paths(repo)`: parse `git status --porcelain=v2 -z`, handling
  renames as both paths.
- `check_coverage(plan, changed)`: every changed path is in exactly one
  commit, a patch, `excluded`, or matched by a new `ignore` pattern. Test
  matches with `git -c core.excludesFile=<temp file of new patterns>
check-ignore --no-index`, so the real `.gitignore` is never edited.
- `ignore_commands(ignore, repo)`: `printf '%s\n' <quoted patterns> >> .gitignore`
  (works in fish, bash and zsh; starts with `''` when the file lacks a final
  newline), then `git rm -r --cached --quiet -- <untrack>` if any, then
  `git add .gitignore` and `git commit -F <msg-01 path>`. If `.gitignore`
  already had uncommitted edits, they go into this commit too; add a note.
- `dry_run(repo, plan)`: copy the index to a temp file and set
  `GIT_INDEX_FILE`. Set `GIT_OBJECT_DIRECTORY` to a temp folder and
  `GIT_ALTERNATE_OBJECT_DIRECTORIES` to the real one, so the blobs that
  `git add` and `hash-object` write never land in `.git/objects`. Run
  `git reset -q` against it. If `ignore` is set, stage the new `.gitignore`
  with `hash-object -w --stdin` and `update-index --cacheinfo`, and remove
  `untrack` with `git rm -r --cached`. Then per commit
  `git add -A -- <paths>` or `git apply --cached <patch>`, then
  `git write-tree`. Finally compare against a temp index built with
  `git add -A` under the same `core.excludesFile`, with the new `.gitignore`
  blob and without the excluded paths. Never touch the real index or object
  store.
- `write_artifacts(plan_dir, plan)`: writes `msg-NN.txt` (header, blank line,
  body, blank line, footers).
- `commands_for(commit, plan_dir)`: builds
  `git add -A -- <shell-quoted paths>` or `git apply --cached <patch>`,
  followed by `git commit -F <msg path>`. Quote paths with `shlex.quote`;
  these quotes work in both POSIX shells and fish for ordinary paths. Reject
  paths containing `'`, a newline or a backslash with a clear error, since
  quoting them differs between shells.
- `render_html(plan, convention)`: fills `page.html`.
- `main()`: flags `plan` (positional), `--output` (default `<plan dir>/index.html`),
  `--no-open`. Prints the absolute HTML path. Opens it with
  `webbrowser.open`; if that fails, prints the path and exits 0.

`page.html` layout follows the approved prototype,
`plans/009-commit-page-prototype.html`. Use its CSS and markup as the
starting point; rewrite the JavaScript as Python string building in `render`,
because the real page needs no script except copy and progress.

- A slim sticky top bar: step progress ("5 steps to run", then "2 of 5 steps
  copied"), a light/dark toggle, and **Copy all commands** (every step, in
  order).
- A title, "N commits for <repository>", and one sentence saying nothing has
  been staged or committed yet.
- A vertical commit graph in one left-aligned column, in run order:
  1. A square node for the current commit (short SHA and subject).
  2. "Before you start" (`git reset -q`), only when the index had staged
     changes.
  3. The ignore commit, when present: each new pattern with its reason and
     the files it now hides; untracked paths are marked "kept on disk,
     removed from Git".
  4. One entry per commit: "Commit n of N", the Conventional header in mono
     with only the type word coloured, the message body, the reason in
     smaller grey sans text, the file list with `+/-` counts (omit `−0`),
     the commands block with a corner Copy button, and a "Message only, for
     a Git app" disclosure.
  5. A dashed end node, "Working tree clean", that fills green once every
     step has been copied.
- Node colour encodes commit type (Geist blue for feat, red for fix, purple
  for refactor, teal for docs, amber for the reset step, grey otherwise).
  It is the only colour on the page apart from the `+/-` counts.
- Copying a step fills its node and updates the progress text.
- `git add` with several paths puts one path per line with a trailing ` \`.
- A "Left out" section for `excluded`, hidden when empty.
- Inline the Geist CSS from `skills/geist/` into the page the same way
  `pr-draft/preview` does, so the page renders from any location. The
  prototype's first version linked the CSS by relative path and rendered
  unstyled in the T3 Code file viewer. Read `skills/geist/SKILL.md` first.

**Verify**: `pnpm test` all green; `npx prettier --check skills/commit test/commit`.

### 4. Repository docs

- `CONTEXT.md`: add a **Commit Plan** term under a new "Commits" heading:
  "The ordered list of Conventional Commits the `commit` skill proposes for a
  working tree, stored as `plan.json` in the Git directory. The user applies
  it; the skill never does." _Avoid_: commit draft, staging plan.
- `README.md`: add the skill to the skill list, if the README has one.
- `plans/README.md`: add a row for 009.

### 5. Install and smoke-test

```sh
go build -o bin/skills . && ./bin/skills --repo . add commit
./bin/skills --repo . refresh commit && ./bin/skills --repo . refresh --check commit
```

In a new Claude Code session, in this repo with a few mixed edits, run
`/commit`. Check that the page opens, every Copy button works (use
`/cmux-browser`, and read the console for errors), and that running "Copy
all" in a scratch clone produces the planned commits and passes the repo's
commitlint hook.

## STOP conditions

- The dry-run can't be made to leave `.git/index` untouched (test 6 or 10
  fails for reasons other than a bug in `render`).
- `skills add commit` reports a name collision with an existing installed
  `commit` skill or plugin command.
- A need arises to execute commits from the skill. That reverses the
  read-only design; ask the user.

## Out of scope (YAGNI)

- An "apply" mode that runs the commands.
- Interactive regrouping on the page (drag cards around). Re-run `/commit`
  with a hint instead.
- Reading JS commitlint configs by executing them.
- Signing, amend or fixup commits.
