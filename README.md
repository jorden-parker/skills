# Skills

Claude Code skills, cloned onto a machine and switched on selectively.

```sh
git clone <this repo> ~/src/skills
cd ~/src/skills
./install
```

The installer symlinks every skill into `~/.claude/skills`, then records which ones are active in `~/.claude/settings.json`. Nothing is copied, so `git pull` updates a skill in place.

**Adding or removing a skill needs a new Claude Code session.** Edits to an existing skill are picked up by `/reload-skills`.

## Commands

```sh
./install                    # pick which skills are on
./install --list             # show current state
./install --sync             # link new skills, keep states, no prompts
./install --set geist=off    # set one directly
./install --unlink geist     # remove the symlink entirely
```

`--set` takes `on`, `off`, `name-only` (the model sees the name but not the description — cheaper context) or `user-invocable-only` (you can `/invoke` it, the model cannot reach for it on its own).

The picker uses `fzf` when it is installed and falls back to a numbered menu when it is not. In the `fzf` picker: `TAB` toggles without moving the cursor, `ctrl-a` selects all, `ctrl-d` selects none, `ENTER` confirms. Everything selected is on; everything unselected is off. Confirming an empty selection changes nothing. JSON edits go through `jq`, or `python3` if `jq` is missing.

## Skills

| Skill                          | What it does                                                                            |
| ------------------------------ | --------------------------------------------------------------------------------------- |
| [`pr-draft`](skills/pr-draft/) | Opens a draft PR for the current branch, or updates the existing one. `/`-invoked only. |
| [`geist`](skills/geist/)       | Vercel's Geist Design System — real tokens, type scale, component recipes.              |

## Repo layout

```
skills/<name>/SKILL.md   the skills themselves
install                  the installer
CONTEXT.md               glossary
docs/adr/                decisions worth remembering
```

## Notes

- Nothing here writes to a global or user-level `CLAUDE.md`. The `geist` skill will offer to add one line to a _project's_ `CLAUDE.md`, and asks first.
- No Claude Code or Anthropic attribution is added to commits or pull requests.
