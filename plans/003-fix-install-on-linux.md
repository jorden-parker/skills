# Plan 003: Make `./install` start on Linux (GNU `mktemp` rejects its template)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- install`
> If `install` changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

The README's first sentence is "Claude Code skills, cloned onto a machine and
switched on selectively." The installer is the only entry point, and on any
Linux machine it exits on line 14 before doing anything:

```
mktemp: too few X's in template 'skillstates'
```

macOS `mktemp -t NAME` treats NAME as a prefix and appends random characters.
GNU coreutils `mktemp -t NAME` treats NAME as a full template and requires at
least three trailing `X` characters. Because the script runs under
`set -euo pipefail`, the failed command substitution aborts the script. This
was reproduced on 2026-09-15 in a `debian:stable-slim` container with
`bash /repo/install --list`.

After this plan, `mktemp` is called in the one form both implementations
accept, and the script works on both platforms.

## Current state

- `install` — the whole installer, ~346 lines of bash. Lines 14–16 create the
  two temp files the script uses for state and the trap that removes them:

```sh
# install:14-16
STATES="$(mktemp -t skillstates)"
BEFORE="$(mktemp -t skillstates-before)"
trap 'rm -f "$STATES" "$STATES.new" "$BEFORE"' EXIT
```

- `$STATES.new` is also written at `install:155` (in `set_state`) and is
  covered by the same trap. Do not change that.
- Repo conventions: plain POSIX-leaning bash, must run on macOS's bash 3.2
  (`bash --version` here prints `3.2.57`). No bash 4 features (no associative
  arrays, no `${var,,}`, no `mapfile`).
- The vocabulary in `CONTEXT.md` calls the recorded on/off entries
  **Overrides** and the symlinks **Links**. Use those words in any comment you
  add.

## Commands you will need

| Purpose           | Command                                           | Expected on success                          |
| ----------------- | ------------------------------------------------- | -------------------------------------------- |
| Syntax check      | `bash -n install`                                 | exit 0, no output                            |
| Static analysis   | `shellcheck -s bash install`                      | only SC2221/SC2222 at lines ~245–247 (known) |
| Local smoke (mac) | `CLAUDE_CONFIG_DIR=$(mktemp -d) ./install --list` | exit 0, lists geist, new-project, pr-draft   |
| Linux smoke       | see Step 3                                        | exit 0, same list                            |

`shellcheck` is at `/opt/homebrew/bin/shellcheck` on this machine. If it is
missing, `brew install shellcheck` — that is an install into Homebrew, not
into the repo, and is acceptable.

## Scope

**In scope** (the only files you should modify):

- `install` (lines 14–15 only)

**Out of scope** (do NOT touch, even though they look related):

- Any other line of `install`. Plans 004 and 005 change other parts of this
  file; keep this diff to two lines so they apply cleanly.
- `README.md` — nothing user-facing changes.

## Git workflow

- Branch: `fix/install-linux-mktemp`
- One commit. Conventional Commits are enforced by a `commit-msg` hook
  (`commitlint`); example from `git log`: `fix: make python3 override writer write overrides`.
  Use: `fix: create temp files with a template GNU mktemp accepts`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Replace the two `mktemp` calls

Change `install:14-15` to:

```sh
STATES="$(mktemp "${TMPDIR:-/tmp}/skillstates.XXXXXX")"
BEFORE="$(mktemp "${TMPDIR:-/tmp}/skillstates-before.XXXXXX")"
```

Rationale to keep in mind (not needed as a comment): passing an explicit path
with six `X`s is the one form accepted by both BSD/macOS and GNU `mktemp`.
`TMPDIR` is honoured on both, and `/tmp` is the fallback both use.

**Verify**: `bash -n install` → exit 0, prints nothing.

### Step 2: Confirm macOS behaviour is unchanged

```sh
CLAUDE_CONFIG_DIR="$(mktemp -d)" ./install --list
```

**Verify**: exit 0; output contains the three skill names `geist`,
`new-project`, `pr-draft`, each shown as `on`; and afterwards
`ls "${TMPDIR:-/tmp}" | grep skillstates` prints nothing (the trap removed the
temp files).

### Step 3: Confirm Linux behaviour

Docker is installed on this machine (`docker --version` works). Run:

```sh
docker run --rm -v "$PWD:/repo:ro" -e CLAUDE_CONFIG_DIR=/tmp/cc debian:stable-slim \
  bash -c 'apt-get install -y -qq jq >/dev/null 2>&1; bash /repo/install --list'
```

The repo is mounted read-only; nothing in the working tree is written.

**Verify**: exit 0 and the same three-skill list. The string `too few X's`
must not appear. If Docker is unavailable, record that in your report and
rely on Step 1's reasoning; do not skip Step 2.

## Test plan

There is no test harness in this repo yet (plan 004 adds one). This plan is
verified by the smoke runs above. When plan 004 lands, its test for `--list`
covers this path on whichever OS runs the tests, and plan 008 (CI on Ubuntu)
covers Linux.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n 'mktemp -t' install` returns no matches
- [ ] `grep -c 'XXXXXX' install` prints `2`
- [ ] `bash -n install` exits 0
- [ ] `CLAUDE_CONFIG_DIR="$(mktemp -d)" ./install --list` exits 0 and lists three skills
- [ ] `git diff --stat` shows only `install` changed, 2 insertions and 2 deletions
- [ ] `plans/README.md` status row for 003 updated

## STOP conditions

Stop and report back (do not improvise) if:

- `install:14-15` does not match the excerpt above.
- `./install --list` fails on macOS after the change (this would mean
  `TMPDIR` is unset AND `/tmp` is not writable — report the environment).
- The Docker run fails for a reason other than Docker being unavailable
  (for example a different error from `install`).

## Maintenance notes

- Any future temp file in `install` must use the same explicit-template form.
  A reviewer should reject a new `mktemp -t NAME` without `XXXXXX`.
- `$STATES.new` (written in `set_state`) is derived from `$STATES`, so it
  needs no change and remains covered by the trap.
