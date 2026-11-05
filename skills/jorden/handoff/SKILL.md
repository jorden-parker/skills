---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up.
argument-hint: "What will the next session be used for?"
disable-model-invocation: true
---

Write a handoff document summarising the current conversation so a fresh agent can continue the work.

## Where it goes

Directory: `${SKILLS_TMPDIR:-/tmp/claude-skills}/handoff/`. Create it with `mkdir -p` and `chmod 700`. Never write into the workspace.

Filename: `<repo-name>-<YYYYMMDD-HHMMSS>.md`. Take the repo name from the git top-level directory, or the cwd basename when not in a git repo.

## Gather facts first

Run these and use their output instead of recalling from memory: `pwd`, `git branch --show-current`, `git status -s`, `git log --oneline -5`, and the unpushed commits (`git log --oneline @{u}..`). Skip the git commands outside a git repo.

## Required sections, in this order

1. **Goal**: the user's ask and what "done" means, quoted where possible.
2. **State**: repo, branch, cwd, and uncommitted or unpushed work.
3. **Progress**: what is done and verified, and what is done but not verified.
4. **Relevant files**: paths with line numbers and one line on why each matters.
5. **Decisions**: each decision with its reason, including approaches that were rejected.
6. **Open questions / blockers**.
7. **Next steps**: a numbered list of concrete actions, each with its command or file.
8. **Verification**: commands that prove the work is done (tests, lint).
9. **Suggested skills**: which skills the next agent should call the Skill tool for.

## Rules

- Do not duplicate content already captured in other artifacts (specs, plans, ADRs, issues, commits, diffs). Reference them by path or URL instead.
- Redact sensitive information, such as API keys, passwords, or personally identifiable information.
- If the user passed arguments, treat them as a description of what the next session will focus on and tailor the doc accordingly.

## Finish

Print the absolute path of the file and a one-line resume prompt, for example: `Read /tmp/claude-skills/handoff/<file>.md and continue from Next steps.`
