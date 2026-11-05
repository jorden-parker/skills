# Plan 018: Browse every skill on skills.sh in the garden and install any of them

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 12a03e1..HEAD -- internal README.md CONTEXT.md`
> Plan 017 must be DONE first. It adds `Scope` to `skills.Paths`, a
> `--scope` flag and a `p` key in the garden; expect those changes. For any
> other change in an in-scope file, compare the "Current state" excerpts with
> the live code; on a mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: MED (it writes downloaded third-party files to disk and links them into the user's agents)
- **Depends on**: `plans/017-add-global-or-project-scope.md`
- **Category**: direction
- **Planned at**: commit `12a03e1`, 2026-11-05

## Why this matters

The garden only lists the skills in this repo's `catalog/`. The maintainer
wants to see every skill published on **skills.sh** (the public skills
registry behind `npx skills`) in the garden. They want to preview any of
them and turn any of them on, the same way as a catalog skill, in the scope
plan 017 added.

Decisions already made by the maintainer (do not revisit them):

1. **Files go to a download cache, then get a Link.** A registry skill
   downloads into
   `${XDG_DATA_HOME:-$HOME/.local/share}/skills/registry/<owner>/<repo>/<skill>/`.
   It is linked exactly like a catalog skill, and all four States work. No
   Node and no `npx`. Nothing is written into this repo.
2. **The list comes from the skills.sh sitemaps, cached for a day.** There
   is no "list everything" API. A new garden view shows every ID from the
   sitemaps, filterable with the existing `/`. `SKILL.md` for the preview is
   fetched only when a row is selected.

## Facts about skills.sh (verified 2026-11-05 with curl)

- `https://skills.sh/` redirects to `https://www.skills.sh/`. Use
  `https://www.skills.sh` as the base URL.
- `GET /sitemap.xml` is a `<sitemapindex>`:

  ```xml
  <sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <sitemap><loc>https://www.skills.sh/sitemap-misc.xml</loc></sitemap>
    <sitemap><loc>https://www.skills.sh/sitemap-owners.xml</loc></sitemap>
    <sitemap><loc>https://www.skills.sh/sitemap-skills-1.xml</loc></sitemap>
    <sitemap><loc>https://www.skills.sh/sitemap-skills-2.xml</loc></sitemap>
  </sitemapindex>
  ```

- Each `sitemap-skills-N.xml` is a `<urlset>` of
  `<url><loc>https://www.skills.sh/<owner>/<repo>/<skill></loc></url>`, for
  example `https://www.skills.sh/vercel-labs/skills/find-skills`. Together
  they hold 20,000 IDs, all with exactly three path segments. 10,000 per file
  suggests the sitemap is capped, so "every skill" means every skill the
  sitemap publishes. One ID contains parentheses:
  `obra/superpowers-skills/test-driven-development-(tdd)`. 1,396 skill names
  (the last segment) are shared by more than one ID.
- `robots.txt` allows `/` and the sitemaps and disallows `/api/` for
  crawlers. Fetching one skill's files when the user selects or installs it
  is what the official `npx skills` CLI does (`fetchSkillDownload` in
  `skills@1.7.6`). Never crawl `/api/`: no prefetching, no bulk downloads.
- `GET /api/download/<owner>/<repo>/<skill>` (each segment URL-path-escaped)
  returns JSON `{"files":[{"path":"SKILL.md","contents":"…"}, …],"hash":"…"}`.
  `path` is relative and slash-separated (seen: `SKILL.md`, `rules/_sections.md`),
  and `contents` is a UTF-8 string. Sizes vary: `vercel-labs/skills/find-skills`
  is about 5 KB, while `vercel-labs/agent-skills/vercel-react-best-practices`
  is about 240 KB. An unknown ID returns HTTP 404.
- `GET /api/search?q=…` exists (two or more characters, at most 200 results).
  **Do not use it** in this plan.

## Current state

- `internal/skills/repo.go`: `Repo{Dir, SrcDir, OldSrcDir}`. Key methods:

  ```go
  func (r Repo) Names() ([]string, error)          // catalog IDs "author/name", sorted
  func (r Repo) Has(id string) bool                // id is author/name with a SKILL.md
  func validSegment(s string) bool                 // not "", ".", "..", no / or \
  func LinkName(id string) string { return path.Base(id) }
  func (r Repo) Resolve(arg string) (string, error) // exact ID, or a unique bare name
  func (r Repo) Source(id string) string { return filepath.Join(r.SrcDir, filepath.FromSlash(id)) }
  func (r Repo) Describe(name string) string       // opens filepath.Join(r.SrcDir, name, "SKILL.md")
  func (r Repo) Manifest(name string) (string, error) // reads filepath.Join(r.SrcDir, name, "SKILL.md")
  ```

  `Find(flag)` builds the `Repo`. `Source` is what every Link points at:
  `links.Apply` (`internal/links/links.go:85`), `internal/cli/sync.go:43`,
  and the TUI preview (`internal/tui/app.go:403`).

- `internal/skills/state.go`: `LoadStates(r, p)` iterates `r.Names()`. A skill
  is linked when `p.LinkDir/<link name>` exists, unless that link resolves to
  a same-name sibling in `ids` (`linkedTo`, lines 105–117). `Stage(entries, id, st)`
  turns off every other entry with the same link name when one goes on.
- `internal/tui/app.go`:
  - The `model` holds `before` and `staged []skills.Entry` (same order),
    a `list.Model` of `item{idx, name, desc}` (`delegate.go:16`), and a
    `preview viewport.Model`.
  - `reload()` (line 121) resets `before`/`staged` from `LoadStates`.
    `items()` (line 128) builds one row per staged entry.
  - `stageAllOn()` (line 319) and the `allOff` case (line 274) loop over
    **all** of `m.staged`.
  - `header()` (line 418) shows `growing() of len(m.staged)`.
  - `refreshPreview()` (around line 380) reads `m.repo.Manifest(e.Name)`
    synchronously.
  - The async pattern to copy: `internal/tui/refresh.go` defines
    `refreshMsg`, and `refreshCmd` returns a `tea.Cmd` that does slow work
    off the UI loop and comes back through `Update` (`case refreshMsg:` at
    `app.go:191`).
- `internal/tui/review.go:38–60`, `save()`: calls
  `links.Apply(m.repo, m.paths, m.staged, warn)`, then
  `settings.WriteOverrides`, then `m.reload()`.
- `internal/cli/set.go:101–108`, `save()`: the same `Apply` + `WriteOverrides`
  for `add`, `remove` and `set`. `applyStates` resolves each argument with
  `r.Resolve`.
- `internal/atomicfile.Write(path, data)` writes a file atomically. Use it for
  the index cache file.
- No HTTP code exists yet. Use only the standard library (`net/http`,
  `encoding/xml`, `encoding/json`, `net/url`). **Add no dependencies.**

### Conventions

- Tests come first; the maintainer's hook blocks code edits until a test in
  the package was edited and run. New code has **no comments**; the hook
  rejects them. Use clear names instead.
- Test patterns: `internal/cli/cli_test.go` `newHarness`/`h.ok`,
  `internal/tui/app_test.go` `newFixture`/`f.open()`/`press`. Use
  `net/http/httptest` for every network test. **No test may hit the real
  skills.sh.**
- Vocabulary (`CONTEXT.md`): **Skill**, **Catalog**, **Link**, **Link name**,
  **State**, **Scope**. Add **Registry**, **Registry ID** and
  **Download cache** (Step 8). Avoid "install" for a link in code names. The
  garden's user-facing text may say "install".

## Commands you will need

| Purpose | Command                                                 | Expected on success |
| ------- | ------------------------------------------------------- | ------------------- |
| Build   | `go build ./...`                                        | exit 0              |
| Vet     | `go vet ./...`                                          | exit 0              |
| Tests   | `go test ./...`                                         | every package `ok`  |
| One pkg | `go test ./internal/registry`                           | `ok`                |
| Format  | `gofmt -l .`                                            | no output           |
| Docs    | `pnpm exec prettier --check README.md CONTEXT.md plans` | exit 0              |

## Scope

**In scope:**

- `internal/registry/` (new package: `registry.go`, `registry_test.go`)
- `internal/skills/repo.go`, `state.go`, `skills_test.go`
- `internal/cli/root.go`, `set.go`, `sync.go`, `cli_test.go`
- `internal/tui/app.go`, `delegate.go`, `review.go`, `app_test.go`, and a new
  `internal/tui/registry.go` for the async messages
- `README.md`, `CONTEXT.md`

**Out of scope:**

- `internal/links/*`: it only needs `r.Source(id)` to return the right
  directory, which Step 2 provides.
- `internal/refresh/*`: refresh is about catalog skills. It uses `r.Names()`,
  which stays catalog-only.
- `/api/search`, install counts, ranking, and any background or bulk download.
- Updating already-downloaded registry skills (see Maintenance notes).
- Deleting downloads when a skill is turned off. The cache is shared across
  scopes: a skill off in project scope may still be linked globally.

## Git workflow

- Commit directly on `main`. Conventional Commits, for example
  `feat(registry): list skills.sh skills from its sitemaps`,
  `feat(tui): browse and install skills.sh skills`. One commit per step or
  per logical unit. The pre-commit hook runs build, vet and tests. Do not
  push.

## Steps

### Step 1: Add the registry locations to `skills.Paths`

Test first, in `skills_test.go`: with `HOME=/h` and no XDG variables,
`DefaultPaths()` has `RegistryDir == /h/.local/share/skills/registry` and
`RegistryIndex == /h/.cache/skills/skills-sh-index.txt`. With
`XDG_DATA_HOME=/d` and `XDG_CACHE_HOME=/c`, they become
`/d/skills/registry` and `/c/skills/skills-sh-index.txt`. Plan 017's
`PathsFor(ProjectScope, …)` keeps the same two values: the cache is shared.

Implement: add the fields `RegistryDir` and `RegistryIndex` to `Paths`, and
set them in `DefaultPaths()`. Treat an empty `XDG_*` variable as unset.

Also add `t.Setenv("XDG_DATA_HOME", "")` and `t.Setenv("XDG_CACHE_HOME", "")`
to `newHarness` (`cli_test.go`) and `newFixture` (`app_test.go`). Without
them, a developer's real XDG variables would point tests at real
directories.

Verify: `go test ./internal/skills` → `ok`.

### Step 2: Let `Repo` resolve Registry IDs

A **Registry ID** is `owner/repo/skill`: exactly three segments, each passing
`validSegment`. A catalog ID has two segments, so the two kinds never clash.

Test first, in `skills_test.go`, with a temp catalog plus a temp registry dir
that holds `acme/tools/lint/SKILL.md`:

- `IsRegistryID("acme/tools/lint")` is true. `"me/alpha"`, `"a/b/c/d"`,
  `"a/../c"` and `"a/b/"` are false.
- `r.Source("acme/tools/lint") == <registryDir>/acme/tools/lint`.
  `r.Source("me/alpha")` is unchanged.
- `r.Has("acme/tools/lint")` is true. `r.Has("acme/tools/missing")` is false.
- `r.Downloaded()` returns `["acme/tools/lint"]`. It ignores a directory
  without `SKILL.md`.
- `r.Describe` and `r.Manifest` work for the registry ID.
- `r.Resolve("acme/tools/other")` returns `"acme/tools/other"` even though it
  is not downloaded yet. `r.Resolve("lint")` still resolves only catalog
  names; a bare name never matches a registry skill.
- `LoadStates`: link `p.LinkDir/lint` to the registry dir. The result
  contains `{acme/tools/lint on}`, and a catalog skill also named `lint`
  shows `off`.

Implement in `repo.go`:

- Add `RegistryDir string` to `Repo`. `Find` sets it from
  `DefaultPaths().RegistryDir`, so the `Repo` value works unchanged in
  every caller.
- `IsRegistryID(id string) bool`.
- `Source`: `if IsRegistryID(id) { return filepath.Join(r.RegistryDir, filepath.FromSlash(id)) }`.
- `Has`: for a registry ID, check `<Source(id)>/SKILL.md` is a regular file.
- `Describe` and `Manifest`: read `filepath.Join(r.Source(name), "SKILL.md")`
  instead of joining `SrcDir`.
- `Downloaded() ([]string, error)`: walk exactly three levels under
  `RegistryDir` (`os.ReadDir` per level, as in `Names`). Keep IDs where `Has`
  is true and return them sorted. A missing `RegistryDir` returns `nil, nil`.
- `Resolve`: return `arg` immediately when `r.Has(arg) || IsRegistryID(arg)`.

Implement in `state.go`: `LoadStates` uses `ids = r.Names()` plus
`r.Downloaded()`. Leave the rest alone: `linkedTo` already compares against
every sibling in `ids`.

Verify: `go test ./internal/skills` → `ok`. Then `go test ./...` → still all
`ok` (catalog behaviour is unchanged).

### Step 3: Create `internal/registry`: index from the sitemaps

Write `internal/registry/registry_test.go` first. Use an `httptest.Server`
that serves:

- `/sitemap.xml`: a sitemapindex whose `<loc>`s use the **real host**
  (`https://www.skills.sh/sitemap-skills-1.xml`, `.../sitemap-skills-2.xml`,
  `.../sitemap-owners.xml`). This proves the client fetches only from its own
  `BaseURL` path.
- `/sitemap-skills-1.xml` and `-2.xml`: urlsets with IDs that include
  `obra/superpowers-skills/test-driven-development-(tdd)`, a duplicate, a
  two-segment URL `https://www.skills.sh/acme/tools` (dropped) and a
  `/a/../b` URL (dropped).
- Requests to `/sitemap-owners.xml` fail the test; that file must never be
  fetched.

Test cases for `Client.Index()`:

1. No cache file: it fetches, returns the sorted, de-duplicated, valid IDs,
   and writes them one per line to `RegistryIndex`.
2. A cache file younger than 24 hours: it returns the cached IDs, and the
   server sees **zero** requests.
3. A cache older than 24 hours (set its mtime with `os.Chtimes`) and a
   working server: it refetches.
4. A stale cache and a server returning 500: it returns the stale IDs and
   no error.
5. No cache and a server returning 500: it returns an error that contains
   `skills.sh`.

Implement `internal/registry/registry.go`:

```go
type Client struct {
	BaseURL   string
	IndexPath string
	HTTP      *http.Client
	Now       func() time.Time
}

func New(p skills.Paths) Client
func (c Client) Index() ([]string, error)
```

- `New` sets `BaseURL: "https://www.skills.sh"`, `IndexPath: p.RegistryIndex`,
  `HTTP: &http.Client{Timeout: 30 * time.Second}`, `Now: time.Now`.
- Use one helper to fetch a path: `GET BaseURL + path`, require status 200,
  read at most 16 MiB (`io.LimitReader`), and return
  `fmt.Errorf("skills.sh: GET %s: %s", path, resp.Status)` on a bad status.
- Sitemap index: `xml.Unmarshal` into
  `struct{ Sitemaps []struct{ Loc string `xml:"loc"`}`xml:"sitemap"` }`.
  For each loc, `url.Parse` it and keep `u.Path` only when
  `path.Base(u.Path)` matches `sitemap-skills-*.xml` (`path.Match`). Fetch
  `BaseURL + u.Path`. Never follow the loc's host.
- Urlset: `struct{ URLs []struct{ Loc string `xml:"loc"`}`xml:"url"` }`.
  Take `strings.Trim(u.Path, "/")` and keep it if `skills.IsRegistryID`.
- Freshness: `os.Stat(IndexPath)`. The cache is fresh when
  `Now().Sub(mtime) < 24*time.Hour`. Write it with `atomicfile.Write` after
  `os.MkdirAll(filepath.Dir(IndexPath), 0o755)`.

Verify: `go test ./internal/registry` → `ok`.

### Step 4: `internal/registry`: fetch and download one skill safely

This is a trust boundary: the files come from arbitrary GitHub repos through
skills.sh.

Tests first (`httptest`, with
`/api/download/acme/tools/lint` returning JSON):

1. `Files("acme/tools/lint")` returns the files. A request for
   `obra/superpowers-skills/test-driven-development-(tdd)` arrives at the
   server with each segment escaped by `url.PathEscape`. Assert on
   `r.URL.EscapedPath()`.
2. `Manifest(id)` returns the `SKILL.md` contents. When no `SKILL.md` is
   present, it returns an error containing `no SKILL.md`.
3. `Download(id, dest)` writes every file under `dest` with mode `0644`
   (directories `0755`), including the nested `rules/a.md`.
4. `Download` rejects the whole skill and leaves **nothing** at `dest`
   (and no temp dir beside it) when any path is `../evil`, `/etc/x`,
   `a/../../x` or empty, or when `SKILL.md` is missing. The error names the
   bad path. Use `filepath.IsLocal(filepath.FromSlash(p))` as the check.
5. `Download` over an existing `dest` replaces it completely: a stale file
   from the old copy is gone.
6. A 404 returns an error containing `not found on skills.sh`.
7. A non-registry ID (`"me/alpha"`) returns an error without any request.

Implement:

```go
type File struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

func (c Client) Files(id string) ([]File, error)
func (c Client) Manifest(id string) (string, error)
func (c Client) Download(id, dest string) error
```

`Download` validates every path first. It writes into
`os.MkdirTemp(filepath.Dir(dest), ".download-*")` (create the parent with
`0o755` first), then runs `os.RemoveAll(dest)` and `os.Rename(tmp, dest)`.
On any error, it removes the temp dir. Never set execute bits.

Verify: `go test ./internal/registry` → `ok`.

### Step 5: Download before linking, in the CLI

Test first, in `cli_test.go`: start an `httptest` server that serves the
download JSON for `acme/tools/lint`, and set `SKILLS_REGISTRY_URL` to its
URL (see below).

- `add acme/tools/lint` downloads into
  `<HOME>/.local/share/skills/registry/acme/tools/lint/SKILL.md`, then links
  `h.links/lint` and `h.claudeLinks/lint` to it. `list` shows
  `acme/tools/lint on`.
- `remove acme/tools/lint` removes the links and **keeps** the downloaded
  folder.
- `add acme/tools/missing` (server 404) fails with `not found on skills.sh`,
  and creates no link.
- `sync` relinks an on registry skill whose `~/.claude/skills` link was
  deleted. That case is a test for Step 6.

Implement:

- `registry.New` reads `SKILLS_REGISTRY_URL` and uses it as `BaseURL` when
  it is set. This is the only test seam for the CLI and TUI. Document it in
  the README in one line ("overrides the skills.sh base URL, for tests").
- `registry.FetchMissing(c Client, r skills.Repo, entries []skills.Entry) error`:
  for each entry with `State.Linked()`, a registry ID and `!r.Has(id)`, call
  `c.Download(id, r.Source(id))`. Stop at the first error.
- In `cli/set.go` `save()`, call `registry.FetchMissing` before
  `links.Apply`. If it fails, return the error before anything is linked.
- `registry` imports `skills`. `skills` must not import `registry`; keep it
  that way.

Verify: `go test ./internal/cli` → `ok`.

### Step 6: `sync` links registry skills to the right place

`internal/cli/sync.go:43` already calls `links.Link(r.Source(e.Name), …)`, and
`LoadStates` now includes downloaded registry skills, so Step 2 should make
this work as is. Write the test from Step 5's last bullet. If it passes with
no change to `sync.go`, make no change. `links.Prune` only removes dangling
links into the catalog; confirm that a registry link survives `sync`.

Verify: `go test ./internal/cli` → `ok`.

### Step 7: The garden: a skills.sh view, async preview, install on save

Tests first, in `app_test.go`, with an `httptest` server and
`SKILLS_REGISTRY_URL` set. Serve the sitemaps listing `acme/tools/lint` and
`acme/tools/me-alpha`, plus the download JSON for `acme/tools/lint`. Drive
async commands by running the returned `tea.Cmd` and feeding its message back
into `m.Update`. Add a small test helper `run(m, cmd)` that does this,
recursively, for `tea.Batch` results if needed.

1. Pressing `b` switches `m.view` to the skills.sh view and returns a command.
   Running it loads the index. The list then shows only registry IDs (2
   rows), and the header shows `skills.sh`. Pressing `b` again returns to the
   catalog view, which shows only catalog skills plus downloaded registry
   skills.
2. A second `b` → `b` → `b` in the same session does **not** fetch again (the
   server counts requests).
3. Selecting `acme/tools/lint` in the skills.sh view triggers a preview fetch.
   After it runs, the preview contains the fetched `SKILL.md` text and
   `skills.sh/acme/tools/lint`. Selecting it again uses the in-memory cache,
   with no new request.
4. `space` on `acme/tools/lint`, then `s` and `enter`: the folder is
   downloaded, both links exist, and after reload the catalog view lists
   `acme/tools/lint` as `on`.
5. In the skills.sh view, `a` and `n` change nothing and set a status that
   contains `catalog view`. In the catalog view, `a` does not turn on any
   undownloaded registry entry. Assert that every registry entry not
   downloaded is still `off`.
6. The header count in the catalog view is unchanged by loading the index:
   `2 of 2 growing` stays `of 2`.
7. Index load fails (server 500, no cache): the view stays usable, and the
   status shows the error.
8. Pressing `R` (reload) in the skills.sh view keeps the registry rows.

Implement:

- `model` gets `registry registry.Client` (set in `newModel` from
  `registry.New(p)`), `view` (`catalogView` / `registryView`),
  `registryIDs []string`, `manifests map[string]string`, and
  `loadingIndex bool`.
- New keymap entry `browse`: `key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "skills.sh"))`.
  Add it to the footer hints and to the `?` screen's `Move` group as
  `{"b", "browse every skill on skills.sh, or back to the catalog"}`.
- New file `internal/tui/registry.go` with:
  - `indexMsg{ids []string; err error}` and `loadIndex() tea.Cmd`, following
    `refreshCmd` in `refresh.go`.
  - `manifestMsg{id, body string; err error}` and `fetchManifest(id) tea.Cmd`.
  - `appendRegistryEntries()`: for each ID in `m.registryIDs` not already in
    `m.staged`, append `skills.Entry{Name: id, State: skills.Off}` to **both**
    `before` and `staged`, so `pending()` is unaffected and indexes stay
    stable.
  - `visible(e skills.Entry) bool`: in the catalog view, true for
    non-registry IDs, for registry IDs where `m.repo.Has(id)`, and for any
    entry whose staged state is linked. In the skills.sh view, true for
    registry IDs.
- `Update`: handle `indexMsg`. On success, store the IDs, call
  `appendRegistryEntries()`, then `resetList()`. On error, set an error
  status. Handle `manifestMsg`: store it in `m.manifests` (also store errors
  as their text) and refresh the preview if that ID is still selected.
- `reload()`: after `LoadStates`, call `appendRegistryEntries()` when
  `m.registryIDs` is set.
- `items()`: emit rows only for `visible` entries, keeping `idx` as the index
  into `m.staged`. For registry IDs that are not downloaded, use the
  description `skills.sh · <owner>/<repo>`; do not call `Describe` on 20,000
  missing files.
- `stageAllOn`, the `allOff` case, `growing()` and the header total: count
  and change only entries visible in the catalog view. In the skills.sh view,
  `a` and `n` set the status
  `a and n work in the catalog view only. Press b to go back.`.
- `refreshPreview()`: for a registry ID that is not downloaded, show
  `skills.sh/<id>` as the source line, and as the body show `m.manifests[id]`
  or `Loading SKILL.md from skills.sh…`. If it is not cached and not already
  requested, return `fetchManifest(id)`. **`refreshPreview` currently returns
  nothing; give it a `tea.Cmd` return and pass it up from every caller.**
  Downloaded registry skills read from disk like catalog skills.
- `review.go` `save()`: call `registry.FetchMissing(m.registry, m.repo, m.staged)`
  before `links.Apply`. On error, call `setStatus("Download failed: "+err.Error(), true)`,
  stay in the garden, and apply nothing. This blocks the UI while it
  downloads. That is acceptable here: it is a handful of small files, with a
  30 s timeout per request.
- Title (`applyStyles`, line 117): `%d skills` counts visible rows.

Verify: `go test ./internal/tui` → `ok`. Then a manual smoke test against the
real site, run by you in a terminal and not added to the tests:
`go build -o /tmp/skills-018 . && SKILLS_REPO=$PWD /tmp/skills-018`. Press
`b`; the list fills within a few seconds. Press `/`, type `find-skills`, and
check the preview loads. Quit with `q` **without saving**. Delete
`/tmp/skills-018`.

### Step 8: Document

- `CONTEXT.md` under `### Distribution`, after **Link**:

  ```
  **Registry**:
  skills.sh, the public index of skills published from GitHub repos. The garden lists every skill its sitemaps publish.
  _Avoid_: marketplace, store, remote

  **Registry ID**:
  A registry skill's ID, `owner/repo/skill`. Three segments, so it never clashes with a Catalog ID.
  _Avoid_: slug, package

  **Download cache**:
  `${XDG_DATA_HOME:-~/.local/share}/skills/registry/`, where registry skills are written before they are linked. Shared by both Scopes; turning a skill off keeps its files.
  _Avoid_: vendor, install dir
  ```

- `README.md`:
  - Under `## Commands`, add
    `skills add vercel-labs/skills/find-skills   # download from skills.sh and link it`.
  - In the garden key table, add `| \`b\` | Browse every skill on skills.sh; \`space\` and \`s\` install one like any catalog skill |`.
  - Add a `### Skills from skills.sh` subsection. It covers where the files
    go; that the index is cached for a day at
    `${XDG_CACHE_HOME:-~/.cache}/skills/skills-sh-index.txt`; that downloads
    are not updated automatically (delete the folder and turn the skill on
    again); that files lose execute bits; that `SKILLS_REGISTRY_URL`
    overrides the base URL; and one sentence of caution: registry skills are
    third-party instructions and scripts, so read the preview before turning
    one on.

Verify: `pnpm exec prettier --check README.md CONTEXT.md` → exit 0.

### Step 9: Full gate

`go build ./... && go vet ./... && go test ./... && gofmt -l .` → exit 0,
every package `ok`, no gofmt output.

## Test plan

Covered per step. New test file: `internal/registry/registry_test.go`, using
`httptest` only. The `atomicfile`, `cli` and `tui` test files show the repo's
table-free, one-behaviour-per-test style; match it. `go test ./...` must make
no real network calls. Check by running it with networking off, or confirm
that every `registry.Client` in the tests has a `BaseURL` from
`httptest`/`SKILLS_REGISTRY_URL`.

## Done criteria

- `go test ./...` → every package `ok`; `go vet ./...` and `gofmt -l .` are clean.
- `grep -rn "skills.sh" internal --include=*.go | grep -v _test.go | grep -c "https://www.skills.sh"`
  prints `1`: the base URL appears once.
- `grep -rn "/api/search" internal` prints nothing.
- `go.mod` is unchanged (`git diff --quiet 12a03e1 -- go.mod go.sum`, or
  unchanged since plan 017's commit).
- The manual smoke test in Step 7 worked.
- The README and CONTEXT text from Step 8 is present. The `plans/README.md`
  row for 018 says DONE.

## STOP conditions

- Plan 017 is not DONE.
- The live sitemap or download response differs from "Facts about
  skills.sh". For example, `/sitemap.xml` has no `sitemap-skills-*.xml`
  entries, or the download JSON keys are not `files`/`path`/`contents`.
  Report what you saw.
- Filtering the 20,000-row skills.sh view with `/` takes more than about one
  second per keystroke in the smoke test. Report it; do not add paging or a
  search API on your own.
- Supporting registry IDs would need a change in `internal/links` or
  `internal/refresh`.
- Any test would need to reach the real skills.sh.

## Maintenance notes

- **Updates are deliberately absent.** The download JSON has a `hash`. A
  later plan could store it in the cache folder and offer "update" when the
  hash changes. `refresh` is the natural home, but it currently recognises
  only catalog history.
- **Security review focus:** the path checks in `Download` (Step 4, case 4),
  the 16 MiB response cap, the rule that index fetches never follow a
  sitemap's host, and the absence of execute bits. Registry content is
  untrusted. The tool never runs it, but an agent that loads it might, which
  is why the README asks users to read the preview first.
- The sitemap appears capped at 10,000 IDs per file. If skills.sh adds
  `sitemap-skills-3.xml`, the `sitemap-skills-*.xml` match picks it up
  without code changes.
- Link-name collisions are common (1,396 shared names). `skills.Stage`
  already keeps one skill per link name on. Reviewers should check that the
  garden makes clear which `owner/repo` is on.
- Plan 010 ("leave links owned by other tools alone") also edits
  `LoadStates`/`links`. Whichever lands second must keep registry links
  counted as links this tool owns: they resolve under `RegistryDir`.
