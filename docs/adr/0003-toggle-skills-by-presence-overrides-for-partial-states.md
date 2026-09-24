# Toggle skills by Presence, with Overrides only for partial states

Supersedes [0001](0001-toggle-skills-with-overrides-not-presence.md).

The installer links a skill into `~/.claude/skills` when it is on and removes the link when it is off. `skillOverrides` in `settings.json` is used only for the two states the filesystem cannot express, `name-only` and `user-invocable-only`, and those keep the link.

0001 chose to link everything and hide with Overrides so that a choice survived `git pull`. In practice the Override was the less honest record: `ls ~/.claude/skills` showed skills that were off, tools that read the directory without consulting `settings.json` saw them, and an `off` entry for a skill that had left the repo lingered in settings with nothing to point at. Presence is what every consumer of the directory already agrees on, so it should be the switch.

The "reappears after pull" problem is solved by defaulting new skills to off. Nothing is linked until someone asks for it, so a pull adds nothing to the directory. `./install --sync` repairs links that point at an old checkout and removes links whose skill is gone from the repo; it never adds.

## Consequences

`ls ~/.claude/skills` is the source of truth for on and off again, and `./install --list` reads it rather than a separate record. Turning a skill off after a pull requires no action. Turning a new one on does, which is the intended trade. A real directory at a skill's path is never deleted or replaced, so a skill that was copied rather than linked shows as on until it is moved aside by hand or by `./refresh`.
