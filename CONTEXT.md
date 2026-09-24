# Skills

A portable collection of Claude Code skills, cloned onto a machine and activated selectively. The repo is the source of truth; a machine's Claude Code config only records which skills are active there.

## Language

### Distribution

**Skill**:
A directory containing a `SKILL.md`, holding instructions an agent loads on demand. The unit this repo ships and the unit the installer toggles.
_Avoid_: command, plugin, prompt

**Skills Directory**:
The `skills/` folder in this repo. The authoritative copy of every skill.
_Avoid_: library, catalogue

**Link**:
The symlink from a machine's `~/.claude/skills/<name>` back into the Skills Directory. A skill is _installed_ when its Link exists.
_Avoid_: install, copy, deploy

**State**:
How visible an installed skill is on a machine: `on`, `off`, `name-only`, or `user-invocable-only`. A skill can be linked and still be off.
_Avoid_: enabled, disabled, active

**Override**:
The recorded entry that gives a skill a State other than `on`. Absence of an Override means `on`.
_Avoid_: setting, flag, config

### Pull requests

**Draft PR**:
A pull request opened in GitHub's draft state. The `pr-draft` skill produces the description for one as a local HTML preview and Markdown file; opening the PR is the user's step.
_Avoid_: WIP PR, PR draft, proposal

**Managed Block**:
The marker-delimited region of a pull request description that the agent owns and may rewrite. Everything outside it is human-owned and never touched.
_Avoid_: generated section, agent block

**Base**:
The branch a Draft PR merges into, taken from the repository's default branch.
_Avoid_: target, trunk, upstream

### Design

**Design System Present**:
The test that decides whether Geist applies to a repo. True when the repo already declares its own tokens, theme, or component primitives. When true, Geist is not used.
_Avoid_: has styling, themed, branded

**Token**:
A named design value — a colour step, radius, shadow, or font stack — extracted from the live Geist stylesheets rather than transcribed by hand.
_Avoid_: variable, constant, theme value

**Step**:
The position of a colour within its scale, `100` through `1000`. The step number carries a fixed role (background, border, solid fill, text) that holds across every hue.
_Avoid_: shade, weight, level

### Artifacts

**Artifact**:
A standalone HTML page rendered from one Markdown file by `artifact/render` and opened in the browser. The way any skill shows a result too long or too linked for the conversation.
_Avoid_: report, preview, output page

**Marker**:
A bracketed number in an Artifact's prose, `[1]`, that points at the item with the same number under its Sources heading.
_Avoid_: footnote, reference, superscript

**Red**:
The renderer's refusal to produce an Artifact because a Marker and the Sources list do not line up. Nothing is written until the Markdown is fixed.
_Avoid_: failed, invalid, broken

### New projects

**Shape**:
The kind of thing being built — web app, CLI, library, service, or script. Settled before anything else, because it decides the framework, the layout, and the naming vocabulary.
_Avoid_: type, category, kind

**Green**:
The state a new or existing project must reach before it is handed over: every applicable build, lint, formatting, typecheck, test, and runtime check passes. A new project's first commit follows green. A failing check is never green.
_Avoid_: working, done, ready
