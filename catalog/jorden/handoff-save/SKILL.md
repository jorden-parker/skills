---
name: handoff-save
description: Write a handoff note so a fresh agent can continue this session's work. Called by the handoff skill on `save`.
argument-hint: "optional: what the next session will focus on"
---

# Save a handoff note

Input: the optional FOCUS (what the next session will focus on). Notes go under `${SKILLS_TMPDIR:-/tmp/claude-skills}/handoff/`, never into the workspace.

1. **Facts.**
   - Run: `<skill-directory>/note facts`
   - Output: `## note path`, then pwd, branch, status, recent commits and unpushed commits.
   - Use this output, not memory.

2. **Write the note.**
   - Write it to the exact path under `## note path`.
   - Use exactly these 9 headings, in this order:

   ```markdown
   ## Goal

   ## State

   ## Progress

   ## Relevant files

   ## Decisions

   ## Open questions / blockers

   ## Next steps

   ## Verification

   ## Suggested skills
   ```

   - Goal: the user's ask and what "done" means, quoted where possible.
   - State: repo, branch, cwd, and uncommitted or unpushed work, from the facts.
   - Progress: what is done and verified, and what is done but not verified.
   - Relevant files: `path:line` with one line on why each matters.
   - Decisions: each decision with its reason, including approaches that were rejected.
   - Open questions / blockers: one bullet each. Write `none` if there are none.
   - Next steps: a numbered list of concrete actions, each with its command or file.
   - Verification: commands that prove the work is done (tests, lint).
   - Suggested skills: skills the next agent should call the Skill tool for. Write `none` if there are none.
   - Do not copy content that already lives in specs, plans, ADRs, issues, commits or diffs. Reference them by path or URL.
   - Never write API keys, passwords, tokens or personal data.
   - If FOCUS was given, tailor the note to it.

3. **Check.**
   - Run: `<skill-directory>/note check <path>`
   - If it prints problems: fix the note and run it again. Repeat until it passes.

4. **Finish.** Print the absolute path, then this line with the path filled in:
   `Read <path> and continue from Next steps.`
