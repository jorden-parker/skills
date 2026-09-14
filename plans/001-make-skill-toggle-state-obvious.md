# Plan 001: Make it obvious which skills are on, off, and just changed in `./install`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: this repo has no commits yet (`git log` fails
> with "does not have any commits yet"), so there is no SHA to diff against.
> Instead, open `install` and compare every excerpt in "Current state" against
> the live file by line number. Any mismatch is a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: no commits yet (unborn `main`), 2026-09-14
- **Executed**: 2026-09-14, APPROVED after 2 revisions; diff saved as `plans/001-make-skill-toggle-state-obvious.patch`

## Why this matters

`./install` is the only way this repo turns skills on and off. Today the
interactive `fzf` picker shows selection with fzf's default marker: a thin
vertical bar (`┃`) in fzf's default marker colour, which renders red/pink in
the user's terminal. The user reports two problems: the bar is visually
unappealing, and it is not obvious when a skill has actually been toggled.
Three things cause the second problem:

1. The row shows the _saved_ state (`on`/`off`) as a text column, while the
   marker shows the _pending_ selection. After a TAB the two disagree and the
   reader cannot tell which one wins.
2. Nothing is pre-selected unless _every_ skill is already on. In a mixed
   state the picker opens with zero rows selected, so the picture on screen
   does not match reality until the user re-selects everything by hand.
3. After ENTER, the script prints the full list but never says what changed.

This plan replaces the marker with a clear check mark, pre-selects the rows
that are currently `on`, drops the misleading state column, and prints a
"what changed" summary after saving. The numbered-menu fallback gets the same
change summary and colour treatment so both paths behave alike.

## Current state

Files:

- `install` — single Bash script; the whole installer. All changes go here.
- `README.md` — documents the picker keys ("TAB toggles ... ENTER confirms").
  Keys do not change, so README does not change.
- `CONTEXT.md` — vocabulary. Use its terms in comments and output: a skill
  has a **State** (`on`, `off`, `name-only`, `user-invocable-only`); a
  non-`on` State is stored as an **Override**. Avoid the words "enabled",
  "disabled", "active" in any new text.
- `docs/adr/0001-toggle-skills-with-overrides-not-presence.md` — decides that
  toggling writes `skillOverrides`, never deletes symlinks. This plan does not
  touch how states are stored, only how they are shown.

Tooling facts verified during planning:

- `fzf 0.74.3` is installed at `/opt/homebrew/bin/fzf`.
- `--marker` and `--pointer` accept strings of display width **1 or 2**. A
  3-char marker fails with `marker display width should be up to 2`.
- `--bind 'load:pos(N)+select'` selects input row N at startup and works
  with `--multi`, `--delimiter` and `--with-nth`. Verified: with rows a,b,c,d
  and `load:pos(2)+select+pos(4)+select+pos(1)+accept`, fzf outputs `b` and
  `d`. **Caution:** `pos(N)` with N past the last row clamps to the last row
  and selects it, so only emit `pos()` for real row numbers.
- `--color` accepts the names `marker`, `pointer`, `selected-fg`.
- `shellcheck` is installed. `shellcheck -s bash install` currently reports
  3 pre-existing issues (lines 80, 211, 213). They are out of scope here (see
  plan 002 for line 80); do not fix them in this plan, and do not add new ones.

### Excerpt: helper printers, `install:19-21`

```bash
bold() { printf '\033[1m%s\033[0m\n' "$1"; }
dim()  { printf '\033[2m%s\033[0m\n' "$1"; }
die()  { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }
```

### Excerpt: state file helpers, `install:125-140`

`load_states` writes one `name<TAB>state` line per skill to the temp file
`$STATES`. `set_state NAME STATE` rewrites one row in place.

```bash
set_state() { # $1 name, $2 state
  awk -F'\t' -v n="$1" -v s="$2" 'BEGIN{OFS="\t"} $1==n{$2=s} {print}' "$STATES" > "$STATES.new"
  mv "$STATES.new" "$STATES"
}
```

### Excerpt: `show_list`, `install:142-160`

```bash
show_list() {
  printf '\n'
  bold "Skills in $REPO_DIR/skills"
  printf '\n'
  while IFS=$'\t' read -r name state; do
    local mark link
    link=" "
    [ -L "$LINK_DIR/$name" ] && link="*"
    case "$state" in
      on)   mark=$'\033[32mon                 \033[0m' ;;
      off)  mark=$'\033[31moff                \033[0m' ;;
      *)    mark=$'\033[33m'"$(printf '%-19s' "$state")"$'\033[0m' ;;
    esac
    printf '  %s %s %-14s %s\n' "$link" "$mark" "$name" "$(describe "$name")"
  done < "$STATES"
  ...
```

### Excerpt: the fzf picker, `install:163-193`

```bash
pick_with_fzf() {
  local sel all_on binds
  # Pre-select everything only when everything is currently on. In a mixed
  # state, pre-selecting would silently switch previously-off skills back on.
  if awk -F'\t' '$2!="on"{found=1} END{exit !found}' "$STATES"; then
    all_on=0
  else
    all_on=1
  fi
  binds='tab:toggle,shift-tab:toggle,ctrl-a:select-all,ctrl-d:deselect-all'
  [ "$all_on" -eq 1 ] && binds="load:select-all,$binds"

  sel="$(while IFS=$'\t' read -r name state; do
           printf '%s\t%s\t%s\n' "$name" "$state" "$(describe "$name")"
         done < "$STATES" \
         | fzf --multi --with-nth=1,2,3 --delimiter='\t' \
               --header=$'TAB toggles (cursor stays put) - ENTER confirms\nSelected = on, unselected = off. ctrl-a all, ctrl-d none.' \
               --preview="sed -n '1,25p' $SRC_DIR/{1}/SKILL.md" \
               --preview-window=right:60%:wrap \
               --bind="$binds" \
           | cut -f1)" || true
  if [ -z "$sel" ]; then
    printf '\nNothing selected - leaving states unchanged.\n'
    printf 'To turn everything off, use: ./install --set NAME=off\n'
    return 1
  fi
  while IFS=$'\t' read -r name _; do
    if printf '%s\n' "$sel" | grep -qx "$name"; then set_state "$name" "on"; else set_state "$name" "off"; fi
  done < "$STATES"
  return 0
}
```

### Excerpt: the numbered-menu fallback row printer, `install:203-206`

```bash
    while IFS=$'\t' read -r name state; do
      printf '  %d) %-14s \033[2m%s\033[0m\n' "$i" "$name" "$state"
      i=$((i+1))
    done < "$STATES"
```

### Excerpt: interactive tail, `install:297-308`

```bash
if command -v fzf >/dev/null 2>&1; then
  pick_with_fzf || exit 0
else
  dim "(install fzf for a nicer picker)"
  pick_with_menu || { printf 'no changes written\n'; exit 0; }
fi

write_overrides < "$STATES"
show_list
bold "Saved to $SETTINGS"
```

Conventions to match: 2-space indent, `local` for function variables,
`printf` not `echo` for anything with escapes, colours via raw `\033[..m`
escapes (green `32`, red `31`, yellow `33`, dim `2`, bold `1`), comments in
plain sentences. No new dependencies.

## Commands you will need

| Purpose           | Command                                           | Expected on success                                                |
| ----------------- | ------------------------------------------------- | ------------------------------------------------------------------ |
| Syntax check      | `bash -n install`                                 | exit 0, no output                                                  |
| Lint              | `shellcheck -s bash install`                      | only the 3 pre-existing findings (lines 80, 211, 213); no new ones |
| Dry-run list      | `CLAUDE_CONFIG_DIR=$(mktemp -d) ./install --list` | prints every skill as `on`, exit 0                                 |
| Picker smoke test | see Step 6                                        | prints the expected skill names                                    |

Always run the interactive script with `CLAUDE_CONFIG_DIR` pointed at a temp
directory. Never run it against the real `~/.claude` while testing.

## Scope

**In scope** (the only file you should modify):

- `install`

**Out of scope** (do NOT touch, even though they look related):

- `install:44-98` (JSON backend, `read_overrides`, `write_overrides`) — how
  Overrides are stored is decided by ADR 0001 and has its own plan (002).
- `install:211-213` (the shellcheck case-pattern warnings) — cosmetic, not
  this plan.
- `README.md`, `CONTEXT.md`, `docs/` — no key bindings or vocabulary change.
- Anything under `skills/`.

## Git workflow

- The repo has no commits and no branch convention yet. Do not create the
  initial commit: the operator owns that. Leave your changes in the working
  tree and report.
- Do NOT push or open a PR.

## Steps

### Step 1: Add a snapshot of the starting states

Right after `load_states` is called in main (`install:248`, the line
`load_states`), add a second temp file that keeps the states as they were
before any picking, so the script can print what changed later.

Do it in three places:

1. `install:14`: after `STATES="$(mktemp -t skillstates)"` add
   `BEFORE="$(mktemp -t skillstates-before)"`.
2. `install:15`: extend the trap to also remove it:
   `trap 'rm -f "$STATES" "$STATES.new" "$BEFORE"' EXIT`.
3. `install:248`: change the line `load_states` to two lines:
   ```bash
   load_states
   cp "$STATES" "$BEFORE"
   ```

**Verify**: `bash -n install && CLAUDE_CONFIG_DIR=$(mktemp -d) ./install --list` → exit 0, list prints unchanged.

### Step 2: Add a `state_colour` helper and use it in `show_list`

Add this helper directly under `die()` (`install:21`). It prints a state
padded to 19 columns in its colour, and is reused by every listing:

```bash
state_colour() { # $1 state -> coloured, padded to 19 columns
  local c
  case "$1" in
    on)  c='32' ;;   # green
    off) c='31' ;;   # red
    *)   c='33' ;;   # yellow: name-only, user-invocable-only
  esac
  printf '\033[%sm%-19s\033[0m' "$c" "$1"
}
```

Then in `show_list` (`install:150-154`) replace the whole `case "$state" in ... esac` block with:

```bash
    mark="$(state_colour "$state")"
```

Keep the `printf` on line 155 as is.

**Verify**: `bash -n install && CLAUDE_CONFIG_DIR=$(mktemp -d) ./install --list` → same output as before Step 1 (states still green `on`).

### Step 3: Add a `show_changes` function

Add this function directly after `show_list` (after `install:160`). It
compares `$BEFORE` to `$STATES` and prints one line per skill whose state
changed, or a single "no changes" line.

```bash
show_changes() { # diff $BEFORE against $STATES, one line per changed skill
  local changed=0 name before after
  while IFS=$'\t' read -r name after; do
    before="$(awk -F'\t' -v n="$name" '$1==n{print $2}' "$BEFORE")"
    [ "$before" = "$after" ] && continue
    if [ "$changed" -eq 0 ]; then
      printf '\n'
      bold "Changed"
      changed=1
    fi
    printf '  %-14s %s -> %s\n' "$name" "$(state_colour "$before")" "$(state_colour "$after")"
  done < "$STATES"
  [ "$changed" -eq 1 ] || { printf '\n'; dim "No states changed."; }
}
```

Call it in the interactive tail: change `install:304-306` (now shifted a few
lines down by earlier steps) from

```bash
write_overrides < "$STATES"
show_list
bold "Saved to $SETTINGS"
```

to

```bash
write_overrides < "$STATES"
show_list
show_changes
printf '\n'
bold "Saved to $SETTINGS"
```

Also call it in the `set)` branch: after `show_list` in that branch
(`install:275`) add a line `show_changes`.

**Verify**:

```bash
d=$(mktemp -d); CLAUDE_CONFIG_DIR=$d ./install --set geist=off | tail -8
```

→ output ends with a `Changed` heading, one line `geist  on -> off`, then the
"Start a new Claude Code session" note. Run it again with `geist=off`: the
second run prints `No states changed.`

### Step 4: Rewrite the fzf picker

Replace the body of `pick_with_fzf` (`install:163-193`) with the version
below. What changes and why:

- Pre-select every row whose current State is `on` via `load:pos(N)+select`,
  so the screen matches reality on open. This replaces the `all_on` logic.
- Show only name and description (`--with-nth=1,3`). The State column is
  gone from the row: the check mark is now the single source of truth. Rows
  whose state is `name-only` or `user-invocable-only` get a yellow tag
  appended to the description so the user knows confirming will overwrite
  that state (unselected rows become `off`, same as today).
- Marker `✓ ` in green, pointer `▸ ` in cyan. Both are width 2, the fzf
  maximum. No more red bar.
- Header says exactly what the check mark means.

```bash
pick_with_fzf() {
  local sel binds i
  # Pre-select the rows that are currently on, so the picker opens showing
  # the real state. fzf has no per-row select flag; the load event walks the
  # rows by position instead. Positions are 1-based input line numbers.
  binds='load:'
  i=0
  while IFS=$'\t' read -r _ state; do
    i=$((i+1))
    [ "$state" = "on" ] && binds="${binds}pos($i)+select+"
  done < "$STATES"
  binds="${binds}pos(1),tab:toggle,shift-tab:toggle,ctrl-a:select-all,ctrl-d:deselect-all"

  sel="$(while IFS=$'\t' read -r name state; do
           printf '%s\t%s\t%s%s\n' "$name" "$state" "$(state_tag "$state")" "$(describe "$name")"
         done < "$STATES" \
         | fzf --multi --ansi --with-nth=1,3 --delimiter='\t' \
               --marker='✓ ' --pointer='▸ ' \
               --color='marker:green,pointer:cyan' \
               --header=$'✓ = on after ENTER, no mark = off.\nTAB toggles (cursor stays put), ctrl-a all, ctrl-d none, ENTER saves, ESC quits.' \
               --preview="sed -n '1,25p' $SRC_DIR/{1}/SKILL.md" \
               --preview-window=right:60%:wrap \
               --bind="$binds" \
           | cut -f1)" || true
  if [ -z "$sel" ]; then
    printf '\nNothing selected - leaving states unchanged.\n'
    printf 'To turn everything off, use: ./install --set NAME=off\n'
    return 1
  fi
  while IFS=$'\t' read -r name _; do
    if printf '%s\n' "$sel" | grep -qx "$name"; then set_state "$name" "on"; else set_state "$name" "off"; fi
  done < "$STATES"
  return 0
}
```

Also add this helper directly under `state_colour()` (from Step 2). It must
be a separate function: macOS ships bash 3.2, which cannot parse a `case`
statement inside a `$(...)` command substitution at runtime even though
`bash -n` accepts it. A function body is parsed at definition time, so
calling it inside the substitution is safe.

```bash
state_tag() { # $1 state -> yellow "[state] " prefix for states the picker cannot express, else nothing
  case "$1" in
    on|off) ;;
    *) printf '\033[33m[%s]\033[0m ' "$1" ;;
  esac
}
```

Notes for the executor:

- `--ansi` is required so the yellow tag renders instead of showing raw
  escape codes.
- The tag goes _before_ the description. Descriptions are cut to 90 chars
  and fzf truncates the row at the list width, so a trailing tag is never
  visible.
- Run the script with `/bin/bash` when verifying: the shebang resolves to
  it on macOS and it is the strictest parser you will meet.
- Keep the empty-selection guard exactly as it is. It is documented in
  README ("Confirming an empty selection changes nothing").

**Verify**: `bash -n install && shellcheck -s bash install` → syntax OK; shellcheck shows only the 3 pre-existing findings.

### Step 5: Colour the numbered-menu fallback

In `pick_with_menu` replace the row printer (`install:203-206`, the
`printf '  %d) %-14s \033[2m%s\033[0m\n' ...` line) so the state is coloured
with the same helper and a changed row is flagged:

```bash
    while IFS=$'\t' read -r name state; do
      local before flag=" "
      before="$(awk -F'\t' -v n="$name" '$1==n{print $2}' "$BEFORE")"
      [ "$before" = "$state" ] || flag="*"
      printf '  %d) %s %-14s %s\n' "$i" "$flag" "$name" "$(state_colour "$state")"
      i=$((i+1))
    done < "$STATES"
```

And change the line `printf '\n  q) quit without saving\n\n'` (`install:207`)
to:

```bash
    printf '\n  * = changed this session\n  q) quit without saving\n\n'
```

**Verify**: `bash -n install` → exit 0.

### Step 6: Smoke-test the fzf picker under a pseudo-terminal

fzf needs a TTY. Save this script to your scratch directory (not the repo)
as `fzf_smoke.py` and run it from the repo root. It opens the real picker
with a temp config where `geist` is off and `pr-draft` is on, presses ENTER
immediately, and checks that the pre-selection survived.

```python
import os, pty, select, tempfile, json, re, fcntl, termios, struct, time
cfg = tempfile.mkdtemp()
json.dump({"skillOverrides": {"geist": "off"}}, open(os.path.join(cfg, "settings.json"), "w"))
env = dict(os.environ, CLAUDE_CONFIG_DIR=cfg, FZF_DEFAULT_OPTS="", TERM="xterm-256color")
pid, fd = pty.fork()
if pid == 0:
    os.execvpe("/bin/bash", ["/bin/bash", "./install"], env)
# fzf needs a real window size and an answer to its cursor-position query,
# otherwise it never draws and the test hangs.
fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
out = b""; sent = False; drawn = None; t0 = time.time()
while time.time() - t0 < 20:
    r, _, _ = select.select([fd], [], [], 0.5)
    if r:
        try:
            d = os.read(fd, 4096)
        except OSError:
            break
        if not d:
            break
        out += d
        if b"\x1b[6n" in d:
            os.write(fd, b"\x1b[24;1R")
        if drawn is None and b"\x1b[?1049h" in out:
            drawn = time.time()
    if drawn and not sent and time.time() - drawn > 1.5:
        os.write(fd, b"\r")   # ENTER
        sent = True
try:
    os.kill(pid, 9)
except ProcessLookupError:
    pass
os.waitpid(pid, 0)
plain = re.sub(rb"\x1b\[[0-9;?$]*[A-Za-z]", b"", out).decode(errors="replace")
print(plain[-600:])
print("SETTINGS:", open(os.path.join(cfg, "settings.json")).read())
```

Expected:

- The printed tail contains a `Changed` heading only if a state changed. With
  ENTER pressed immediately, nothing should change, so it prints
  `No states changed.`
- `SETTINGS:` still shows `"geist": "off"` and no entry for `pr-draft`.

If instead `geist` disappears from `skillOverrides` (it was turned back on),
the pre-selection selected the wrong row. Check the `binds` construction in
Step 4: positions must be 1-based input line numbers in `$STATES` order.

**Verify**: `python3 /path/to/scratch/fzf_smoke.py` → prints `No states changed.` and the settings JSON still contains `"geist": "off"`.

### Step 7: Manual look

Run `CLAUDE_CONFIG_DIR=$(mktemp -d) ./install` in a real terminal. Confirm:

- Every row opens with a green `✓` (a fresh config means everything is on).
- TAB removes the check on the current row; no red bar appears anywhere.
- ENTER prints the list, then a `Changed` block listing exactly the rows you
  toggled as `on -> off`.

Press `q` or ESC on a second run and confirm `Nothing selected` prints and
no `Changed` block appears.

**Verify**: visual, as listed above. Record what you saw in your report.

## Test plan

There is no test suite in this repo. Verification is:

- `bash -n install` and `shellcheck -s bash install` (no new findings).
- The pseudo-terminal smoke test in Step 6 (pre-selection correct, no
  accidental state flip).
- `./install --set geist=off` twice against a temp config: first run prints
  `geist  on -> off` under `Changed`, second prints `No states changed.`

## Done criteria

ALL must hold:

- [ ] `bash -n install` exits 0
- [ ] `shellcheck -s bash install` reports only lines 80, 211, 213 (pre-existing)
- [ ] `grep -n "load:select-all" install` returns no matches
- [ ] `grep -n "marker='✓ '" install` returns exactly 1 match
- [ ] `grep -n "^show_changes()" install` returns exactly 1 match
- [ ] `grep -n "^state_tag()" install` returns exactly 1 match
- [ ] Step 6 smoke test prints `No states changed.` and leaves `"geist": "off"` in the temp settings
- [ ] `git status --short` shows `install` as the only modified tracked path (untracked files are unchanged from before)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Any excerpt in "Current state" does not match `install` at the given lines.
- `fzf --version` reports a version below 0.29 (`pos()` action missing) or
  fzf rejects `--marker='✓ '` with a width error.
- The Step 6 smoke test turns `geist` back on after two attempts to fix the
  `binds` construction.
- Making the change seems to require editing `write_overrides`,
  `read_overrides`, or any file other than `install`.

## Maintenance notes

- `show_changes` and the menu's `*` flag both depend on `$BEFORE` being a
  copy of `$STATES` taken right after `load_states`. If `load_states` is ever
  called again later in the script, re-copy or the diff is wrong.
- The fzf picker is still binary: unselected rows become `off`, so a
  `name-only` or `user-invocable-only` skill loses that state if left
  unselected. The yellow tag warns about it, and `--set` remains the way to
  set those states. A follow-up could make the picker cycle all four states,
  but that changes the documented key bindings in README and is deferred.
- Reviewer focus: the `binds` string. A wrong `pos()` index silently flips a
  skill's state on ENTER.
