# Skills

Claude Code skills, cloned onto a machine and switched on selectively.

```sh
curl -fsSL https://raw.githubusercontent.com/jorden-parker/skills/main/install.sh | bash
```

That clones the repo into `~/src/skills` (or `$SKILLS_REPO`), builds the `skills` command with Go, and installs it to `~/.local/bin`. From an existing checkout, run `./install.sh` instead; it installs that checkout. To build without installing: `go build -o bin/skills .`.

The command symlinks the skills you pick into both `~/.agents/skills` and `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills`. It reads `~/.agents/skills` to determine which skills are installed. Turning a skill off removes its links from both directories. Nothing is copied, so `git pull` updates a linked skill in place. Skills new to the repo stay off until you add them. `sync` also adds missing Claude links for enabled skills; it does not enable new skills. Existing installations outside these two directories are left untouched.

**Adding or removing a skill needs a new Claude Code session.** Edits to an existing skill are picked up by `/reload-skills`.

## Where skills write

Skills write only under `/tmp/claude-skills` (mode 700), never into the repository or its git dir: `commit/<repo-key>/`, `pr-draft/<repo-key>/`, `artifacts/` and `handoff/`. `<repo-key>` hashes the git dir and branch, so repos, worktrees and branches stay apart. Set `SKILLS_TMPDIR` to use another base, as the tests do.

## Commands

```sh
skills                        # open the garden: pick which skills are on
skills list                   # show current state
skills add geist              # link a skill
skills add geist --scope project   # link into the current Git project only
skills add vercel-labs/skills/find-skills   # download from skills.sh and link it
skills remove geist           # remove its links
skills set geist name-only    # set a partial state directly (also NAME=STATE)
skills sync                   # repair stale links, drop links to removed skills, no prompts
skills refresh                # refresh existing installations across agent apps
skills refresh new-project    # refresh one skill
skills refresh --check        # inspect without changing files
```

After pulling a change that moves skills, such as the move from `skills/` to `catalog/`, rebuild with `./install.sh` and run `skills sync`; it repoints old links.

`refresh` needs Git. It checks `~/.agents/skills`,
`${CODEX_HOME:-$HOME/.codex}/skills`, and
`${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills`. Recognised copies are backed up under
`skill-backups` beside the installation's `skills` directory, then replaced with
links to this checkout. Future repo edits are immediately available through those
links. Absent installations and app settings are left unchanged.

A copy is recognised when its `SKILL.md` matches the current source or a version
in this checkout's Git history. Unrecognised copies, including broken links, are
left untouched and reported for review. `--check` exits with status 1 when a
refresh or review is needed. Use a fresh agent session if it already loaded old
instructions.

`set` takes `on`, `off`, `name-only` (the model sees the name but not the description — cheaper context) or `user-invocable-only` (you can `/invoke` it, the model cannot reach for it on its own). `on` and `off` add or remove the link. The two partial states keep the link and write a Claude-specific `skillOverrides` entry in `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/settings.json`. `CLAUDE_CONFIG_DIR` selects both Claude’s settings directory and its skills directory. Real directories and files in either skills directory are never replaced or deleted.

Redirected output is plain text, and `NO_COLOR` disables colour. Bare `skills` without a terminal exits with an error; use the subcommands from scripts.

### Where the checkout lives

Links point into `<checkout>/catalog/<author>/<name>`, so the command needs to know where the checkout is. It looks, in order, at `--repo PATH` (or `-C PATH`), `$SKILLS_REPO`, the path baked in when `install.sh` built it, then `~/src/skills`. A path without a `catalog/` directory is an error that names the path.

### Scope

`--scope global` (the default) links into `~/.agents/skills` and Claude's skills directory, and keeps states in `settings.json`. `--scope project` links into the current Git project instead: `<git root>/.agents/skills` and `<git root>/.claude/skills`, with partial states in `<git root>/.claude/settings.local.json`. The garden switches between the two with `p`. Project scope needs the current directory to be inside a Git repository.

`skills refresh` works on global installs only. Project links are absolute symlinks into this checkout and are machine-local, so do not commit them.

### The skill garden

`skills` with no arguments opens the garden, a terminal UI built on Charm’s Bubble Tea and Lip Gloss. The left pane lists every skill with its state; the right pane previews the selected `SKILL.md`. Nothing changes on disk until you save.

| Key             | Does                                                                                                           |
| --------------- | -------------------------------------------------------------------------------------------------------------- |
| `↑` `↓` `j` `k` | Move between skills                                                                                            |
| `space` `tab`   | Turn an off skill on, or anything else off. Partial states stay as they are until toggled off                  |
| `enter` `t`     | Tune: pick `on`, `off`, `name-only` or `user-invocable-only`                                                   |
| `a` / `n`       | Stage every skill on (partial states kept) / every skill off                                                   |
| `s`             | Review the staged changes, then `enter` saves, `d` discards, `esc` returns                                     |
| `r`             | Refresh: show which installed copies would be replaced; `enter` applies, `f` scopes to the selected skill      |
| `R`             | Reload from disk and drop staged changes                                                                       |
| `p`             | Switch scope: global, or the current Git project (only when nothing is staged)                                 |
| `b`             | Browse every skill on skills.sh, or go back to the catalog; `space` and `s` install one like any catalog skill |
| `/`             | Filter by name or description                                                                                  |
| `ctrl+d`        | Show only the preview in a narrow terminal                                                                     |
| `?`             | Key reference                                                                                                  |
| `q` `esc`       | Quit; asks first when changes are pending                                                                      |

Saving writes the links and `settings.json`, keeps a `.bak` of the previous settings, and reminds you that a new Claude Code session is needed.

### Skills from skills.sh

`skills add owner/repo/skill` and the garden's `b` key reach every skill published on [skills.sh](https://www.skills.sh). `b` lists them, and `space` and `s` install one like any catalog skill. A registry skill is downloaded into `${XDG_DATA_HOME:-~/.local/share}/skills/registry/<owner>/<repo>/<skill>/` and then linked. The list of IDs comes from skills.sh's sitemaps and is cached for a day at `${XDG_CACHE_HOME:-~/.cache}/skills/skills-sh-index.txt`.

Downloads are not updated automatically. To fetch a newer copy, delete the skill's folder, then turn the skill off and on again. Downloaded files lose their execute bits. `SKILLS_REGISTRY_URL` overrides the skills.sh base URL, for tests.

Registry skills are third-party instructions and scripts, so read the preview before turning one on.

### PR description preview

`/pr-draft` describes the current working tree, committed or not, and renders the description as a local HTML page. It makes no Git or GitHub changes. It uses [a reusable body template](catalog/jorden/pr-draft/assets/body.md) when the target repository has none. Preview a body file directly:

```sh
pnpm install
./catalog/jorden/pr-draft/preview /tmp/claude-skills/pr-draft/pr-body.md --title "Cache the resolved config" --output /tmp/claude-skills/pr-draft/pr-preview.html
```

The draft follows the target repository's PR template, preserving its sections and checklists. Pass the completed body to preview it. Omit the body path to preview the local repository's template, using `--repo <path>` when outside that repository; the bundled template is the fallback when none exists. If several templates exist, pass the chosen file explicitly.

The command requires Python 3 and a one-time `pnpm install`. It renders the full description as HTML with copyable code blocks and writes a new file. Opening it is a separate `open <path>` step. It preserves existing files and makes no GitHub changes. The renderer and Geist styles are embedded; Google Fonts uses local fallbacks when offline.

### Artifacts

Skills show a result as a page through [one shared renderer](artifact/README.md): Markdown in, a standalone Geist-styled HTML file out, then opened with a separate `open <path>` step. `/web-research` uses its citation gate, which refuses to render until every `[n]` marker resolves to a numbered source with a URL.

```sh
./artifact/render findings.md --title "Which Node is LTS?" --citations
```

It needs Python 3 and the same one-time `pnpm install`.

## Skills

| Skill                                          | What it does                                                                                                    |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| [`pr-draft`](catalog/jorden/pr-draft/)         | Opens a draft PR for the current branch, or updates the existing one. `/`-invoked only.                         |
| [`geist`](catalog/jorden/geist/)               | Vercel's Geist Design System — real tokens, type scale, component recipes.                                      |
| [`new-project`](catalog/jorden/new-project/)   | Initialises tooling in an existing project or scaffolds a new one, inferred from the directory.                 |
| [`web-research`](catalog/jorden/web-research/) | Researches a question on the live web and shows the findings as a cited page. `/`-invoked only.                 |
| [`commit`](catalog/jorden/commit/)             | Sorts changes into Conventional Commits as copyable commands on a local page. `/`-invoked only.                 |
| [`handoff`](catalog/jorden/handoff/)           | Compacts the conversation into a handoff document for the next agent. Same name as Matt’s; one is on at a time. |

### Upstream skills

This checkout also includes all 29 published skills from [Matt Pocock](https://github.com/mattpocock/skills) and [shadcn’s `improve`](https://github.com/shadcn/improve). Matt’s `in-progress` directory is excluded. Each skill includes its supporting files and upstream MIT license.

These are editable snapshots under `catalog/mattpocock/<name>/` and `catalog/shadcn/improve/`, available through the same picker as the local skills. New skills stay off until selected:

```sh
skills                          # select the skills you want
skills add improve              # enable shadcn’s codebase advisor
```

[upstream-skills.json](upstream-skills.json) records the exact source commits and original paths. To update a snapshot, copy the full upstream skill directory and license, update its recorded commit, then run `skills refresh <name>` and `skills refresh --check <name>`.

For Matt’s engineering workflow, enable `setup-matt-pocock-skills` and run `/setup-matt-pocock-skills` once in the target project to configure its issue tracker and documentation layout. Start a fresh agent session after enabling skills.

### Authors and same-name skills

Each skill lives in its author's folder: `catalog/jorden/` for this repo's own, `catalog/mattpocock/` and `catalog/shadcn/` for upstream snapshots. A skill's ID is `author/name`, and it is linked as `name`. Two authors may ship the same name, such as `mattpocock/handoff` and `jorden/handoff`, but only one can be on at a time: turning one on switches its sibling off. A bare name works wherever it is unique:

```sh
skills add jorden/handoff       # switches mattpocock/handoff off
skills add geist                # unique, so the author can be left out
```

## Repo layout

```
catalog/<author>/<name>/  the skills themselves, grouped by author; linked into agent apps, so no tests here
main.go, internal/       the skills command (Go + Charm); each package's tests sit beside it
install.sh               builds and installs the command
artifact/                the shared Markdown-to-page renderer and its test.sh
test/<skill>/            Python tests for skill scripts; pnpm test runs them
CONTEXT.md               glossary
docs/adr/                decisions worth remembering
plans/                   implementation plans and their status
```

## Notes

- Nothing here writes to a global or user-level `CLAUDE.md`. The `geist` skill will offer to add one line to a _project's_ `CLAUDE.md`, and asks first.
- The pre-commit hook runs lint-staged (Prettier, gofmt), then `go build`, `go vet` and `go test`.
- Commit messages are Conventional Commits. The `commit-msg` hook runs `commitlint` and rejects anything else, so `Add a thing` fails and `feat: add a thing` passes.
- The `commit` and `pr-draft` skills add no Claude Code or Anthropic attribution to the commits and pull requests they draft.
