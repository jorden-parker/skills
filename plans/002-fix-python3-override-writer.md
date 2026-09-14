# Plan 002: Make the python3 JSON fallback in `./install` actually write Overrides

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: this repo has no commits yet, so there is no
> SHA to diff against. Open `install` and compare the excerpt in "Current
> state" against lines 79-96 of the live file. Any mismatch is a STOP
> condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (independent of plan 001; both edit `install`, so
  run them one after the other, not in parallel)
- **Category**: bug
- **Planned at**: no commits yet (unborn `main`), 2026-09-14
- **Executed**: 2026-09-14, APPROVED first pass; diff saved as `plans/002-fix-python3-override-writer.patch`

## Why this matters

On a machine without `jq`, `./install` falls back to `python3` to edit
`settings.json`. That fallback never receives the state list: the Python
script is fed on stdin via a heredoc, and the heredoc replaces the pipe that
carries the `name<TAB>state` lines. `sys.stdin` is already at end-of-file when
the loop runs, so zero Overrides are written. The script then prints the list
from its in-memory temp file, which shows `off`, so the user believes the
change saved. `settings.json` stays `{}`.

Verified during planning by running a copy of `install` with the jq branch
disabled: `--set geist=off` printed `geist off` and left `settings.json` as
`{}`. `shellcheck` reports it as `SC2259 (error): This redirection overrides
piped input` at line 80.

macOS ships `jq` in `/usr/bin` since macOS 15, so most users hit the jq path
and never see this. Linux boxes without jq lose every toggle silently.

## Current state

Files:

- `install` — the installer. The bug is in `write_overrides`, lines 67-98.
- `CONTEXT.md` — vocabulary: an **Override** is the recorded entry giving a
  skill a State other than `on`. Absence of an Override means `on`. Use the
  word Override in any new comment.
- `docs/adr/0001-toggle-skills-with-overrides-not-presence.md` — the reason
  Overrides exist. Nothing in this plan changes the storage format.

### Excerpt: `write_overrides`, python3 branch, `install:79-96`

```bash
  else
    printf '%s\n' "$pairs" | python3 - "$SETTINGS" "$SETTINGS.tmp" <<'PY'
import json,sys
settings,out=sys.argv[1],sys.argv[2]
d=json.load(open(settings))
ov=d.get("skillOverrides") or {}
for line in sys.stdin:
    line=line.rstrip("\n")
    if not line: continue
    k,_,v=line.partition("\t")
    ov[k]=v
ov={k:v for k,v in ov.items() if v!="on"}
if ov: d["skillOverrides"]=ov
else: d.pop("skillOverrides",None)
json.dump(d,open(out,"w"),indent=2)
open(out,"a").write("\n")
PY
  fi
  mv "$SETTINGS.tmp" "$SETTINGS"
```

For contrast, the working jq branch (`install:72-78`) passes the pairs
through a real pipe and reads the script from `-` arguments. The Python
branch must get the pairs some way other than stdin. The `read_overrides`
function at `install:55-63` already uses the same heredoc-on-stdin shape but
needs no extra input, so it is fine and must not be touched.

Conventions: 2-space indent, no new dependencies, keep the `PY` heredoc
style (script inline, quoted delimiter so Bash does not expand `$`).

## Commands you will need

| Purpose           | Command                      | Expected on success                                                                                                      |
| ----------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Syntax check      | `bash -n install`            | exit 0                                                                                                                   |
| Lint              | `shellcheck -s bash install` | SC2259 at line 80 gone; only SC2221/SC2222 (lines 211/213) remain, or fewer if plan 001 already landed and shifted lines |
| Forced-python run | see Step 2                   | `settings.json` contains `"geist": "off"`                                                                                |

## Scope

**In scope** (the only file you should modify):

- `install`, lines 79-96 only (the python3 branch of `write_overrides`)

**Out of scope**:

- The jq branch (`install:72-78`) — it works.
- `read_overrides` (`install:50-65`) — its heredoc is correct.
- Everything else in `install`; README; docs.

## Git workflow

- No commits exist yet. Leave changes in the working tree; the operator
  makes the first commit. Do NOT push or open a PR.

## Steps

### Step 1: Pass the pairs as an argument instead of stdin

Replace the python3 branch (`install:79-96`) with:

```bash
  else
    # The script arrives on stdin via the heredoc, so the pairs cannot also
    # come through stdin. Pass them as an argument instead.
    python3 - "$SETTINGS" "$SETTINGS.tmp" "$pairs" <<'PY'
import json,sys
settings,out,pairs=sys.argv[1],sys.argv[2],sys.argv[3]
d=json.load(open(settings))
ov=d.get("skillOverrides") or {}
for line in pairs.split("\n"):
    if not line: continue
    k,_,v=line.partition("\t")
    ov[k]=v
ov={k:v for k,v in ov.items() if v!="on"}
if ov: d["skillOverrides"]=ov
else: d.pop("skillOverrides",None)
json.dump(d,open(out,"w"),indent=2)
open(out,"a").write("\n")
PY
  fi
```

The `printf '%s\n' "$pairs" |` prefix is removed. Everything else in the
function stays.

**Verify**: `bash -n install && shellcheck -s bash install 2>&1 | grep -c SC2259` → `0`

### Step 2: Prove the python3 path writes Overrides

Make a copy of the script with the jq branch disabled and run it against a
temp config. Use your scratch directory, not the repo.

```bash
s=<your scratch dir>
sed 's/^if command -v jq >\/dev\/null 2>&1; then json_tool="jq"/if false; then json_tool="jq"/' install > "$s/install-py"
chmod +x "$s/install-py"
cp -R skills "$s/"          # install-py resolves skills/ next to itself
d=$(mktemp -d)
CLAUDE_CONFIG_DIR=$d bash "$s/install-py" --set geist=off >/dev/null
cat "$d/settings.json"
CLAUDE_CONFIG_DIR=$d bash "$s/install-py" --set geist=on >/dev/null
cat "$d/settings.json"
```

**Verify**: first `cat` prints `{"skillOverrides": {"geist": "off"}}` (pretty-printed); second `cat` prints `{}` (an `on` Override is dropped, matching the jq branch).

### Step 3: Confirm the jq path is untouched

```bash
d=$(mktemp -d); CLAUDE_CONFIG_DIR=$d ./install --set geist=off >/dev/null; cat "$d/settings.json"
```

**Verify**: prints `{"skillOverrides": {"geist": "off"}}` (pretty-printed).

## Test plan

No test suite exists. The forced-python run in Step 2 is the regression
check. Both backends must produce byte-identical `settings.json` for the
same `--set` input except for whitespace.

## Done criteria

- [ ] `bash -n install` exits 0
- [ ] `shellcheck -s bash install` no longer reports SC2259
- [ ] `grep -n 'printf .%s.n. "\$pairs" | python3' install` returns no matches
- [ ] Step 2 forced-python run writes `"geist": "off"`
- [ ] Step 3 jq run writes `"geist": "off"`
- [ ] `git status --short` shows `install` as the only modified path
- [ ] `plans/README.md` status row updated

## STOP conditions

- The excerpt does not match `install:79-96` (if plan 001 landed first, the
  line numbers shift by roughly 10; match on content, and STOP only if the
  content differs).
- Step 2 still writes `{}` after the change.
- The fix seems to need a change to the jq branch or to `read_overrides`.

## Maintenance notes

- Argument length: `$pairs` is one short line per skill. It would need
  thousands of skills to approach the OS argument limit. Not a concern.
- Reviewer focus: the two backends must stay in lockstep. Any future change
  to the jq filter needs the same change in the Python block.
