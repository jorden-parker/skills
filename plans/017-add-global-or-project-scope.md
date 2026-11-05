# Plan 017: Let every command and the garden work on either the global install or the current project

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a03e1..HEAD -- internal README.md CONTEXT.md`
> Plans 010, 011 and 012 also edit `internal/links`, `internal/settings` and
> `internal/tui`. If any in-scope file changed since `12a03e1`, compare the
> "Current state" excerpts with the live code; on a mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (new write locations inside the user's project repos)
- **Depends on**: none. Plan 018 depends on this one.
- **Category**: direction
- **Planned at**: commit `12a03e1`, 2026-11-05

## Why this matters

Today every link goes into the user's global discovery directories:
`~/.agents/skills` and `${CLAUDE_CONFIG_DIR:-~/.claude}/skills`, with partial
states in `~/.claude/settings.json`. The maintainer wants to pick a **scope**:
`global` (today's behaviour, still the default) or `project`. Project scope
puts the links in the current Git project instead, so a skill can be on in
one project without being on everywhere. The scope applies to the CLI
(`--scope`) and to the garden TUI (a key that switches between the two).

Decisions already made by the maintainer (do not revisit them):

- The scopes are `global` and `project`. There is no "both" view.
- Project means the Git top level of the current working directory.
- Project links go in `<root>/.agents/skills` and `<root>/.claude/skills`.
- Project partial states go in `<root>/.claude/settings.local.json`. That file
  is personal; Claude Code does not commit it by default.

## Current state

- `internal/skills/state.go`: `Paths` holds every location. `DefaultPaths()`
  builds the global ones. Lines 39–73:

  ```go
  type Paths struct {
  	Home          string
  	LinkDir       string // ~/.agents/skills
  	ClaudeDir     string // CLAUDE_CONFIG_DIR or ~/.claude
  	ClaudeLinkDir string // ClaudeDir/skills
  	Settings      string // ClaudeDir/settings.json
  	CodexDir      string // CODEX_HOME or ~/.codex
  }

  func DefaultPaths() Paths { ... }

  // LinkDirs are the two discovery directories a skill is linked into.
  func (p Paths) LinkDirs() []string { return []string{p.LinkDir, p.ClaudeLinkDir} }
  ```

  `LoadStates(r, p)` reads link state from `p.LinkDir` and overrides from
  `p.Settings`. `links.Apply`, `links.Migrate`, `links.Prune` and
  `settings.WriteOverrides` all take their directories and files from a
  `Paths`. **So a project-scoped `Paths` is all the link and settings code
  needs; it does not change.**

- Every caller builds paths with `skills.DefaultPaths()`:
  - `internal/cli/list.go:20`, `internal/cli/sync.go:24`,
    `internal/cli/set.go:69` (`applyStates`, used by `add`, `remove`, `set`),
    `internal/cli/refresh.go:36`, `internal/cli/root.go:86` (`tui.Run`).
- `internal/cli/root.go`: `Root(program)` registers one persistent flag,
  `--repo`/`-C`, and passes a `finder` closure (`func() (skills.Repo, error)`)
  to each subcommand:

  ```go
  root.PersistentFlags().StringVarP(&repoFlag, "repo", "C", "", "path to the skills checkout (...)")
  find := func() (skills.Repo, error) { return skills.Find(repoFlag) }
  root.AddCommand(listCmd(find), addCmd(find), removeCmd(find), setCmd(find), syncCmd(find), refreshCmd(find))
  ```

- `refresh` (`internal/refresh/refresh.go:49`) scans `p.LinkDir`,
  `$CODEX_HOME/skills` and `p.ClaudeLinkDir` for installed **copies** across
  agent apps. That is a global maintenance job. It stays global-only.
- `internal/tui/app.go`: the `model` keeps `paths skills.Paths`.
  `keys` (lines 36–48) is the keymap. `updateGarden` (line 241) handles the
  keys. `header()` (line 418) prints the catalog dir on line 2. The help screen
  rows are at lines 462–466 (footer hints) and 503–505 (the `?` screen).
  `reload()` re-reads states from `m.paths`.
- Running Git: `internal/refresh` already shells out to `git`, so needing Git
  for project scope is fine.

### Conventions

- Errors are plain `fmt.Errorf` strings that name the bad value, for example
  `fmt.Errorf("bad state: %s (valid: on off name-only user-invocable-only)", s)`
  in `ParseState` (`state.go:26`). Copy that style for `ParseScope`.
- CLI tests use `newHarness(t)` and `h.ok(args...)` / `h.run(args...)` in
  `internal/cli/cli_test.go`; see `TestAddCreatesLinkAndRemoveDeletesIt`
  (line 160). TUI tests use `newFixture(t)`, `f.open()`, `press(m, keys...)` in
  `internal/tui/app_test.go`; see `TestStageThenSavePreservesPartialStates`
  (line 175).
- The maintainer's hooks reject new code comments and require TDD: edit a
  test in the package and run it before you edit the code file. Use clear
  names instead of comments.
- Vocabulary (`CONTEXT.md`): say **Link**, **Skill**, **State**; never
  "install" for a link. Add a **Scope** term (Step 5).

## Commands you will need

| Purpose | Command                                                 | Expected on success |
| ------- | ------------------------------------------------------- | ------------------- |
| Build   | `go build ./...`                                        | exit 0              |
| Vet     | `go vet ./...`                                          | exit 0              |
| Tests   | `go test ./...`                                         | every package `ok`  |
| One pkg | `go test ./internal/cli -run X`                         | `ok`                |
| Format  | `gofmt -l .`                                            | no output           |
| JS/MD   | `pnpm exec prettier --check README.md CONTEXT.md plans` | exit 0              |

## Scope

**In scope:**

- `internal/skills/state.go`, `internal/skills/skills_test.go`
- `internal/cli/root.go`, `list.go`, `set.go`, `sync.go`, `refresh.go`,
  `cli_test.go`
- `internal/tui/app.go`, `internal/tui/app_test.go`
- `README.md`, `CONTEXT.md`

**Out of scope:**

- `internal/links/*`, `internal/settings/*`: they already work from a
  `Paths`. Plans 010–012 change them.
- `internal/refresh/*`: refresh stays global.
- Writing to `.gitignore` or `.git/info/exclude` in the user's project. Do not
  touch project ignore files (see Maintenance notes).
- Anything about skills.sh (that is plan 018).

## Git workflow

- Commit directly on `main`. The maintainer does not use branches in this
  repo.
- Conventional Commits, as in `git log`: `feat(scope): …`, `test: …`,
  `docs: …`. Example: `feat: add --scope global|project to every command`.
- The pre-commit hook runs `go build`, `go vet`, `go test` and lint-staged.
- Do not push.

## Steps

### Step 1: Add `Scope`, `ParseScope` and `PathsFor` to `internal/skills`

Test first, in `internal/skills/skills_test.go`:

- `ParseScope("global")` and `ParseScope("project")` succeed;
  `ParseScope("both")` returns an error containing `valid: global project`.
- `DefaultPaths().Scope == GlobalScope`.
- `PathsFor(ProjectScope, dir)`, where `dir` is a subfolder of a temp Git repo
  (`git init` in `t.TempDir()`, then `os.MkdirAll(dir)`), returns:
  - `Scope == ProjectScope`
  - `LinkDir == <root>/.agents/skills`
  - `ClaudeLinkDir == <root>/.claude/skills`
  - `Settings == <root>/.claude/settings.local.json`
  - `Home`, `ClaudeDir` and `CodexDir` equal to `DefaultPaths()`'s.

  Compare against `filepath.EvalSymlinks(root)`: on macOS `t.TempDir()`
  sits under `/var`, which is a symlink to `/private/var`, and Git prints
  the resolved path.

- `PathsFor(ProjectScope, t.TempDir())` outside any repo returns an error that
  contains `not inside a Git repository`.
- `PathsFor(GlobalScope, anything)` equals `DefaultPaths()`.

Then implement in `internal/skills/state.go`:

```go
type Scope string

const (
	GlobalScope  Scope = "global"
	ProjectScope Scope = "project"
)
```

- Add `Scope Scope` as the first field of `Paths`. Set it in `DefaultPaths()`.
- `ParseScope(s string) (Scope, error)`; error text
  `bad scope: <s> (valid: global project)`.
- `PathsFor(scope Scope, dir string) (Paths, error)`: global returns
  `DefaultPaths()`. Project runs `git -C <dir> rev-parse --show-toplevel`
  (`os/exec`, trim the output). On failure, return
  `fmt.Errorf("project scope needs a Git repository: %s is not inside a Git repository", dir)`.
  Otherwise start from `DefaultPaths()` and override `Scope`, `LinkDir`,
  `ClaudeLinkDir` and `Settings`.

Verify: `go test ./internal/skills` → `ok`.

### Step 2: Add a `--scope` flag to the CLI

Test first, in `internal/cli/cli_test.go`. Add a helper to the harness that
makes a Git repo under `h.base` (`git init <base>/project`), and run commands
from inside it with `t.Chdir(projectDir)` (Go 1.24+; this repo is on 1.27):

- `add alpha --scope project` creates links at
  `<project>/.agents/skills/alpha` and `<project>/.claude/skills/alpha`,
  resolving to the catalog. It creates **no** link in `h.links` or
  `h.claudeLinks`.
- `set alpha=name-only --scope project` writes
  `{"skillOverrides":{"alpha":"name-only"}}` into
  `<project>/.claude/settings.local.json`, and leaves `h.settings` absent.
- `list --scope project` shows `alpha` as `name-only`; `list` (global) shows
  it as `off`.
- `--scope nope list` exits non-zero with `bad scope: nope`.
- `refresh --scope project` exits non-zero with
  `refresh works on global installs only`.
- Run outside a repo (`t.Chdir(t.TempDir())`): `list --scope project` fails
  with `not inside a Git repository`.

Implement:

- In `root.go`, add a persistent `--scope` string flag (default `"global"`,
  help: `where links go: global (~/.agents/skills, Claude's skills dir) or project (the current Git project)`).
  Build one closure next to `find`:

  ```go
  pathsFor := func() (skills.Paths, error) {
  	scope, err := skills.ParseScope(scopeFlag)
  	if err != nil {
  		return skills.Paths{}, err
  	}
  	dir, err := os.Getwd()
  	if err != nil {
  		return skills.Paths{}, err
  	}
  	return skills.PathsFor(scope, dir)
  }
  ```

  Add a `type pathsFinder func() (skills.Paths, error)` beside `finder`.
  Pass it to `listCmd`, `addCmd`, `removeCmd`, `setCmd`, `syncCmd` and
  `refreshCmd`, and to the root `RunE`, which calls `tui.Run(r, paths)`.

- Replace every `skills.DefaultPaths()` call in `list.go`, `set.go`
  (`applyStates` gets the `pathsFinder` as a parameter) and `sync.go` with the
  closure. In `sync.go`, the heading line already prints `paths.LinkDir` and
  `paths.ClaudeLinkDir`, so the output shows the project dirs without further
  change.
- `refresh.go`: if the scope is `project`, return
  `errors.New("refresh works on global installs only; drop --scope project")`
  before doing anything. Otherwise keep using the global paths.
- Update the root command's `Long` help: add one line after the States block:
  `Scope: --scope global (default) or --scope project (links in <git root>/.agents/skills and .claude/skills, states in .claude/settings.local.json).`

Verify: `go test ./internal/cli` → `ok`. Then
`go build -o /tmp/skills-017 . && cd /tmp && /tmp/skills-017 list --scope project`
prints an error mentioning `not inside a Git repository`. Delete
`/tmp/skills-017` afterwards.

### Step 3: Switch scope in the garden with `p`

Test first, in `internal/tui/app_test.go`. The fixture's `HOME` is a temp
dir. Make a Git repo in `f.base` and `t.Chdir` into it:

- `press(m, "p")` with nothing pending sets `m.paths.Scope == skills.ProjectScope`,
  and `m.paths.LinkDir` is under the repo. Press `space` on `me/alpha`, then `s`
  and `enter`. The link exists under `<repo>/.agents/skills/alpha` and not
  under `f.paths.LinkDir`.
- Pressing `p` again returns to global, and `me/alpha` shows `off` there.
- With one change pending, `press(m, "p")` leaves the scope unchanged, and
  `m.status` contains `Save or discard`.
- In project scope, pressing `r` does not start a refresh. `m.mode` stays
  `modeGarden`, and `m.status` contains `global`.
- Outside a Git repo, `p` leaves the scope global and sets an error status
  that contains `not inside a Git repository`.

Implement in `internal/tui/app.go`:

- Add `scope key.Binding` to `keymap`:
  `key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "scope"))`.
- Add a `case key.Matches(k, keys.scope): return m, m.switchScope()` in
  `updateGarden`, placed before the `keys.refresh` case.
- `switchScope()`: if `m.pending() > 0`, call
  `setStatus("Save or discard staged changes before switching scope.", true)`.
  Otherwise pick the other scope, `os.Getwd()`, then `skills.PathsFor`. On an
  error, call `setStatus(err.Error(), true)`. On success, set `m.paths`, call
  `m.reload()`, set the status to `Scope: <scope> · <tildify(LinkDir)>`, and
  return `m.resetList()`.
- In the `keys.refresh` case, when `m.paths.Scope == skills.ProjectScope`,
  set the status `Refresh works on global installs only. Press p to switch.`
  and do not start a refresh.
- `header()` line 2: append `"  ·  " + string(m.paths.Scope) + " " + tildify(m.paths.LinkDir)`
  in the subtle style after the catalog dir.
- Help: add `hint{"p", "scope"}` to the footer hints (line 466 list), and
  `{"p", "switch scope: global or this Git project"}` to the `Save` group of
  the `?` screen (line 505).

Verify: `go test ./internal/tui` → `ok`.

### Step 4: Full gate

Run `go build ./... && go vet ./... && go test ./... && gofmt -l .`. Expect
exit 0, every package `ok`, and no gofmt output.

### Step 5: Document

- `README.md` `## Commands` block: add
  `skills add geist --scope project   # link into the current Git project only`.
  Under `### The skill garden`, add a key-table row:
  `| \`p\` | Switch scope: global, or the current Git project (only when nothing is staged) |`.
Add a short `### Scope`subsection after "Where the checkout lives". It
covers the two scopes, the exact project paths and settings file, that`refresh` is global-only, and that project links are absolute symlinks into
  this checkout and should not be committed.
- `CONTEXT.md` under `### Distribution`, after **State**:

  ```
  **Scope**:
  Where a machine's Links and partial States live: `global` (the user's discovery directories and settings.json) or `project` (the current Git project's `.agents/skills`, `.claude/skills` and `.claude/settings.local.json`).
  _Avoid_: level, target, location
  ```

Verify: `pnpm exec prettier --check README.md CONTEXT.md` → exit 0.

## Test plan

All new tests are listed in Steps 1–3. Pattern files:
`internal/skills/skills_test.go`, `TestAddCreatesLinkAndRemoveDeletesIt` in
`internal/cli/cli_test.go`, and `TestStageThenSavePreservesPartialStates` in
`internal/tui/app_test.go`. Every existing test must still pass unchanged.
Global stays the default, so if an existing test needs editing, that is a
regression.

## Done criteria

- `go test ./...` → every package `ok`; `go vet ./...` and `gofmt -l .` are clean.
- `grep -rn "DefaultPaths()" internal --include=*.go | grep -v _test.go`
  prints only `state.go` (the definition and `PathsFor`) and
  `internal/cli/refresh.go`.
- `go run . --help` mentions `--scope`.
- `README.md` and `CONTEXT.md` have the Scope text.
- The `plans/README.md` row for 017 says DONE.

## STOP conditions

- An existing test needs its expectations changed for the global scope.
- `links.Apply`, `links.Prune` or `settings.WriteOverrides` turn out to read a
  location that is not on `Paths` (a hard-coded `~/.agents` path, say).
  Report it; do not patch `internal/links` or `internal/settings` here.
- The drift check shows plans 010–012 landed and `LoadStates`/`Paths` look
  different from the excerpts.

## Maintenance notes

- Project links are absolute symlinks into this checkout, or into the plan 018
  download cache. They are machine-local. The tool does not edit the
  project's ignore files. If the maintainer wants that later, add the paths
  to `.git/info/exclude`, never `.gitignore`.
- The assumption to check by hand once: Claude Code honours `skillOverrides`
  from `.claude/settings.local.json` (settings merge across user, project and
  local scopes). If it does not, partial states in project scope only take
  effect after moving them to `.claude/settings.json`. Revisit then.
- Plan 018 adds registry skills. They follow whatever `Paths` the scope picks;
  the download cache itself is shared across scopes.
