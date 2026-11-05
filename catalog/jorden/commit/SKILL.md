---
name: commit
description: Organise the working tree's changes into Conventional Commits and open them as copyable commands on a local HTML page. Makes no Git changes.
argument-hint: "optional: hints on grouping, scope names, or an issue to reference"
disable-model-invocation: true
---

# Commit plan

Produce one thing: a commit plan for the current working tree, shown as an HTML page of copyable commands. The user runs the commands.

This skill reads only. It never stages, commits, amends or pushes, and it never edits `.git/index`. Do not run `git add`, `git commit`, `git reset` or `git stash` here, and do not run `git status` without `--no-optional-locks`.

Output rule: say nothing in chat before or between commands and do not list commits. The final reply is only the page path, plus one line if the build failed and could not be fixed.

## Steps

1. **Gather.** Run `<skill-directory>/plan gather` from the repository root. It prints the commitlint config, recent subjects, `.gitignore`, status, diff and untracked file contents. If it prints `nothing to commit`, stop.

2. **Learn the repo's convention.** From that output, note commitlint `type-enum`, `scope-enum` and `header-max-length`, and the scope habits and casing in recent subjects. With nothing to go on, use the Conventional Commits defaults: `feat fix refactor perf test docs build ci chore style revert`. Keep subjects lowercase and without a final full stop.

3. **Ignore first.** Before grouping, find untracked or changed files that no repository should track: build output, caches, dependency folders, logs, editor and OS files (`.DS_Store`, `.idea/`), local env files, and output from tools run in this session (`.playwright-mcp/`). Read the existing `.gitignore` and do not repeat a pattern it already has. Prefer a folder or glob pattern (`dist/`, `*.log`) over single files. Put the patterns, each with a short reason, in `ignore.patterns`. If a tracked file matches, add it to `ignore.untrack`; the page shows that the next push removes it from the remote. Leave out anything a teammate might rely on, such as a shared editor config that is already committed.

4. **Group.** One logical change per commit:
   - Tests go in the same commit as the code they test, unless the change is test-only.
   - Lockfiles go with the manifest change that caused them.
   - Generated files go with their source.
   - Pure formatting or renames come before behaviour changes, as their own `style:` or `refactor:` commit.
   - Order commits so every prefix of the list builds: dependencies first.
   - Split within a file only when it clearly holds two unrelated changes. Then put the unified diff text of only that commit's hunks, relative to the state after the previous commits, in the commit's `patch`. A commit may have both `paths` and `patch`.
   - Prefer fewer, coherent commits. If unsure, merge.

5. **Exclude.** A change that should not be committed yet, but should not be ignored either (such as a secret pasted into a tracked file), goes in `excluded` with a reason. Never put a secret's content in the plan.

6. **Build.** Run once, from the repository root:

   ```sh
   <skill-directory>/plan build <<'EOF'
   {"commits": [{"header": "feat(cli)!: add --dry-run flag", "why": "One line: why these files belong together.",
                 "paths": ["internal/cli/run.go", "internal/cli/run_test.go"], "patch": "", "body": ""}],
    "ignore": {"patterns": {".playwright-mcp/": "browser tool output"}, "subject": "ignore local files", "untrack": []},
    "excluded": [{"path": ".env.local", "reason": "looks like a secret"}]}
   EOF
   ```

   - `ignore` and `excluded` are optional. `patch` and `body` are optional.
   - A `!` in the header sets `breaking`. Never add footers (no `BREAKING CHANGE:`, `Refs:`, trailers).
   - Leave `body` out unless the change is very complex and the header cannot carry the why; then keep it short plain text, wrapped at 72, no attribution lines.
   - It writes `plan.json` under `/tmp/claude-skills/commit/` (or `$SKILLS_TMPDIR`), checks the plan against the working tree, replays every commit in a temporary index, writes the HTML page and prints its path. If it fails, fix the JSON and rerun. Never hand the user a plan that failed.

7. **Open** the page: Run `open <path>` as its own command (no `&&`, no quotes).

8. **Reply** with only the page path. The user runs the commands.
