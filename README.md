# Skills

Claude Code skills, cloned onto a machine and switched on selectively.

```sh
git clone <this repo> ~/src/skills
cd ~/src/skills
./install
```

The installer symlinks the skills you pick into both `~/.agents/skills` and `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills`. It reads `~/.agents/skills` to determine which skills are installed. Turning a skill off removes its links from both directories. Nothing is copied, so `git pull` updates a linked skill in place. Skills new to the repo stay off until you add them. `--sync` also adds missing Claude links for enabled skills; it does not enable new skills. Existing installations outside these two directories are left untouched.

**Adding or removing a skill needs a new Claude Code session.** Edits to an existing skill are picked up by `/reload-skills`.

## Commands

```sh
./install                    # pick which skills are on
./install --list             # show current state
./install --add geist        # link a skill
./install --remove geist     # remove its link
./install --set geist=name-only  # set a partial state directly
./install --sync             # repair stale links, drop links to removed skills, no prompts
./refresh                   # refresh existing installations across agent apps
./refresh new-project       # refresh one skill
./refresh --check            # inspect without changing files
```

`refresh` requires Python 3 and Git. It checks `~/.agents/skills`,
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

`--set` takes `on`, `off`, `name-only` (the model sees the name but not the description — cheaper context) or `user-invocable-only` (you can `/invoke` it, the model cannot reach for it on its own). `on` and `off` add or remove the link. The two partial states keep the link and write a Claude-specific `skillOverrides` entry in `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/settings.json`. `CLAUDE_CONFIG_DIR` selects both Claude’s settings directory and its skills directory. Real directories and files in either skills directory are never replaced or deleted.

### The skill garden

Install [Charm’s Gum](https://github.com/charmbracelet/gum) for the interactive garden:

```sh
brew install gum
./install
```

- **Pick skills** opens a searchable multi-select with your enabled skills already marked. Type to search, use the arrow keys to move, `TAB` to toggle, and `ENTER` to stage your selection. Existing partial states are preserved.
- **Tune one skill** lets you choose any of the four states directly, including turning off the last enabled skill.
- **Review & save** shows the exact changes before writing them. Choose **Save changes** to apply them, or **Keep tending** to return.
- **Leave without saving** discards your staged changes. `Ctrl+C` cancels a picker; at the garden menu it exits without saving. `ESC` in search first moves focus to the results. An empty bulk selection changes nothing.

The garden uses Gum’s Bubble Tea interactions and Lip Gloss styling, with a live count of enabled skills and pending changes. Without Gum, the installer uses `fzf` or a numbered menu. In `fzf`, `TAB` toggles, `ctrl-a` selects all, `ctrl-d` selects none, and `ENTER` saves a nonempty selection. Command flags stay noninteractive. Redirected output is plain text, and `NO_COLOR` disables colour. JSON edits use `jq`, or `python3` when `jq` is unavailable.

### PR description preview

`/pr-draft` describes the current working tree, committed or not, and renders the description as a local HTML page. It makes no Git or GitHub changes. It uses [a reusable body template](skills/pr-draft/assets/body.md) when the target repository has none. Preview a body file directly:

```sh
pnpm install
./skills/pr-draft/preview /tmp/pr-body.md --title "Cache the resolved config" --output /tmp/pr-preview.html
```

The draft follows the target repository's PR template, preserving its sections and checklists. Pass the completed body to preview it. Omit the body path to preview the local repository's template, using `--repo <path>` when outside that repository; the bundled template is the fallback when none exists. If several templates exist, pass the chosen file explicitly.

The command requires Python 3 and a one-time `pnpm install`. It renders the full description as HTML with copyable code blocks, writes a new file, and opens it in your default browser. It preserves existing files and makes no GitHub changes. The renderer and Geist styles are embedded; Google Fonts uses local fallbacks when offline.

### Artifacts

Skills show a result as a page through [one shared renderer](artifact/README.md): Markdown in, a standalone Geist-styled HTML file out, opened in your default browser. `/web-research` uses its citation gate, which refuses to render until every `[n]` marker resolves to a numbered source with a URL.

```sh
./artifact/render findings.md --title "Which Node is LTS?" --citations
```

It needs Python 3 and the same one-time `pnpm install`.

## Skills

| Skill                                  | What it does                                                                                    |
| -------------------------------------- | ----------------------------------------------------------------------------------------------- |
| [`pr-draft`](skills/pr-draft/)         | Opens a draft PR for the current branch, or updates the existing one. `/`-invoked only.         |
| [`geist`](skills/geist/)               | Vercel's Geist Design System — real tokens, type scale, component recipes.                      |
| [`new-project`](skills/new-project/)   | Initialises tooling in an existing project or scaffolds a new one, inferred from the directory. |
| [`web-research`](skills/web-research/) | Researches a question on the live web and opens the findings as a cited page. `/`-invoked only. |

### Upstream skills

This checkout also includes all 29 published skills from [Matt Pocock](https://github.com/mattpocock/skills) and [shadcn’s `improve`](https://github.com/shadcn/improve). Matt’s `in-progress` directory is excluded. Each skill includes its supporting files and upstream MIT license.

These are editable snapshots under `skills/<name>/`, available through the same picker as the local skills. New skills stay off until selected:

```sh
./install                       # select the skills you want
./install --add improve         # enable shadcn’s codebase advisor
```

[upstream-skills.json](upstream-skills.json) records the exact source commits and original paths. To update a snapshot, copy the full upstream skill directory and license, update its recorded commit, then run `./refresh <name>` and `./refresh --check <name>`.

For Matt’s engineering workflow, enable `setup-matt-pocock-skills` and run `/setup-matt-pocock-skills` once in the target project to configure its issue tracker and documentation layout. Start a fresh agent session after enabling skills.

## Repo layout

```
skills/<name>/SKILL.md   the skills themselves
install                  the installer
artifact/                the shared Markdown-to-page renderer
CONTEXT.md               glossary
docs/adr/                decisions worth remembering
```

## Notes

- Nothing here writes to a global or user-level `CLAUDE.md`. The `geist` skill will offer to add one line to a _project's_ `CLAUDE.md`, and asks first.
- Commit messages are Conventional Commits. The `commit-msg` hook runs `commitlint` and rejects anything else, so `Add a thing` fails and `feat: add a thing` passes.
- No Claude Code or Anthropic attribution is added to commits or pull requests.
