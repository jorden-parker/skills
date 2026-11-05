---
name: handoff
description: Save where this session stopped, or offer to pick up an earlier session's handoff note from `${SKILLS_TMPDIR:-/tmp/claude-skills}/handoff/`. Use on /handoff, /handoff save, 'hand off', 'write a handoff', 'pick up where we left off', 'continue from last session', or when the user asks what they were working on.
argument-hint: "save [focus] | [topic or request]"
---

# Handoff router

This skill only chooses between two skills. Do nothing else.

1. Read the arguments.
2. Pick the skill:
   - The first argument word is `save`: call the Skill tool with `handoff-save`. Pass the rest of the arguments as FOCUS.
   - There are no arguments and the user is wrapping up ("hand off", "wrap up", "write a handoff"): call the Skill tool with `handoff-save`, with no FOCUS.
   - Anything else (no arguments to resume, a topic word, a request): call the Skill tool with `handoff-resume`. Pass all the arguments.
3. Follow the skill you called.
