---
name: commit
description: Organise the working tree's changes into Conventional Commits and open them as copyable commands on a local HTML page. Makes no Git changes.
argument-hint: "optional: hints on grouping, scope names, or an issue to reference"
disable-model-invocation: true
---

# Commit plan

Produce one thing: a commit plan for the current working tree, shown as an HTML page of copyable commands. The user runs the commands.

This skill reads only. It never stages, commits, amends or pushes, and it never edits `.git/index`. Do not run `git add`, `git commit`, `git reset` or `git stash` here, and do not run `git status` without `--no-optional-locks`.

Set `PLAN_DIR="$(<skill-directory>/plan dir)"`. It creates the directory under `/tmp/claude-skills/commit/` (or `$SKILLS_TMPDIR`). Everything the skill writes goes there, not in the tree or the git dir.

## Steps

1. **Gather.** Run `git --no-optional-locks status --porcelain=v2 --untracked-files=all`, `git diff HEAD --stat`, and `git diff HEAD`. Read untracked files that are not obviously generated. If nothing changed, say so and stop.

2. **Learn the repo's convention.** Read `.commitlintrc.json`, `commitlint.config.*`, `.commitlintrc*` or `package.json#commitlint` if present, and note `type-enum`, `scope-enum` and `header-max-length`. Read `git log --format=%s -30` for scope habits and casing. With nothing to go on, use the Conventional Commits defaults: `feat fix refactor perf test docs build ci chore style revert`. Keep subjects lowercase and without a final full stop.

3. **Ignore first.** Before grouping, find untracked or changed files that no repository should track: build output, caches, dependency folders, logs, editor and OS files (`.DS_Store`, `.idea/`), local env files, and output from tools run in this session (`.playwright-mcp/`). Read the existing `.gitignore` and do not repeat a pattern it already has. Prefer a folder or glob pattern (`dist/`, `*.log`) over single files. Put the patterns, each with a short reason, in `ignore`. If a tracked file matches, add it to `untrack` and say so in the report, because the next push removes it from the remote. Leave out anything a teammate might rely on, such as a shared editor config that is already committed.

4. **Group.** One logical change per commit:
   - Tests go in the same commit as the code they test, unless the change is test-only.
   - Lockfiles go with the manifest change that caused them.
   - Generated files go with their source.
   - Pure formatting or renames come before behaviour changes, as their own `style:` or `refactor:` commit.
   - Order commits so every prefix of the list builds: dependencies first.
   - Split within a file only when it clearly holds two unrelated changes. Then write `patch-NN.diff` into `PLAN_DIR`, using only the hunks for that commit, relative to the state after the previous commits. Use `patch` in place of `paths` for that file.
   - Prefer fewer, coherent commits. If unsure, merge.

5. **Exclude.** A change that should not be committed yet, but should not be ignored either (such as a secret pasted into a tracked file), goes in `excluded` with a reason. Never put a secret's content in the plan.

6. **Build the plan.** Run `<skill-directory>/plan` from the repository root. It writes `PLAN_DIR/plan.json` and records the base sha. Run `new` first, then one `commit` per commit in order, plus `ignore` and `exclude` as needed:

   ```sh
   plan new
   plan ignore .playwright-mcp/ --reason "browser tool output"
   plan commit "feat(cli)!: add --dry-run flag" --why "One line: why these files belong together." internal/cli/run.go internal/cli/run_test.go
   plan exclude .env.local --reason "looks like a secret"
   ```

   - `ignore` is optional and repeatable; it takes `--subject` (the `chore:` header) and `--untrack PATH...`. `render` turns it into the first commit.
   - Pass whole files as PATHs. Use `--patch patch-01.diff` (a file inside `PLAN_DIR`) when a commit takes only some hunks of a file. A commit may have both.
   - A `!` in the header sets `breaking`. Never add footers (no `BREAKING CHANGE:`, `Refs:`, trailers).
   - Leave `--body` out unless the change is very complex and the header cannot carry the why; then keep it short plain text, wrapped at 72, no attribution lines.
   - Editing `plan.json` by hand is still allowed for fixes.

7. **Render.** Run `<skill-directory>/render "$PLAN_DIR/plan.json"` from the repository root. It checks the plan against the working tree, replays every commit in a temporary index, and writes the HTML page. It prints the page path and opens it. If it fails, fix the plan and run it again. Never hand the user a plan that failed.

8. **Report.** Give the HTML path, the number of commits with their headers, the new ignore patterns, any files being untracked, the excluded files, and any notes (for example, that staged changes will be unstaged first). Then stop. The user runs the commands.
