# Plan 015: Put every skill-script test under `test/`, keep `skills/` shippable, and document the layout

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat aeb7cfc..HEAD -- skills/jorden/commit test .gitignore .prettierignore README.md package.json`
> If any of those changed since this plan was written, compare the "Current
> state" excerpts below against the live files; on a mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `aeb7cfc`, 2026-11-05

## Why this matters

The repo was asked to be reorganised so related files are grouped and the
layout is easy to navigate. Most of it already is: the Go command is one
package per concern under `internal/`, with tests beside each package, and
skills are grouped as `skills/<author>/<name>/`. The one real inconsistency is
where tests for skill scripts live. They are meant to live in `test/<skill>/`
(run by `pnpm test`), but one file, `skills/jorden/commit/tests/test_plan.py`,
sits inside the skill. That file is broken in three ways:

1. No command runs it. `pnpm test` only discovers `test/`.
2. It needs pytest, which is not installed (`No module named pytest`).
3. `skills/<author>/<name>/` is symlinked into agent apps
   (`~/.claude/skills/commit` and others), so the test file and its
   `__pycache__` ship to every agent that loads the skill.

Three of its six cases test behaviour that nothing else covers. This plan
moves those three into the existing `test/commit/test_plan.py` in that file's
unittest style, deletes the stray directory, ignores a pnpm store that sits
untracked in the repo root, and updates the README's "Repo layout" section so
the convention is written down. No runtime behaviour changes.

## Current state

### Layout today (first-party, non-vendored)

```
main.go, internal/<pkg>/      Go command; *_test.go beside each package   (keep)
install.sh                    builds and installs the command             (keep)
artifact/render, page.html    shared Markdown-to-page renderer            (keep)
artifact/test.sh              renderer tests, 22 cases                    (keep)
test/commit/test_plan.py      unittest, 7 cases, run by pnpm test         (extend)
test/commit/test_render.py    unittest, run by pnpm test                  (keep)
test/pr_draft/test_preview.py unittest, run by pnpm test                  (keep)
skills/jorden/commit/tests/test_plan.py   pytest, run by nothing          (delete after porting)
skills/jorden/commit/tests/__pycache__/   untracked bytecode              (delete)
.pnpm-store/                  untracked pnpm store, not ignored           (ignore)
```

### `package.json` test script (do not change)

```json
"test": "python3 -m unittest discover -s test -p 'test_*.py'"
```

### The stray file: `skills/jorden/commit/tests/test_plan.py`

Tracked in Git (`git ls-files skills/jorden/commit/tests` prints it). Its six
cases and where each is already covered:

| Stray case (line)                                       | Already covered by                                                                 | Action |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------- | ------ |
| `test_gather_reports_clean_tree` (47)                   | nothing                                                                            | port   |
| `test_gather_includes_diff_and_untracked_contents` (53) | `test/commit/test_plan.py` `test_gather_shows_untracked_contents`                  | drop   |
| `test_build_writes_plan_and_page` (61)                  | `test_plan_lives_outside_the_git_dir`, `test_build_records_head_and_splits_header` | drop   |
| `test_build_rejects_invalid_header` (82)                | `test_bad_header_is_rejected_and_plan_unchanged`                                   | drop   |
| `test_build_rejects_no_open_flag` (90)                  | nothing                                                                            | port   |
| `test_render_rejects_no_open_flag` (96)                 | nothing                                                                            | port   |

The three cases to port, as they are today:

```python
# skills/jorden/commit/tests/test_plan.py:47-50
def test_gather_reports_clean_tree(repo, tmp_path):
    result = run_plan(repo, tmp_path, "gather")
    assert result.returncode == 0
    assert "nothing to commit" in result.stdout

# :90-93
def test_build_rejects_no_open_flag(repo, tmp_path):
    dirty(repo)
    result = run_plan(repo, tmp_path, "build", "--no-open", stdin=json.dumps({"commits": []}))
    assert result.returncode == 2

# :96-98
def test_render_rejects_no_open_flag():
    result = subprocess.run([sys.executable, str(RENDER_SCRIPT), "--no-open", "plan.json"], capture_output=True, text=True)
    assert result.returncode == 2
```

Behaviour confirmed at `aeb7cfc`:

- `skills/jorden/commit/plan:97` prints `nothing to commit` when `gather` runs
  on a clean tree.
- `skills/jorden/commit/render --no-open plan.json` exits 2 with
  `render: error: unrecognized arguments: --no-open` (argparse).
- `skills/jorden/commit/plan build --no-open` prints
  `plan: error: unrecognized arguments: --no-open` (argparse, exit 2).

### Exemplar to match: `test/commit/test_plan.py`

unittest classes, a `setUp` that builds a throwaway repo, helpers named for
what they do, no comments. Key parts:

```python
# test/commit/test_plan.py:8
PLAN = Path(__file__).resolve().parents[2] / "skills" / "jorden" / "commit" / "plan"

# :12-26  setUp makes self.repo with one commit of README.md, then writes an
#         UNTRACKED file a.txt, so the tree is dirty in every test by default.
        (self.repo / "a.txt").write_text("a\n", encoding="utf-8")

# :34-37
    def plan(self, *arguments, document=None):
        environment = {**os.environ, "SKILLS_TMPDIR": str(self.skills_tmpdir)}
        stdin = json.dumps(document) if document is not None else ""
        return subprocess.run([str(PLAN), *arguments], cwd=self.repo, input=stdin, capture_output=True, text=True, check=False, env=environment)

# :93-96
    def test_gather_shows_untracked_contents(self):
        result = self.plan("gather")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("### a.txt\na\n", result.stdout)
```

`test/commit/test_render.py:10` defines the render path the same way:
`RENDER = Path(__file__).resolve().parents[2] / "skills" / "jorden" / "commit" / "render"`.

### Ignore files today

`.gitignore`:

```
.DS_Store
*.local
node_modules
/bin/
/dist/
__pycache__/
*.pyc
```

`.prettierignore`:

```
node_modules
pnpm-lock.yaml
```

### README "Repo layout" section today (`README.md:132-141`)

````markdown
## Repo layout

```
skills/<author>/<name>/  the skills themselves, grouped by author
main.go, internal/       the skills command (Go + Charm)
install.sh               builds and installs the command
artifact/                the shared Markdown-to-page renderer
CONTEXT.md               glossary
docs/adr/                decisions worth remembering
```
````

### Repo rules that apply

- No code comments, including in tests (the maintainer's hooks reject them).
- Names say what they hold or do.
- Prettier formats Markdown, JSON and other files on commit (`.lintstagedrc`).

## Commands you will need

| Purpose          | Command                                                           | Expected on success                          |
| ---------------- | ----------------------------------------------------------------- | -------------------------------------------- |
| Python tests     | `pnpm test`                                                       | `Ran 34 tests` … `OK` (31 before step 1)     |
| One test file    | `python3 -m unittest discover -s test -p 'test_plan.py' -v`       | every line ends `ok`                         |
| Go build + tests | `go build ./... && go vet ./... && go test ./...`                 | every package `ok` or `[no test files]`      |
| Renderer tests   | `bash artifact/test.sh`                                           | `# all 22 tests passed`                      |
| Formatting       | `pnpm exec prettier --check README.md .gitignore .prettierignore` | `All matched files use Prettier code style!` |

## Scope

**In scope** (the only files you may modify):

- `.husky/pre-commit` (Step 0: one line)
- `test/commit/test_plan.py` (add three tests and one constant)
- `skills/jorden/commit/tests/` (delete the whole directory)
- `.gitignore`, `.prettierignore` (one line each)
- `README.md` (the "Repo layout" code block only)
- `plans/README.md` (your status row only)

**Out of scope** (do NOT touch):

- Anything under `internal/` or `main.go`. The Go layout is already one
  package per concern; moving it changes `go build -o bin/skills .` in
  `install.sh`, `AGENTS.md` and `README.md` for no gain.
- `internal/tui/app.go` (588 lines). Splitting it is tempting but plans
  010–012 edit the TUI; a split now would conflict with them.
- `artifact/` and `artifact/test.sh`. The symlink
  `skills/jorden/web-research/artifact -> ../../../artifact/render` and plan
  014 both depend on these paths.
- `package.json`. Its `test` script already discovers `test/`. Plan 014 owns
  changes to it.
- `skills/jorden/commit/plan`, `render`, `SKILL.md`, `assets/` — the skill
  itself does not change.
- Vendored skills under `skills/mattpocock/` and `skills/shadcn/`.
- The untracked `.pnpm-store/` and `.pytest_cache/` directories on disk: do
  not delete them, only ignore `.pnpm-store/`.

## Git workflow

The maintainer commits directly on `main` in this repo. Use Conventional
Commits (the `commit-msg` hook runs commitlint), matching `git log`, e.g.
`test(commit): cover plan gather and build; update docs`. Suggested commits:

0. `fix(hooks): keep pre-commit Go tests out of the real repo in worktrees`
1. `test(commit): move stray commit-skill tests into test/commit`
2. `chore: ignore the in-repo pnpm store`
3. `docs: describe where tests live in the repo layout`

End each message with your operator's attribution trailer. Do not push unless
told to.

## Steps

### Step 0: Stop the pre-commit hook's Go tests from writing into the real repo

Added 2026-11-05 after the first execution attempt. In a linked git worktree,
git exports absolute `GIT_DIR` and `GIT_INDEX_FILE` to hooks (in the main
checkout `GIT_INDEX_FILE` is the relative `.git/index`, which is harmless).
`.husky/pre-commit` then runs `go test ./...`, and the tests' helpers run
`git -C <tempdir> init/add/commit` (`internal/refresh/refresh_test.go:57`,
`internal/cli/cli_test.go:415`, `internal/tui/app_test.go:346`). Those
calls inherit the variables and act on the real repository. The first
attempt got stray `initial` / `group by author` commits on the worktree
branch, an emptied index, and `core.bare = true` in the shared `.git/config`.

`.husky/pre-commit` today:

```sh
pnpm exec lint-staged
go build ./...
go vet ./...
go test ./...
```

Insert one line after `pnpm exec lint-staged` (lint-staged needs the variables;
the Go checks must not see them):

```sh
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE
```

Commit this change **first and on its own**. The hook runs from the working
tree, so this commit is already protected.

**Verify**: after the commit, `git log --oneline -2` shows only your commit on
top of the start commit (no `initial` / `group by author`), and
`git config --get core.bare` prints `false`. If either fails, STOP: do not
retry, reset or repair anything; report.

### Step 1: Port the three uncovered cases into `test/commit/test_plan.py`

Add a constant under `PLAN` (line 8), built the same way:

```python
RENDER = Path(__file__).resolve().parents[2] / "skills" / "jorden" / "commit" / "render"
```

Add three methods to `PlanTestCase`, after `test_gather_shows_untracked_contents`
and before the `if __name__` block. `setUp` leaves `a.txt` untracked, so the
clean-tree test must remove it first:

```python
    def test_gather_on_a_clean_tree_says_nothing_to_commit(self):
        (self.repo / "a.txt").unlink()
        result = self.plan("gather")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("nothing to commit", result.stdout)

    def test_build_rejects_the_no_open_flag(self):
        result = self.plan("build", "--no-open", document=self.one_commit())
        self.assertEqual(result.returncode, 2)
        self.assertIn("unrecognized arguments: --no-open", result.stderr)

    def test_render_rejects_the_no_open_flag(self):
        result = subprocess.run([str(RENDER), "--no-open", "plan.json"], cwd=self.repo, capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 2)
        self.assertIn("unrecognized arguments: --no-open", result.stderr)
```

Do not add comments.

**Verify**: `python3 -m unittest discover -s test -p 'test_plan.py' -v` → 10 tests, all
`ok`, including the three new names. Then `pnpm test` → `Ran 34 tests` and `OK`.

### Step 2: Delete the stray test directory

```sh
git rm -r skills/jorden/commit/tests
rm -rf skills/jorden/commit/tests
```

(The second command removes the untracked `__pycache__` that `git rm` leaves.)

**Verify**:

- `ls skills/jorden/commit` → `assets  plan  render  SKILL.md` (no `tests`).
- `git ls-files skills/jorden/commit | grep -c tests` → `0`.
- `grep -rn "commit/tests" --exclude-dir=node_modules --exclude-dir=.pnpm-store --exclude-dir=.git --exclude-dir=plans .` → no output.
- `pnpm test` → `Ran 34 tests`, `OK`.

### Step 3: Ignore the in-repo pnpm store

Append `.pnpm-store/` as a new last line of `.gitignore`, and `.pnpm-store` as
a new last line of `.prettierignore`.

**Verify**: `git status --short` no longer lists `?? .pnpm-store/`;
`git check-ignore .pnpm-store` prints `.pnpm-store`.

### Step 4: Write the layout convention into the README

Replace the code block under `## Repo layout` in `README.md` with:

```
skills/<author>/<name>/  the skills themselves, grouped by author; linked into agent apps, so no tests here
main.go, internal/       the skills command (Go + Charm); each package's tests sit beside it
install.sh               builds and installs the command
artifact/                the shared Markdown-to-page renderer and its test.sh
test/<skill>/            Python tests for skill scripts; pnpm test runs them
CONTEXT.md               glossary
docs/adr/                decisions worth remembering
plans/                   implementation plans and their status
```

Change nothing else in the README.

**Verify**: `pnpm exec prettier --check README.md .gitignore .prettierignore`
→ `All matched files use Prettier code style!`

### Step 5: Run every existing check

**Verify**, all must pass:

- `go build ./... && go vet ./... && go test ./...` → every package `ok` or `[no test files]`.
- `pnpm test` → `Ran 34 tests`, `OK`.
- `bash artifact/test.sh` → `# all 22 tests passed`.
- If `skills` is on PATH: `skills refresh --check commit` → exit 0 (an
  installed `commit` link now simply has no `tests/` inside). If it is not on
  PATH, run `go build -o bin/skills . && ./bin/skills --repo . refresh --check commit`.

## Test plan

Three new unittest cases in `test/commit/test_plan.py` (step 1) replace the
three uncovered pytest cases. The other three pytest cases are dropped because
`test/commit/test_plan.py` already covers them (table above). Net Python test
count goes from 31 to 34.

## Done criteria

- [ ] `skills/jorden/commit/tests` does not exist, on disk or in `git ls-files`
- [ ] `pnpm test` prints `Ran 34 tests` and `OK`
- [ ] `go test ./...` and `bash artifact/test.sh` pass as before
- [ ] `git status --short` does not list `.pnpm-store/`
- [ ] `README.md` "Repo layout" names `test/<skill>/` and says skills hold no tests
- [ ] `git diff --stat aeb7cfc..HEAD` touches only the in-scope files
- [ ] Your row in `plans/README.md` says DONE with the commit SHA

## STOP conditions

- `pnpm test` does not print `Ran 31 tests` / `OK` before you change anything.
- `skills/jorden/commit/plan` no longer prints `nothing to commit`, or either
  script accepts `--no-open` (exit 0) — the behaviour this plan pins has
  changed; report it rather than adjusting the assertion.
- Anything other than the stray tests references `skills/jorden/commit/tests`.
- `skills refresh --check commit` reports an entry for review or a failure.

## Maintenance notes

- Convention from now on: tests for a skill's scripts go in
  `test/<skill_name>/` (Python package with `__init__.py`, `test_*.py`, using
  unittest, since pytest is not a dependency). Nothing test-shaped goes under
  `skills/`, because those directories are linked into agent apps as-is.
  `artifact/test.sh` is the one co-located exception: `artifact/` is a repo
  tool, not a linked skill.
- Plan 006 suggests a `skills/pr-draft/test.sh`. Under this convention it
  belongs at `test/pr_draft/` instead (a `test_preview.py` already exists
  there).
- Plan 014 predates the current `package.json` `test` script. Whoever runs 014
  must keep `python3 -m unittest discover -s test -p 'test_*.py'` in the
  combined `test` script.
