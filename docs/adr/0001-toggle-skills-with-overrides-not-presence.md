# Toggle skills with Overrides, not by adding and removing Links

The installer links every skill in the repo into `~/.claude/skills` and then controls visibility through `skillOverrides` in `settings.json`, rather than creating a Link only for the skills you want and deleting the rest.

Deleting a directory to express "I don't want this one" destroys the choice itself: after the next `git pull` the skill reappears with no record that it was ever turned off, and there is nothing to reverse. An Override is a stated preference that survives updates and can be read back.

It also buys expressiveness the filesystem cannot offer. Presence is binary; `skillOverrides` has four states, and `user-invocable-only` — you can `/invoke` it, the model cannot reach for it — is exactly what a skill with outward-facing side effects wants.

## Consequences

Every skill in the repo is linked on every machine, so `ls ~/.claude/skills` no longer tells you what is active. `./install --list` does. Skills also stay visible to anything that reads the directory without consulting `settings.json`.
