---
name: handoff
description: Save where this session stopped, or offer to pick up an earlier session's handoff note from `${SKILLS_TMPDIR:-/tmp/claude-skills}/handoff/`. Use on /handoff, /handoff save, 'hand off', 'write a handoff', 'pick up where we left off', 'continue from last session', or when the user asks what they were working on.
argument-hint: "save [focus] | [topic or request]"
---

Save a handoff note so a fresh agent can continue the work, or resume from an earlier note.

## Where it goes

Directory: `${SKILLS_TMPDIR:-/tmp/claude-skills}/handoff/`. Create it with `mkdir -p` and `chmod 700`. Never write into the workspace.

Filename: `<repo-name>-<YYYYMMDD-HHMMSS>.md`. Take the repo name from the git top-level directory, or the cwd basename when not in a git repo.

## Mode

- `save` or `save FOCUS`, or the user is wrapping up: **Save**.
- No argument or a `TOPIC`: **Resume**.

## Save

### Gather facts first

Run these and use their output instead of recalling from memory: `pwd`, `git branch --show-current`, `git status -s`, `git log --oneline -5`, and the unpushed commits (`git log --oneline @{u}..`). Skip the git commands outside a git repo.

### Required sections, in this order

1. **Goal**: the user's ask and what "done" means, quoted where possible.
2. **State**: repo, branch, cwd, and uncommitted or unpushed work.
3. **Progress**: what is done and verified, and what is done but not verified.
4. **Relevant files**: paths with line numbers and one line on why each matters.
5. **Decisions**: each decision with its reason, including approaches that were rejected.
6. **Open questions / blockers**.
7. **Next steps**: a numbered list of concrete actions, each with its command or file.
8. **Verification**: commands that prove the work is done (tests, lint).
9. **Suggested skills**: which skills the next agent should call the Skill tool for.

### Rules

- Do not duplicate content already captured in other artifacts (specs, plans, ADRs, issues, commits, diffs). Reference them by path or URL instead.
- Redact sensitive information, such as API keys, passwords, or personally identifiable information.
- If the user passed a `FOCUS`, treat it as a description of what the next session will focus on and tailor the doc accordingly.

### Finish

Print the absolute path of the file and a one-line resume prompt, for example: `Read /tmp/claude-skills/handoff/<file>.md and continue from Next steps.`

## Resume

1. List the notes: `ls -t "$dir"/<repo-name>-*.md 2>/dev/null | head -5`. Take the newest. A short-word argument picks the newest note whose content matches it. If no note exists, say so and ask what to work on.
2. Read the note. Compare it with the repo: `git branch --show-current`, `git log --oneline -10`, `git status --short | head -40`. Skip these outside a git repo.
3. Report in a few lines: the note's path and age, its goal, its first next step, and what changed since it was written. List the other recent notes by name.
4. Ask, don't start. Offer three choices: continue this note, open another listed note, or ignore the notes and do something else. Act only after the user answers. If they decline, drop the note's content and take their new request. If the invocation already carried a longer request, treat it as the pending task and still confirm the note applies before using it.
