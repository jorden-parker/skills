---
name: handoff-resume
description: Pick up an earlier session's handoff note for this repo. Called by the handoff skill when it is not saving.
argument-hint: "optional: a word to match in the note, or a request"
---

# Resume from a handoff note

Input: the optional arguments (a short word to match in a note, or a longer request).

1. **List notes.**
   - Run: `<skill-directory>/notes [WORD]`, with WORD only if the argument is one short word.
   - Output: `## notes` (up to 5, newest first, with ages), `## chosen` (a path), then the current branch, recent commits and status.
   - If it prints `no notes`: say so, ask what to work on, and stop.

2. **Read** the file under `## chosen`.

3. **Report** exactly these 5 lines, filled in:
   - `Note: <path> (<age>)`
   - `Goal: <one line from the note>`
   - `First next step: <step 1 of Next steps>`
   - `Changed since: <one line comparing the note's State with the branch, commits and status above>`
   - `Other notes: <the other paths from ## notes, or none>`

4. **Ask** with AskUserQuestion, three choices:
   - Continue this note.
   - Open another listed note.
   - Ignore the notes and do something else.

5. **Act** only on the answer.
   - Continue: follow the note's Next steps.
   - Another note: go back to step 2 with that path.
   - Ignore: drop the note's content and take the user's new request.
   - If the arguments held a longer request, treat it as the pending task and still confirm with step 4 that the note applies before using it.
