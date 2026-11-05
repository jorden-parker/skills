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

Run every command from the repository root. `<skill-directory>` is this skill's folder.

1. **Gather.**
   - Run: `<skill-directory>/plan gather`
   - Output: sections `convention`, `ignore candidates`, `commitlint config`, `recent subjects`, `.gitignore`, `status`, `diff stat`, `diff`, `untracked files`.
   - If the output is `nothing to commit`: stop.

2. **Starter JSON.**
   - Run: `<skill-directory>/plan skeleton`
   - Output: a JSON object. Keep it as your working JSON. It has one commit holding every changed path, `"header": "TODO"`, and `ignore` filled from the candidates.

3. **Ignore.**
   - Input: `## ignore candidates` (`path | pattern | reason`, plus `untrack` for tracked paths).
   - Keep every candidate in `ignore`. Remove one only if it is a shared config a teammate relies on; then also move its path back into the commit `paths`.
   - Add another pattern only if it is clearly local-only in the same categories (build output, caches, dependencies, logs, editor or OS files, local env files, tool output). Prefer folder or glob patterns.

4. **Group.** Split the commit into more commits only with these rules, in this order:
   1. Dependencies first, so every prefix of the list builds.
   2. `style` or `refactor` changes come before behaviour changes.
   3. Tests go in the same commit as the code they test, unless the change is test-only.
   4. A lockfile goes with the manifest that changed it.
   5. Generated files go with their source.
   - Default: one commit. If unsure, keep one commit.
   - `patch` is only for a file that holds two clearly unrelated changes. Put the unified diff of that commit's hunks, relative to the state after the previous commits, in `patch`. Otherwise leave `patch` out.
   - Every changed path appears in exactly one commit's `paths`, in `ignore`, or in `excluded`.

5. **Header.** For each commit set `header` to `type(scope): subject`.
   - Take `type` and `scope` from `## convention`. If `scopes` is `any`, copy the scope habit of `## recent subjects`, or leave the scope out.
   - Lowercase, no full stop, whole header within `max header length`.
   - `!` before the colon marks a breaking change. No footers (no `BREAKING CHANGE:`, `Refs:`, trailers).
   - `why`: one line on why these files belong together.
   - `body`: empty, unless the change is very complex and the header cannot carry the why. Then short plain text wrapped at 72, no attribution lines.

6. **Excluded.** A secret pasted into a tracked file goes in `excluded` as `{"path": "...", "reason": "..."}`, and its path leaves the commit `paths`. Give the path and reason only, never the secret's content.

7. **Build.**
   - Run: `<skill-directory>/plan build <<'EOF'` followed by the JSON and a line `EOF`.
   - Output: the page path on success.
   - On failure, read the error line and apply the fix, then rerun. Retry up to 3 times, then stop and reply with the last error line.

   | Error line contains                                       | Fix                                                                                         |
   | --------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
   | `header must look like`                                   | Rewrite the header as `type(scope): subject`. Replace every `TODO`.                         |
   | `type '...' is not allowed`, `scope '...' is not allowed` | Use a type or scope listed in `## convention`.                                              |
   | `full stop`, `characters, over the limit`                 | Remove the full stop, or shorten the subject.                                               |
   | `needs paths or patch`                                    | Give that commit `paths`, or delete the commit.                                             |
   | `do not reproduce the working tree`                       | Fix the `patch` so it matches the diff, or move its file into `paths` and drop the `patch`. |
   | `HEAD is`                                                 | HEAD moved. Rerun from step 1.                                                              |
   | `is not in any commit, excluded, or ignored`              | Add the path to a commit's `paths`.                                                         |
   | `is listed in more than one commit`                       | Keep the path in one commit only.                                                           |
   | `has no changes`                                          | Remove the path from `paths`.                                                               |
   | `is tracked and matches a new pattern`                    | Add the path to `ignore.untrack`.                                                           |
   | `is in untrack but no tracked file`                       | Remove the path from `ignore.untrack`.                                                      |
   | `is already in .gitignore`                                | Remove that pattern from `ignore.patterns`.                                                 |
   | `patch file not found`, or another patch error            | Move the file into `paths` and drop the `patch`.                                            |
   | `stdin is not valid JSON`                                 | Fix the JSON syntax.                                                                        |

8. **Open.**
   - Run: `open <path>` as its own command (no `&&`, no quotes).
   - Reply with only the page path.
