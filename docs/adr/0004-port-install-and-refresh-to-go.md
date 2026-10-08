# Port `install` and `refresh` to one Go command

The installer was a Bash script that wrapped `gum`, `fzf` or a numbered menu for the interactive garden and shelled out to `jq` or `python3` to edit `settings.json`. `refresh` was a separate Python script. Tests drove both through subprocesses with a fake `gum`. Every machine needed the right mix of Bash, Python, Gum and jq before the picker looked the same twice.

Both scripts are now one Go binary, `skills`, built on Charm (Bubble Tea, Bubbles, Lip Gloss, Huh, Fang) and laid out like `claude-config-cli`. Bare `skills` opens the garden; `list`, `add`, `remove`, `set`, `sync` and `refresh` do the same work without prompts and keep the old output lines so scripts and agents see no change. Refresh is also reachable from the garden. There are no fallbacks to other pickers.

## Where the checkout lives

Links point into `<checkout>/skills/<name>`, so the binary must know where the checkout is even when run from `~/.local/bin`. It resolves the path in this order: the `--repo`/`-C` flag, `$SKILLS_REPO`, the path baked in at build time by `install.sh` (`-ldflags -X …/internal/skills.DefaultDir=…`), then `~/src/skills`. A path without a `skills/` directory is an error that names the path and the two ways to fix it.

## Consequences

Go is the one build dependency; `install.sh` builds from a checkout or clones one into `~/src/skills` first. The binary is always built as `bin/skills`, because the `skills/` directory already owns the name at the repo root. Behaviour that mattered is kept and tested in Go: `CLAUDE_CONFIG_DIR` and `CODEX_HOME`, the `.bak` plus atomic write of `settings.json` with its key order preserved, never replacing or deleting a real directory or file, and refresh matching against the whole Git history. Exit codes still carry meaning (`refresh --check` exits 1 when work is needed, unknown names exit 2), and colour only appears on a terminal without `NO_COLOR`.
