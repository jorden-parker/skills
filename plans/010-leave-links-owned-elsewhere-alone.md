# Plan 010: Leave links owned by other tools alone, and only touch skills whose state changed

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- internal/links internal/cli internal/tui/review.go internal/refresh docs/adr README.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED (changes which existing symlinks the tool will replace)
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Why this matters

A second repository on the same machine, `~/src/claude-custom`, also links
skills into `~/.claude/skills`. Today `~/.claude/skills/handoff` and
`~/.claude/skills/pr-draft` point into `claude-custom/shared/skills/`, while
`~/.agents/skills/handoff` and `~/.agents/skills/pr-draft` point into this
repo. The `skills` command replaces **any** existing symlink, and every
`add`/`remove`/`set`/garden save re-applies **every** skill, not only the one
the user named. So `skills add geist` silently re-points claude-custom's
links into this repo. It also adds about 30 links to `~/.claude/skills`, which
`claude-ollama` (claude-custom's local-model launcher) then copies into its
sessions with `cp -RL`. claude-custom's own installer already refuses to
replace links it does not own (`bin/claude-custom:199-216`, `place_link`).
This plan makes this repo do the same. The two tools then stop overwriting
each other, and a one-skill command changes one skill.

`refresh` also reports those two foreign links as `review` on every run, so
`skills refresh --check` exits 1 forever. `AGENTS.md` tells agents to pass
that check before reporting completion, and that becomes impossible.

## Current state

Files:

- `internal/links/links.go` — `Link`, `Unlink`, `Apply`, `Prune`. Has the bug.
- `internal/links/links_test.go` — unit tests. `TestLinkCreatesBothAndRelinks` (line 38) asserts the current, unwanted behaviour ("An existing link, even to somewhere else, is replaced").
- `internal/cli/set.go` — `applyStates` (line 62) and `save` (line 105) call `links.Apply` with every entry.
- `internal/cli/sync.go` — calls `links.Link` for every non-off entry (line 36).
- `internal/tui/review.go` — garden `save()` (line 39) calls `links.Apply` with every staged entry.
- `internal/refresh/refresh.go` — `Run` (line 96) marks every unknown entry `Review`; `KnownVersions` (line 62).
- `internal/cli/refresh.go` — prints one line per result (lines 36-50).
- `internal/cli/cli_test.go` — CLI tests; harness `newHarness` (line 22), `symlink`, `resolves`, `isLink` helpers.
- `docs/adr/0003-toggle-skills-by-presence-overrides-for-partial-states.md` — outdated: says `~/.claude/skills` is the source of truth. The code reads `~/.agents/skills` (`internal/skills/state.go:93`).

`internal/links/links.go:17-33`:

```go
// Link points dir/name at src in every dir. An existing symlink is replaced;
// anything else is reported through warn and blocks the result.
func Link(src, name string, dirs []string, warn Warn) (blocked bool, err error) {
	for _, dir := range dirs {
		target := filepath.Join(dir, name)
		...
		fi, lerr := os.Lstat(target)
		switch {
		case lerr == nil && fi.Mode()&os.ModeSymlink != 0:
			if err := os.Remove(target); err != nil {
				return blocked, err
			}
			if err := os.Symlink(src, target); err != nil {
```

`internal/links/links.go:46-56` (`Unlink` deletes any symlink with the name):

```go
		case lerr == nil && fi.Mode()&os.ModeSymlink != 0:
			if err := os.Remove(target); err != nil {
```

`internal/links/links.go:64-86` (`Apply` walks every entry):

```go
func Apply(r skills.Repo, p skills.Paths, entries []skills.Entry, warn Warn) ([]skills.Entry, error) {
	out := make([]skills.Entry, len(entries))
	for i, e := range entries {
		out[i] = e
		if e.State == skills.Off {
			blocked, err := Unlink(e.Name, p.LinkDirs(), warn)
			...
		if _, err := Link(r.Source(e.Name), e.Name, p.LinkDirs(), warn); err != nil {
```

`internal/cli/set.go:104-111`:

```go
// save applies the staged states to the filesystem and settings.json.
func save(r skills.Repo, paths skills.Paths, staged []skills.Entry, warn links.Warn) ([]skills.Entry, error) {
	after, err := links.Apply(r, paths, staged, warn)
	if err != nil {
		return nil, err
	}
	return after, settings.WriteOverrides(paths.Settings, overrides(after))
}
```

`internal/refresh/refresh.go:111-129`:

```go
			if isLinkTo(target, source) {
				res.Status = Current
				...
			if versions == nil {
				versions = KnownVersions(r.Dir, filepath.Join("skills", name, "SKILL.md"))
			}
			manifest, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
			if _, known := versions[string(manifest)]; err != nil || !known || !isRegular(filepath.Join(target, "SKILL.md")) {
				res.Status = Review
				results = append(results, res)
				failed = true
				continue
			}
```

`sync` is documented as repairing "links that point at an old checkout"
(`internal/cli/sync.go:14-17`), and `TestSyncRepairsStaleLinksAndPrunesDanglingOnes`
(`internal/cli/cli_test.go:303`) creates an old checkout at
`<base>/old-checkout/skills/alpha` **with no SKILL.md** and expects it to be
re-pointed. The new ownership rule must still repair a real old checkout, so
that fixture gets a SKILL.md (step 4).

Vocabulary (`CONTEXT.md`): **Link** = the symlink from a discovery directory
into the Skills Directory; **Repo** = the checkout Links point into;
**Refresh** = replacing an installed copy with a Link after a backup. Use these
words in identifiers and docs.

Conventions: small functions, doc comment on every exported identifier
(see `internal/links/links.go`), table-free tests with `t.Fatal`/`t.Fatalf`
and temp dirs (see `internal/links/links_test.go:12-31`). **No code comments
beyond Go doc comments on exported identifiers**: the user's hooks reject
other comments. Go formatting is gofmt.

## The ownership rule (implement exactly this)

An existing **symlink** at `<dir>/<name>` belongs to this Repo when any of
the following holds:

1. It is dangling (its target does not exist).
2. Its fully resolved target (`filepath.EvalSymlinks`) is inside the fully
   resolved `r.SrcDir`.
3. It resolves to a directory whose `SKILL.md` bytes equal a known version of
   `skills/<name>/SKILL.md` (current file or any Git-history version, from
   `refresh.KnownVersions`). This covers an old checkout of this repo.

Anything else is **owned elsewhere**:

- `Link` reports `<target> - links to <resolved dest>, owned elsewhere, not touching it` through `warn` and blocks.
- `Unlink` reports the same message and blocks.
- `refresh` gives it the new status `Foreign`. That status does not count as failed.

Real directories and files keep their current behaviour.

## Commands you will need

| Purpose      | Command                                      | Expected on success    |
| ------------ | -------------------------------------------- | ---------------------- |
| Build        | `go build ./...`                             | exit 0                 |
| Vet          | `go vet ./...`                               | exit 0, no output      |
| Tests        | `go test ./...`                              | `ok` for every package |
| One package  | `go test ./internal/links/ -run TestName -v` | `PASS`                 |
| Format check | `gofmt -l .`                                 | no output              |

Never run the built `skills` binary against your real HOME. The tests isolate
HOME, `CLAUDE_CONFIG_DIR` and `CODEX_HOME` in temp dirs.

## Scope

**In scope**:

- `internal/links/links.go`, `internal/links/links_test.go`
- `internal/refresh/refresh.go`, `internal/refresh/refresh_test.go`
- `internal/cli/set.go`, `internal/cli/sync.go`, `internal/cli/refresh.go`, `internal/cli/cli_test.go`
- `internal/tui/review.go`, `internal/tui/refresh.go` (only if it switches on `refresh.Status` and needs the new case), `internal/tui/app_test.go`
- `docs/adr/0005-links-owned-elsewhere-and-the-agents-directory.md` (create)
- `docs/adr/0003-toggle-skills-by-presence-overrides-for-partial-states.md` (add one "Amended by 0005" line only)
- `README.md` (the paragraph about real directories, and the refresh paragraph)

**Out of scope**:

- `internal/settings/` — plan 011 hardens settings.json writes. Do not touch it here.
- Moving warnings from stdout to stderr. Tests assert the current stream; that is a separate change.
- The relative-symlink backup problem in `refresh.relink`, where a link is moved into `skill-backups` and breaks. It is deferred; see Maintenance notes.
- Anything in `~/src/claude-custom`.

## Git workflow

- Branch: `advisor/010-links-owned-elsewhere`
- Conventional Commits (commitlint enforces it), e.g. `fix: leave links owned by other tools alone` or `docs: record ownership rule for links`. End each message with the attribution trailer your operator gives you.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add an ownership predicate type to `links`

In `internal/links/links.go`, add:

```go
// Owns reports whether the existing symlink at target, installed under name,
// belongs to this Repo and may be replaced or removed.
type Owns func(name, target string) bool
```

Add `owns Owns` as a parameter to `Link` and `Unlink`, after `dirs`. In the
symlink case of each, call `owns(name, target)` first. When it returns false,
call `warn(target + " - links to " + dest + ", owned elsewhere, not touching it")`,
where `dest` is the `filepath.EvalSymlinks(target)` result (fall back to
`os.Readlink` text on error). Then set `blocked = true` and `continue`. In
`Link`, skip the remove-and-recreate when `os.Readlink(target)` already equals
`src`, so a correct link stays untouched.

Update every caller to compile. For now, pass a temporary `func(string, string) bool { return true }`
from `internal/cli` and `internal/tui`; steps 2–3 replace it.

**Verify**: `go build ./... && go vet ./...` → exit 0.

### Step 2: Implement the rule in `refresh`

In `internal/refresh/refresh.go`, add an exported constructor:

```go
// Ownership returns the rule that decides which existing symlinks belong
// to r: dangling links, links that resolve inside r.SrcDir, and links to a
// directory whose SKILL.md matches a version r knows.
func Ownership(r skills.Repo) links.Owns
```

Cache `KnownVersions` per skill name inside the closure. `refresh` will now
import `internal/links`; `links` must not import `refresh` (check with
`go build ./...`, which fails on an import cycle). Resolve `r.SrcDir` with
`filepath.EvalSymlinks` once and compare with
`strings.HasPrefix(resolved, srcResolved+string(filepath.Separator))`.

**Verify**: `go build ./...` → exit 0.

### Step 3: Make `Apply` act only on skills whose link state changes

Change the signature to:

```go
func Apply(r skills.Repo, p skills.Paths, before, staged []skills.Entry, owns Owns, warn Warn) ([]skills.Entry, error)
```

Match entries by `Name` (build a map from `before`). Skip filesystem work for
an entry when `before.State.Linked() == staged.State.Linked()`, and copy the
staged state to the output. Moving between `on`, `name-only` and
`user-invocable-only` changes only settings.json. Keep the existing rule
that a blocked unlink reports the entry as `On`.

Update callers:

- `internal/cli/set.go` `save`: take `before` as well, and pass `refresh.Ownership(r)`.
- `internal/tui/review.go` `save()`: pass `m.before`, `m.staged`, `refresh.Ownership(m.repo)`.
- `internal/cli/sync.go`: keep linking every non-off entry, which is sync's documented job, but pass `refresh.Ownership(r)` to `links.Link`.

**Verify**: `go build ./... && go test ./...`. Expect failures **only** in
`TestLinkCreatesBothAndRelinks` and `TestSyncRepairsStaleLinksAndPrunesDanglingOnes`;
step 4 fixes them. Any other failure is a STOP condition.

### Step 4: Update the two tests that encoded the old behaviour

- `internal/links/links_test.go` `TestLinkCreatesBothAndRelinks`: replace the
  "even to somewhere else, is replaced" block. Assert instead that a link to
  `/elsewhere` (dangling, so owned) **is** replaced. Then add a new test
  `TestLinkLeavesLinkOwnedElsewhere`: create a real directory outside the repo
  with a `SKILL.md` of different content, symlink `<LinkDir>/alpha` to it, and
  call `Link` with the test predicate `func(name, target string) bool { return false }`.
  Assert `blocked == true`, one warning containing `owned elsewhere`, and the
  link unchanged. Use plain predicates in `links` tests; `links` tests must
  not import `refresh`.
- `internal/cli/cli_test.go` `TestSyncRepairsStaleLinksAndPrunesDanglingOnes`:
  write the alpha SKILL.md body (`"---\nname: alpha\ndescription: alpha skill\n---\n"`,
  the same bytes `newHarness` writes) into the old checkout dir before running
  `sync`, so it matches the current version (rule 3).

**Verify**: `go test ./...` → all `ok`.

### Step 5: Add the regression tests that motivate this plan

In `internal/cli/cli_test.go`, modelled on `TestRealClaudeDirectoryIsPreserved` (line 246):

1. `TestSyncLeavesOtherRepoLinkAlone`: create `<base>/other-repo/skills/alpha/SKILL.md` with content `"other\n"`. Symlink `h.claudeLinks/alpha` to it, and symlink `h.links/alpha` into this repo. Run `h.ok("sync")`. `sync` touches every on skill; `add` would skip alpha because its state did not change. Assert that `h.claudeLinks/alpha` still resolves to the other repo, and that the output contains `owned elsewhere`.
2. `TestAddTouchesOnlyTheNamedSkill`: with alpha linked only in `h.links` (no Claude link), run `h.ok("add", "beta")`. Assert `h.claudeLinks/alpha` does **not** exist, which proves Apply skipped the unchanged skill. Then run `h.ok("sync")` and assert it now exists.
3. `TestRemoveLeavesOtherRepoLinkAlone`: same setup as 1, run `h.ok("remove", "alpha")`. Assert that the Claude link still exists and resolves to the other repo.

**Verify**: `go test ./internal/cli/ -run 'TestSyncLeaves|TestAddTouches|TestRemoveLeaves' -v` → 3 `PASS`.

### Step 6: Give refresh a non-failing `Foreign` status

In `internal/refresh/refresh.go`:

- Add `Foreign Status = "foreign" // a working link into another tree; left untouched`.
- In `Run`, before the `KnownVersions` check, add a branch. When `target` is a symlink, `filepath.EvalSymlinks(target)` succeeds, the resolved path is not inside the resolved `r.SrcDir`, and its SKILL.md is not a known version, set `Status = Foreign` and `Source` to the resolved path. Do not set `failed`. A dangling link stays `Review`, as before.
- `Failed()` is unchanged; `Foreign` is not in it.

In `internal/cli/refresh.go`, print `foreign  <target> -> <source>; owned elsewhere, left untouched` to **stdout**. Update the command's `Long` help to mention it. If `internal/tui/refresh.go` renders statuses with a `switch`, add a matching case showing the same words.

Test: in `cli_test.go`, modelled on `TestRefreshUnrecognisedCopyAndBrokenLinkArePreserved` (line 472), add `TestRefreshReportsForeignLinkWithoutFailing`. Put a symlink in a root pointing to an outside dir whose SKILL.md is `"someone else\n"`. Assert exit 0, the `foreign  ` line on stdout, and the link unchanged.

**Verify**: `go test ./...` → all `ok`.

### Step 7: Record the decision

Create `docs/adr/0005-links-owned-elsewhere-and-the-agents-directory.md`, in the style of `docs/adr/0004-port-install-and-refresh-to-go.md`: a title, prose paragraphs, and a `## Consequences` section, with no front matter. State:

- `~/.agents/skills` decides whether a skill is installed. `${CLAUDE_CONFIG_DIR:-~/.claude}/skills` mirrors it, and `sync` fills missing mirror links. This supersedes the "source of truth" sentence in 0003.
- The ownership rule, verbatim from this plan.
- Commands change only the skills whose state changed; `sync` is the only command that touches every on skill.
- Why: another tool (claude-custom) links same-name skills into the Claude directory.

In ADR 0003, add one line under the title: `Amended by [0005](0005-links-owned-elsewhere-and-the-agents-directory.md): the agents directory decides installation, and links owned elsewhere are left alone.`

In `README.md`, change "Real directories and files in either skills directory are never replaced or deleted." to also cover symlinks owned by another tool, and add one sentence about `foreign` to the `refresh` paragraph.

**Verify**: `grep -n "owned elsewhere" README.md docs/adr/0005-*.md` → at least one line in each file. Then run `pnpm exec prettier --check README.md docs/adr/` → exit 0. If `node_modules` is missing, run `pnpm install --frozen-lockfile` first.

## Test plan

New tests: `TestLinkLeavesLinkOwnedElsewhere` (links), `TestSyncLeavesOtherRepoLinkAlone`, `TestAddTouchesOnlyTheNamedSkill`, `TestRemoveLeavesOtherRepoLinkAlone`, `TestRefreshReportsForeignLinkWithoutFailing` (cli). Changed: `TestLinkCreatesBothAndRelinks`, `TestSyncRepairsStaleLinksAndPrunesDanglingOnes`. Pattern: `internal/cli/cli_test.go` harness (`newHarness`, `h.ok`, `symlink`, `resolves`). `go test ./...` must be all `ok`.

## Done criteria

- [ ] `go build ./... && go vet ./...` exit 0; `gofmt -l .` prints nothing
- [ ] `go test ./...` all `ok`, including the 5 new tests
- [ ] `grep -n "even to somewhere else, is replaced" internal/links/links_test.go` → no match
- [ ] `grep -n "Foreign" internal/refresh/refresh.go` → at least 2 matches
- [ ] `docs/adr/0005-links-owned-elsewhere-and-the-agents-directory.md` exists; ADR 0003 has the "Amended by" line
- [ ] `git status --short` lists only in-scope files
- [ ] `plans/README.md` row 010 updated

## STOP conditions

- The excerpts above don't match the live code.
- `go build` reports an import cycle between `links` and `refresh` that you can't remove without moving `KnownVersions`. Report it; don't move code across packages on your own.
- Any existing test other than the two named in step 3 fails after step 3.
- Satisfying the rule seems to need changes in `internal/settings/` or `internal/skills/state.go`.

## Maintenance notes

- This is half of the contract with `~/src/claude-custom`. That repo already skips links it does not own (`bin/claude-custom` `place_link`). After this lands, neither tool overwrites the other. The names `handoff` and `pr-draft` still collide; which repo keeps each name is the maintainer's decision, recorded in `plans/README.md`.
- Rule 3 runs `git log` and `git show` per skill only for links that don't already point into the Repo, and the result is cached per name.
- Deferred: `refresh.relink` moves a symlink into `skill-backups/` by `os.Rename`. A relative link then breaks, and the "backup" holds no content. It is rare now that links into other trees are `Foreign`, but a recognised link into another tree still takes that path.
- Reviewers: check that `Apply` no longer touches unchanged skills (test 2), and that nothing deletes a symlink without asking `owns`.
