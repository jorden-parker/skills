# Rename the skills/ source folder to catalog/

The repo, the Go module, the `skills` command and the checkout (`~/src/skills`) are all called `skills`, and so was the folder that holds the skills: `~/src/skills/skills/<author>/<name>/`. The name was confusing, so the source folder is now `catalog/`. The outer names stay: renaming the module and checkout would touch every import and every machine's install for no gain.

Skills now live at `catalog/<author>/<name>/`. `Repo.SrcDir` points there, and `Repo.OldSrcDir` still points at `skills/`.

## Migration

Installed links point at `<checkout>/skills/<author>/<name>`, and old flat links point at `<checkout>/skills/<name>`. After the move they dangle. `sync` and `refresh` look under `OldSrcDir` as well as `SrcDir`: a link into the old folder is relinked to the same skill in `catalog/`, and `Prune` removes dangling links under both folders. `OldSrcDir` stays until no machine can still have links into the old folder.

## Consequences

After pulling this change, rebuild with `./install.sh` and run `skills sync`. Skill paths in docs and tooling use `catalog/`. Historical records such as ADR 0001 to 0005 keep the old path.
