# Plan 016: Rename the `skills/` source folder to `catalog/`, and let `skills sync` move existing links across

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat aeb7cfc..HEAD -- internal main.go install.sh artifact README.md AGENTS.md CONTEXT.md test`
> Plan 015 is expected to have changed `test/commit/test_plan.py` and
> `README.md`'s layout block; that is fine. Any other change in the listed
> paths: compare the "Current state" excerpts with the live code; on a
> mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED (it rewrites the user's real skill links when `skills sync` runs after merge)
- **Depends on**: `plans/015-put-every-skill-script-test-under-test.md` (run it first, so this plan's path sweep also covers the test it adds)
- **Category**: tech-debt
- **Planned at**: commit `aeb7cfc`, 2026-11-05

## Why this matters

The project is called `skills` (the repo, the Go module
`github.com/jorden-parker/skills`, the `skills` command and the checkout
`~/src/skills`), and it has a child folder also called `skills/` that holds
the skills: `~/src/skills/skills/<author>/<name>/`. The maintainer finds this
confusing and chose to rename the **child folder** to `catalog/`. The repo,
module, command and checkout path keep their names.

A plain folder rename would break things:

1. Every installed link (`~/.agents/skills/<name>` and
   `~/.claude/skills/<name>`) points at `<checkout>/skills/<author>/<name>`.
   After the rename those links dangle. `skills sync` relinks enabled skills,
   but a link that exists only in `~/.claude/skills` would be left dangling
   for ever, because `Prune` only removes dead links under the source folder.
2. `links.Migrate` and `links.FlatLinkSkill` repair links from before skills
   moved into author folders (`<checkout>/skills/<name>`). They find those
   links by looking under `r.SrcDir`. Once `SrcDir` is `catalog/`, that
   repair silently stops working.
3. `artifact/render` reads the Geist CSS from a hard-coded
   `skills/jorden/geist` path at runtime.

This plan renames the folder, teaches `sync` and `refresh` about the old
folder so existing installs repair themselves, and updates every path
reference. The commands behave the same as before.

## Current state

### Production code that names the source folder (only these)

`internal/skills/repo.go:18-46`:

```go
type Repo struct {
	Dir    string // the checkout
	SrcDir string // Dir/skills, the source of every skill
}
...
	r := Repo{Dir: dir, SrcDir: filepath.Join(dir, "skills")}
	if fi, err := os.Stat(r.SrcDir); err != nil || !fi.IsDir() {
		return r, fmt.Errorf("no skills/ directory in %s (set SKILLS_REPO or pass --repo)", dir)
	}
```

`internal/skills/repo.go:48`: comment `// Names lists the ID (author/name) of every skills/<author>/<name>/ that`.

`internal/refresh/refresh.go:202`:

```go
	v := KnownVersions(in.repo.Dir, filepath.Join("skills", filepath.FromSlash(id), "SKILL.md"))
```

`artifact/render:150`:

```python
        geist = root.parent / "skills/jorden/geist"
```

`install.sh:4`: comment `# finds its skills/ directory from anywhere.`

### Code that looks under the source folder for old links

`internal/links/links.go:92-153` (abridged):

```go
// Migrate relinks dangling links to a flat skills/<name> path, left from
// before skills moved into author folders, to the one skill with that name.
func Migrate(r skills.Repo, dirs []string) ([]string, error) {
	ids, err := r.Names()
	...
	for _, target := range danglingLinksInto(r.SrcDir, dirs) {
		id, ok := FlatLinkSkill(r, ids, target)
		...
		if err := os.Symlink(r.Source(id), target); err != nil {
	...
}

// FlatLinkSkill reports the skill a link to a flat skills/<name> path now
// belongs to, when exactly one of ids has that name.
func FlatLinkSkill(r skills.Repo, ids []string, link string) (string, bool) {
	dest, err := os.Readlink(link)
	...
	flatName, ok := strings.CutPrefix(dest, r.SrcDir+string(filepath.Separator))
	...
	for _, id := range ids {
		if skills.LinkName(id) == flatName {
			matches = append(matches, id)
	...
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

// Prune removes dangling symlinks in dirs that point under srcDir and
// returns the paths it removed. Links elsewhere are left alone.
func Prune(srcDir string, dirs []string) ([]string, error) {
	for _, target := range danglingLinksInto(srcDir, dirs) {
```

Callers:

- `internal/cli/sync.go:28` `links.Migrate(r, paths.LinkDirs())`
- `internal/cli/sync.go:51` `links.Prune(r.SrcDir, paths.LinkDirs())`
- `internal/refresh/refresh.go:179` `links.FlatLinkSkill(in.repo, in.ids, target)`
- `internal/links/links_test.go:137` `Prune(r.SrcDir, p.LinkDirs())`

How `sync` works today (`internal/cli/sync.go:22-56`): `Migrate`, then
`LoadStates`, then `links.Link(...)` for every skill that isn't Off (`Link`
replaces **any** existing symlink with one to the current source), then
`Prune`, then write overrides. `LoadStates` counts a skill as linked when a
link with its name exists in `~/.agents/skills`, even a dangling one
(`internal/skills/state.go:105-116`, uses `lexists`). So enabled skills keep
their state across the rename and `sync` re-points them; the gaps are items
1 and 2 above.

### `"skills"` literals that must NOT change

They name other things:

- `~/.agents/skills`, `$CLAUDE_CONFIG_DIR/skills`, `$CODEX_HOME/skills`:
  agent link folders (`internal/skills/state.go:64,66`,
  `internal/refresh/refresh.go:49`, and the `h.links`, `h.claudeLinks`,
  `codex/skills` paths in tests).
- `~/src/skills`: the default checkout (`internal/skills/repo.go:36`,
  `skills_test.go:106`).
- `cli.Main("skills", ...)` in `main.go:9`, `Root("skills")` in tests, and
  `SetStatusBarItemName("skill", "skills")` in `internal/tui/app.go:95`.
- The Go module path `github.com/jorden-parker/skills` and every import.
- `upstream-skills.json`: its values (`"skills/engineering/ask-matt"`, …) are
  paths in the **upstream** repos. Do not edit this file.
- `internal/cli/refresh.go:24` help text "A link to the old flat
  skills/<name> path…": that describes the legacy path, which really is
  under `skills/`. Keep it.
- `docs/adr/0001`–`0005`: historical records. Do not edit them.
- Everything inside `catalog/` after the move, especially vendored skills in
  `catalog/mattpocock/` and `catalog/shadcn/`.

### Runtime paths that need no change

These are relative and keep working after the move:

- `catalog/jorden/web-research/artifact -> ../../../artifact/render` (same depth).
- `skills/jorden/pr-draft/preview` uses `skill.parent / "geist"` and
  `skill.parents[2] / "node_modules"`.
- `skills/jorden/commit/render` uses `SKILL_DIRECTORY.parent / "geist"`.

### Proven

`git log --all --follow --name-only -- catalog/me/example/SKILL.md` lists
`catalog/me/example/SKILL.md`, `skills/me/example/SKILL.md` and
`skills/example/SKILL.md` in a scratch repo after the moves
flat → author → `catalog/`. So `refresh` still recognises old installed
copies once line 202 uses the new path.

### Repo rules

- TDD: write or change the failing test first, then the code. The
  maintainer's hooks enforce this (a code file can only be edited after a test
  in its package was edited and a test command ran).
- No new code comments. Updating an existing doc comment's text is fine.
- Self-describing names. `gofmt` on save (lint-staged).
- Exemplar for a link test: `TestMigrateRelinksFlatLinksToTheirAuthorFolder`,
  `internal/links/links_test.go:170-196`, using `setup(t)` at `:12-31`.

## Commands you will need

| Purpose        | Command                                                                                                            | Expected on success                     |
| -------------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------- |
| Go tests       | `go build ./... && go vet ./... && go test ./...`                                                                  | every package `ok` or `[no test files]` |
| One package    | `go test ./internal/links/`                                                                                        | `ok`                                    |
| Python tests   | `pnpm test`                                                                                                        | `OK` (34 tests after plan 015)          |
| Renderer tests | `bash artifact/test.sh`                                                                                            | `# all 22 tests passed`                 |
| Format         | `test -z "$(gofmt -l .)" && pnpm exec prettier --check README.md AGENTS.md CONTEXT.md artifact/README.md docs/adr` | exit 0                                  |
| Build binary   | `go build -o bin/skills .`                                                                                         | exit 0                                  |

## Scope

**In scope**:

- `skills/` → `catalog/` (the move itself)
- `internal/skills/repo.go`, `internal/links/links.go`, `internal/cli/sync.go`, `internal/refresh/refresh.go`
- Tests: `internal/skills/skills_test.go`, `internal/links/links_test.go`, `internal/refresh/refresh_test.go`, `internal/cli/cli_test.go`, `internal/tui/app_test.go`, `test/commit/test_plan.py`, `test/commit/test_render.py`, `test/pr_draft/test_preview.py`
- `artifact/render` (line 150 only), `install.sh` (line 4 comment only)
- Docs: `README.md`, `AGENTS.md`, `CONTEXT.md`, `artifact/README.md`, `docs/adr/0006-rename-skills-folder-to-catalog.md` (create)
- `plans/README.md` (your status row only)

**Out of scope**: everything in "`"skills"` literals that must NOT change"
above; renaming the repo, module, command or checkout; any change to the
**contents** of files inside `catalog/`; plans 010–014.

## Git workflow

The maintainer commits straight to `main` in this repo, so commit there unless
your operator says otherwise. Use Conventional Commits; commitlint enforces
them (see `git log --oneline`, e.g. `feat: group skills by author and allow
same-name skills`). Suggested commits:

1. `refactor!: rename the skills/ source folder to catalog/`: the whole
   change, including the move. Use one commit so the history is never broken
   at any point. Footer:
   `BREAKING CHANGE: rebuild with ./install.sh, then run skills sync to move existing links.`

End the message with your operator's attribution trailer. Do not push unless
told to.

**Never run `skills sync`, `skills add/remove/set`, or `skills refresh`
without `--check` against your real `$HOME` from a worktree.** The Go tests
use temporary homes; the real-home migration is a post-merge operator step
(Step 7).

## Steps

### Step 1: Baseline

**Verify**: all four commands in the table above (Go tests, `pnpm test`,
`bash artifact/test.sh`, format) pass before any change. If not, STOP.

### Step 2: Tests first: point fixtures at `catalog/` and describe the new behaviour

Make these test edits now. Most will fail to compile or fail until Step 3.

**Rule for fixtures**: a path that builds the **current** source of a skill
changes `"skills"` → `"catalog"`. A path that builds an **old** link target
(a flat `skills/<name>` link, or anything from before this rename) uses the
old folder: `r.OldSrcDir` where a `Repo` is in hand, or the literal
`"skills"` under the checkout. Exact lines at `aeb7cfc`:

- `internal/skills/skills_test.go`: `:17` and `:30` → `"catalog"`; `:102`
  → `filepath.Join(home, "src", "skills", "catalog")` (the first `skills`
  is the checkout name and stays). Leave `:106`, `:136`, `:143`.
- `internal/tui/app_test.go:48` → `"catalog"`.
- `internal/links/links_test.go`: `:16` → `"catalog"`; `:177` and `:193`
  (flat links in the Migrate test) `r.SrcDir` → `r.OldSrcDir`; `:137`
  `Prune(r.SrcDir, p.LinkDirs())` → `Prune(r, p.LinkDirs())`.
- `internal/refresh/refresh_test.go`: in `newRepo`, keep lines 21–41 as
  they are (they build the history under `skills/`). After the
  `"group by author"` commit (line 41) and **before** the
  `current instructions` write, add a third move and commit:
  ```go
  gitIn(t, dir, "mv", "skills", "catalog")
  gitIn(t, dir, append(commit, "rename to catalog")...)
  src = filepath.Join(dir, "catalog", "me", "example")
  ```
  Line `:84` → `filepath.Join("catalog", "me", "example", "SKILL.md")` and
  rename that test to `TestKnownVersionsFollowEveryMove`. Line `:148`
  `filepath.Join(r.SrcDir, "example")` (a flat legacy link) →
  `filepath.Join(r.OldSrcDir, "example")`. Leave `:124`.
- `internal/cli/cli_test.go`: lines `43, 149, 167, 227, 245, 281, 289, 325,
385, 388, 539, 563` → `"catalog"`. **Keep** `"skills"` at `317` and `559`.
  Those are legacy flat links (`h.repo/skills/gone`, `h.repo/skills/alpha`)
  and now test the old-folder repair. Leave `457, 481, 502` (Codex folder).
- `test/commit/test_plan.py`, `test/commit/test_render.py`: `"skills"` →
  `"catalog"` in the `PLAN` / `RENDER` constants.
  `test/pr_draft/test_preview.py:6`: `"skills/jorden/pr-draft/preview"` →
  `"catalog/jorden/pr-draft/preview"`.

Add new tests in `internal/links/links_test.go`, modelled on
`TestMigrateRelinksFlatLinksToTheirAuthorFolder`:

```go
func TestMigrateRelinksAuthorLinksIntoTheOldSkillsFolder(t *testing.T) {
	r, p := setup(t)
	for _, dir := range p.LinkDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(r.OldSrcDir, "me", "alpha"), filepath.Join(dir, "alpha")); err != nil {
			t.Fatal(err)
		}
	}
	relinked, err := Migrate(r, p.LinkDirs())
	if err != nil {
		t.Fatal(err)
	}
	if len(relinked) != 2 {
		t.Fatalf("relinked = %v", relinked)
	}
	for _, dir := range p.LinkDirs() {
		if dest, _ := os.Readlink(filepath.Join(dir, "alpha")); dest != r.Source("me/alpha") {
			t.Fatalf("%s/alpha -> %q", dir, dest)
		}
	}
}

func TestPruneRemovesDanglingLinksIntoTheOldSkillsFolder(t *testing.T) {
	r, p := setup(t)
	if err := os.MkdirAll(p.ClaudeLinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(p.ClaudeLinkDir, "gone")
	if err := os.Symlink(filepath.Join(r.OldSrcDir, "me", "gone"), stale); err != nil {
		t.Fatal(err)
	}
	removed, err := Prune(r, p.LinkDirs())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{stale}) || isLink(stale) {
		t.Fatalf("removed = %v", removed)
	}
}
```

Add one test in `internal/skills/skills_test.go` next to the existing `Find`
tests:

```go
func TestFindNamesTheCatalogAndTheOldSkillsFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Find(dir)
	if err != nil || r.SrcDir != filepath.Join(dir, "catalog") || r.OldSrcDir != filepath.Join(dir, "skills") {
		t.Fatalf("Find = %+v, %v", r, err)
	}
}
```

(If an existing `Find` test already asserts the error text
`no skills/ directory`, change it to `no catalog/ directory`.)

**Verify**: `go test ./...` fails. The failures are compile errors about
`OldSrcDir` and `Prune`'s signature, or missing-`catalog` failures. Nothing
else may fail; if an unrelated test fails, STOP.

### Step 3: Move the folder and change the code

1. Move: `mv skills catalog && git add -A skills catalog`. `mv` also carries
   untracked files such as `.DS_Store`. Afterwards `test ! -e skills` must
   succeed.
2. `internal/skills/repo.go`: add the field and set it in `Find`; change the
   error text and the doc comments:
   ```go
   type Repo struct {
   	Dir       string // the checkout
   	SrcDir    string // Dir/catalog, the source of every skill
   	OldSrcDir string // Dir/skills, where skills lived before catalog/; old links may still point here
   }
   ...
   	r := Repo{Dir: dir, SrcDir: filepath.Join(dir, "catalog"), OldSrcDir: filepath.Join(dir, "skills")}
   	if fi, err := os.Stat(r.SrcDir); err != nil || !fi.IsDir() {
   		return r, fmt.Errorf("no catalog/ directory in %s (set SKILLS_REPO or pass --repo)", dir)
   	}
   ```
   In the `Names` doc comment, `skills/<author>/<name>/` → `catalog/<author>/<name>/`.
3. `internal/links/links.go`:
   - Rename `FlatLinkSkill` → `OldLinkSkill`. It reads the link, cuts the
     prefix `r.OldSrcDir + string(filepath.Separator)` (was `r.SrcDir`), and:
     if `r.Has(filepath.ToSlash(rest))` it returns that ID (an
     `author/name` link); otherwise it keeps today's unique-flat-name match.
     Doc comment: "OldLinkSkill reports the skill a link into the old
     skills/ folder now belongs to: the same author/name, or the one skill
     with a flat link's name."
   - `Migrate`: scan `danglingLinksInto(r.OldSrcDir, dirs)` and call
     `OldLinkSkill`. Doc comment: "Migrate relinks dangling links into the
     old skills/ folder, flat or by author, to the skill's place in the
     catalog. It returns the paths it relinked; links to an ambiguous flat
     name are left alone."
   - `Prune(r skills.Repo, dirs []string)`: remove dangling links under
     `r.SrcDir` and then under `r.OldSrcDir`, in that order, and return all
     removed paths. Keep its error handling: return what was removed so far
     on the first error.
4. `internal/cli/sync.go:51`: `links.Prune(r, paths.LinkDirs())`.
5. `internal/refresh/refresh.go:179`: `links.OldLinkSkill(...)`; `:202`:
   `filepath.Join("catalog", ...)`. Update the `recognise` doc comment's
   "a dangling flat link" → "a dangling link into the old skills/ folder".

**Verify**: `go build ./... && go vet ./... && go test ./...` passes, and
`grep -rn FlatLinkSkill internal` prints nothing.

### Step 4: Non-Go runtime path

`artifact/render:150`: `root.parent / "skills/jorden/geist"` →
`root.parent / "catalog/jorden/geist"`.

**Verify**: `bash artifact/test.sh` → `# all 22 tests passed`; `pnpm test`
→ `OK`.

### Step 5: Docs

- `README.md`, `AGENTS.md`, `CONTEXT.md`, `artifact/README.md`: change every
  reference to the **source folder** (`skills/<author>/<name>`,
  `skills/jorden/…`, `skills/mattpocock/…`, `skills/shadcn/…`,
  `./skills/jorden/pr-draft/preview`, "A path without a `skills/`
  directory") to `catalog/`. Keep everything in the must-not-change list
  (agent folders, `~/src/skills`, the command, `upstream-skills.json`).
- `install.sh:4` comment: `skills/ directory` → `catalog/ directory`.
- In `README.md` under the `skills sync` description (in the
  `## Commands` section), add one sentence: "After pulling a change that
  moves skills, such as the move from `skills/` to `catalog/`, rebuild with
  `./install.sh` and run `skills sync`; it repoints old links."
- Create `docs/adr/0006-rename-skills-folder-to-catalog.md`, matching the
  style of `docs/adr/0005-group-skills-by-author.md` (read it first; copy
  its heading and section structure). Content: the repo, module, command
  and checkout are all called `skills`, so `~/src/skills/skills/` was
  confusing; the source folder is now `catalog/`; `Repo.OldSrcDir` lets
  `sync`/`refresh` repair links into the old folder; the outer names stay
  because renaming the module and checkout would touch every import and
  every machine's install for no gain.

**Verify**:

- `grep -nE '(^|[ (\x60])\.?/?skills/(<author>|<name>|jorden|mattpocock|shadcn)' README.md AGENTS.md CONTEXT.md artifact/README.md install.sh` → no output.
- `grep -rn "skills/jorden\|skills/mattpocock\|skills/shadcn" --include='*' internal test artifact` → no output.
- Format command from the table → exit 0.

### Step 6: Full check

**Verify**, all pass:

- `go build ./... && go vet ./... && go test ./...`
- `pnpm test`, `bash artifact/test.sh`
- `go build -o bin/skills . && ./bin/skills --repo . list | head -5`: lists
  skills and prints `Skills in <checkout>/catalog`. This is read-only, but it
  reads your real home's link state; that's fine.
- `./bin/skills --repo . refresh --check`: read-only. Exit 1 is expected from
  a worktree, because installed links point at `~/src/skills`. Only report
  the output; do not act on it.

### Step 7: Post-merge, operator only (do not run from a worktree)

After the commit is on `main` in `~/src/skills`:

```sh
cd ~/src/skills
./install.sh                 # rebuild: the old binary looks for skills/
skills sync                  # repoints links from skills/... to catalog/...
skills refresh --check       # expect exit 0
```

**Verify**: running
`find ~/.agents/skills ~/.claude/skills -maxdepth 1 -type l -lname "$HOME/src/skills/skills/*"`
prints nothing, and `skills list` shows the same On/partial states as
before. Record that state first with `skills list > /tmp/claude-skills/before.txt`,
run **before** `./install.sh`.

## Test plan

- Changed fixtures (Step 2) keep every existing behaviour under `catalog/`.
- Legacy flat-link tests (`links_test` Migrate, `refresh_test:148`,
  `cli_test:317,559`) now exercise links into the old `skills/` folder, which
  is where such links really point.
- New: `TestMigrateRelinksAuthorLinksIntoTheOldSkillsFolder`,
  `TestPruneRemovesDanglingLinksIntoTheOldSkillsFolder`,
  `TestFindNamesTheCatalogAndTheOldSkillsFolder`, and the renamed
  `TestKnownVersionsFollowEveryMove` (three moves of history).

## Done criteria

- [ ] `test ! -e skills && test -d catalog/jorden/commit`
- [ ] `go test ./...`, `pnpm test`, `bash artifact/test.sh` pass
- [ ] `grep -rn '"skills"' internal --include='*.go'` shows only the must-not-change literals (agent folders, `~/src/skills`, command name, status bar, `OldSrcDir`'s `"skills"`) plus `cli_test.go:317,559`
- [ ] `git diff --stat aeb7cfc..HEAD -- upstream-skills.json docs/adr/000[1-5]*` is empty
- [ ] No file inside `catalog/` changed content (`git diff -M --stat aeb7cfc..HEAD -- catalog` shows only renames, `=>` lines with 0 changes)
- [ ] Status row in `plans/README.md` updated

## STOP conditions

- The baseline in Step 1 fails.
- After the move, `git status` shows content changes inside `catalog/`
  rather than pure renames.
- Any file other than those listed references the source folder by path
  (for example a skill script outside `jorden/commit`, `jorden/pr-draft` or
  `artifact/render` that builds `skills/...`). Report it; don't widen scope
  yourself.
- Plan 010 has already landed (check whether `links.Link` still replaces
  any existing symlink). If it now leaves foreign links alone, Step 7's
  `sync` may not repoint old links. STOP and report so the advisor can
  reconcile the two plans.
- You are tempted to run `skills sync` or a non-`--check` refresh to "test"
  the migration on your real home.

## Maintenance notes

- `OldSrcDir` is a migration aid. Remove it, together with
  `OldLinkSkill`'s old-folder branch and the second pass in `Prune`, only
  once no machine can still have links into `<checkout>/skills/`. That's a
  maintainer call; keep it until then. It costs one `ReadDir` per link dir.
- Plan 010 (leave links owned elsewhere alone) must count links into
  `r.OldSrcDir` as owned by this repo, or `sync` will stop repairing them.
- Plans 006, 007, 013 and 014 cite `skills/...` paths. Their drift checks
  will flag it; read `skills/X` as `catalog/X` there.
