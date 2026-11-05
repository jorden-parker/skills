# Group skills by author; one skill per name is on

`skills/` was flat, so a folder name was both a skill's identity and the name of its link in `~/.agents/skills` and `~/.claude/skills`. That ruled out keeping Matt Pocock's `handoff` and writing our own `handoff` beside it.

Skills now live at `skills/<author>/<name>/`: `jorden/` for this repo's own, `mattpocock/` and `shadcn/` for upstream snapshots. A skill's ID is `author/name`; the repo, the CLI and the garden use it. The link name is still `name`, and so is the `skillOverrides` key, so upstream `SKILL.md` files stay unedited and `/handoff` keeps its name.

## One skill per name

Two skills with the same link name cannot both be linked, so turning one on turns its siblings off. Every staging path goes through `skills.Stage`; the garden's "all on" skips names a sibling already holds. A link that resolves to a sibling's source reads as off for this skill; a link anywhere else keeps the old "exists means on" rule. `links.Apply` unlinks off entries before linking the rest, so a swap never removes the new link.

On the command line a bare name works when it is unique (`skills add geist`); an ambiguous one is an error listing the IDs.

## Migration

Links made before the move point at `skills/<name>` and dangle. `sync` and `refresh` relink each one to the only skill with that name before pruning, so existing on-states survive. `refresh` follows `SKILL.md` through the rename (Git's `--follow`), so copies of versions saved before the move are still recognised. A dangling flat link whose name now has two authors is left for `sync` to prune.

## Consequences

Tests and scripts that addressed skills by path use the author folder. Adding a same-name skill is a new folder under its author; nothing else changes.
