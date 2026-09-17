# Plan 004: Give `./install` a test suite and a lint gate, run on every commit

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- install package.json .husky/pre-commit README.md test/`
> Plan 003 is expected to have changed `install:14-15` (the `mktemp` lines).
> Any other change to `install` since `067367d`: compare the "Current state"
> excerpts against the live code; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: plans/003-fix-install-on-linux.md (so the suite starts green on Linux too)
- **Category**: tests
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

`install` is ~346 lines of bash with two JSON backends (`jq` and a `python3`
fallback), two interactive pickers, and a mode switch, and it edits the
user's `~/.claude/settings.json`. It has zero automated tests. The last two
plans run against it (001, 002) hit a bash 3.2 incompatibility at execution
time, and a Linux-only crash (plan 003) went unnoticed. The only gate on
commit today is Prettier, which skips `install` because it has no extension.

The script already reads `CLAUDE_CONFIG_DIR` (`install:11`) so it can be
pointed at a throwaway directory. That makes every non-interactive mode
(`--list`, `--set`, `--sync`, `--unlink`, and the "not a terminal" refusal)
testable in under a second with no mocks. After this plan:

- `pnpm test` runs a bash test file that exercises those modes with both JSON
  backends and fails loudly on regressions.
- `pnpm lint` runs `shellcheck` on `install`.
- The pre-commit hook runs both, matching the convention the repo's own
  `new-project` skill prescribes for new projects (`skills/new-project/SKILL.md:104-112`).

Plans 005, 006 and 008 build on this harness.

## Current state

- `install` — the installer. Relevant excerpts:

```sh
# install:11-13  (config dir is overridable — this is what makes testing possible)
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
LINK_DIR="$CLAUDE_DIR/skills"
SETTINGS="$CLAUDE_DIR/settings.json"
```

```sh
# install:60-64  (backend selection: jq if present, else python3)
json_tool=""
if command -v jq >/dev/null 2>&1; then json_tool="jq"
elif command -v python3 >/dev/null 2>&1; then json_tool="python3"
else die "need either jq or python3 to edit $SETTINGS"; fi
```

```sh
# install:244-248  (shellcheck flags the "" pattern on 247 as dead: SC2221/SC2222)
    case "$choice" in
      "") return 0 ;;
      q|Q) return 1 ;;
      *[!0-9]*|"") printf 'not a number\n'; continue ;;
    esac
```

```sh
# install:301-313  (--set mode: validates, links, writes Overrides)
  set)
    for pair in "${SETS[@]}"; do
      name="${pair%%=*}"; state="${pair#*=}"
      grep -qx "$name" <<< "$(list_skills)" || die "no such skill: $name"
      case " $VALID_STATES " in *" $state "*) ;; *) die "bad state: $state (valid: $VALID_STATES)" ;; esac
      link_skill "$name" || true
      set_state "$name" "$state"
    done
    write_overrides < "$STATES"
```

```sh
# install:326-327  (interactive mode refuses without a tty)
[ -t 0 ] || die "not a terminal - use --sync or --set instead"
```

- `write_overrides` (`install:83-115`) drops `on` entries and deletes the
  `skillOverrides` key entirely when it becomes empty. Both backends do this.
  It also copies the previous file to `settings.json.bak` first.
- `package.json` today has a single script:

```json
"scripts": {
  "prepare": "husky"
},
```

- `.husky/pre-commit` today is one line: `pnpm exec lint-staged`
- `.husky/commit-msg` is `pnpm exec commitlint --edit "$1"` — leave it alone.
- Skills in the repo (used as fixtures by the tests; they are real
  directories under `skills/`): `geist`, `new-project`, `pr-draft`. If a
  fourth appears, the tests below must not hard-code "3"; they check for the
  presence of these three names.
- Vocabulary from `CONTEXT.md`: **Link** = the symlink in `~/.claude/skills`;
  **State** = `on`/`off`/`name-only`/`user-invocable-only`; **Override** = the
  `skillOverrides` entry recording a State other than `on`. Use these words in
  test descriptions.
- A user-level preference recorded for this repo: anything JavaScript-flavoured
  must be TypeScript. This plan adds no JavaScript; the tests are bash because
  the thing under test is bash.

## Commands you will need

| Purpose       | Command                            | Expected on success                          |
| ------------- | ---------------------------------- | -------------------------------------------- |
| Install deps  | `pnpm install`                     | exit 0 (already done on this machine)        |
| Run tests     | `pnpm test`                        | exit 0; last line `# all N tests passed`     |
| Lint          | `pnpm lint`                        | exit 0, no output                            |
| Format check  | `pnpm exec prettier --check .`     | `All matched files use Prettier code style!` |
| Tools present | `command -v jq python3 shellcheck` | three paths printed                          |

`jq` is `/usr/bin/jq`, `python3` is 3.14, `shellcheck` is from Homebrew.

## Scope

**In scope** (the only files you should modify):

- `test/install.test.sh` (create)
- `package.json` (add `test` and `lint` scripts only)
- `.husky/pre-commit` (append two lines)
- `install` — ONLY lines 244–248, to remove the dead `""` alternative so
  `shellcheck` passes clean (see Step 4)
- `README.md` — add a short "Development" note (Step 6)

**Out of scope** (do NOT touch):

- Any other part of `install`. Plan 005 changes `link_skill` and the
  `--unlink` mode; do not pre-empt it.
- `.lintstagedrc`, `.prettierrc`, `.commitlintrc.json`.
- `skills/**` — the tests use the real skills as fixtures and must not modify
  them.
- Anything under `$HOME/.claude`. Every test run MUST set `CLAUDE_CONFIG_DIR`
  to a fresh temp dir. A test that touches the real settings file is a bug in
  the test.

## Git workflow

- Branch: `feat/install-tests`
- Commits, Conventional Commits enforced by hook. Suggested:
  - `test: add install test harness`
  - `chore: add test and lint scripts and run them pre-commit`
  - `fix: remove dead empty pattern in menu picker` (if done as its own commit)
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create `test/install.test.sh`

Create the file with this skeleton. It is a self-contained TAP-style runner:
no `bats`, no npm packages.

```sh
#!/usr/bin/env bash
# Tests for ./install. Every case runs against a throwaway CLAUDE_CONFIG_DIR.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL="$REPO/install"
pass=0; fail=0; n=0

ok()   { n=$((n+1)); pass=$((pass+1)); printf 'ok %d - %s\n' "$n" "$1"; }
nok()  { n=$((n+1)); fail=$((fail+1)); printf 'not ok %d - %s\n    %s\n' "$n" "$1" "$2"; }
assert_eq() { # $1 desc, $2 expected, $3 actual
  if [ "$2" = "$3" ]; then ok "$1"; else nok "$1" "expected [$2] got [$3]"; fi
}
assert_contains() { # $1 desc, $2 needle, $3 haystack
  if printf '%s' "$3" | grep -qF -- "$2"; then ok "$1"; else nok "$1" "missing [$2] in: $3"; fi
}
assert_not_contains() { # $1 desc, $2 needle, $3 haystack
  if printf '%s' "$3" | grep -qF -- "$2"; then nok "$1" "unexpected [$2] in: $3"; else ok "$1"; fi
}

fresh() { # new config dir; prints it
  local d; d="$(mktemp -d "${TMPDIR:-/tmp}/install-test.XXXXXX")"
  printf '%s' "$d"
}

overrides() { # $1 config dir -> compact, key-sorted JSON of .skillOverrides or "null"
  jq -cS '.skillOverrides' "$1/settings.json" 2>/dev/null || printf 'null'
}

run_install() { # $1 config dir, rest: args. Captures stdout+stderr, never aborts.
  local d="$1"; shift
  set +e
  out="$(CLAUDE_CONFIG_DIR="$d" "$INSTALL" "$@" 2>&1 </dev/null)"
  rc=$?
  set -e
}

# ---- cases ----

# --list on an empty config dir: every skill is on, nothing is linked yet
d="$(fresh)"; run_install "$d" --list
assert_eq "list exits 0" 0 "$rc"
for s in geist new-project pr-draft; do assert_contains "list shows $s" "$s" "$out"; done
assert_not_contains "list creates no settings file" "settings.json" "$(ls "$d")"

# --set writes an Override and creates the Link
d="$(fresh)"; run_install "$d" --set geist=off
assert_eq "set off exits 0" 0 "$rc"
assert_eq "set off records Override" '{"geist":"off"}' "$(overrides "$d")"
assert_eq "set off creates Link" "$REPO/skills/geist" "$(readlink "$d/skills/geist")"
assert_contains "set off reports change" "Changed" "$out"

# --set on removes the Override and the whole key when it is the last one
run_install "$d" --set geist=on
assert_eq "set on exits 0" 0 "$rc"
assert_eq "set on drops Override and key" "null" "$(overrides "$d")"

# the two non-binary States round-trip
run_install "$d" --set pr-draft=user-invocable-only --set geist=name-only
assert_eq "two sets in one run" '{"geist":"name-only","pr-draft":"user-invocable-only"}' "$(overrides "$d")"

# unrelated settings survive a write
d="$(fresh)"; mkdir -p "$d"; printf '{"theme":"dark","skillOverrides":{"pr-draft":"off"}}\n' > "$d/settings.json"
run_install "$d" --set geist=off
assert_contains "other keys preserved" '"theme": "dark"' "$(cat "$d/settings.json")"
assert_eq "existing Override preserved" '{"geist":"off","pr-draft":"off"}' "$(overrides "$d")"
assert_contains "backup written" "settings.json.bak" "$(ls "$d")"

# validation
d="$(fresh)"; run_install "$d" --set geist=bogus
assert_eq "bad State exits 1" 1 "$rc"
assert_contains "bad State message" "bad state: bogus" "$out"
run_install "$d" --set nope=off
assert_eq "unknown skill exits 1" 1 "$rc"
assert_contains "unknown skill message" "no such skill: nope" "$out"
run_install "$d" --wat
assert_eq "unknown option exits 1" 1 "$rc"

# --sync links everything and keeps States
d="$(fresh)"; run_install "$d" --set geist=off; run_install "$d" --sync
assert_eq "sync exits 0" 0 "$rc"
for s in geist new-project pr-draft; do
  assert_eq "sync links $s" "$REPO/skills/$s" "$(readlink "$d/skills/$s")"
done
assert_eq "sync keeps Override" '{"geist":"off"}' "$(overrides "$d")"

# --unlink removes the Link
run_install "$d" --unlink geist
assert_eq "unlink exits 0" 0 "$rc"
[ -L "$d/skills/geist" ] && nok "unlink removes Link" "symlink still there" || ok "unlink removes Link"
run_install "$d" --unlink geist
assert_eq "unlink twice exits 1" 1 "$rc"

# interactive mode refuses without a terminal (stdin is /dev/null in run_install)
d="$(fresh)"; run_install "$d"
assert_eq "no tty exits 1" 1 "$rc"
assert_contains "no tty message" "not a terminal" "$out"

# python3 fallback: a PATH with no jq on it
bin="$(mktemp -d "${TMPDIR:-/tmp}/install-bin.XXXXXX")"
for t in bash python3 awk sed find sort head cut grep mkdir cp mv ln rm cat mktemp dirname; do
  ln -s "$(command -v "$t")" "$bin/$t"
done
d="$(fresh)"; mkdir -p "$d"; printf '{"theme":"dark"}\n' > "$d/settings.json"
set +e
out="$(PATH="$bin" CLAUDE_CONFIG_DIR="$d" "$INSTALL" --set geist=off 2>&1 </dev/null)"; rc=$?
set -e
assert_eq "python fallback exits 0" 0 "$rc"
assert_eq "python fallback records Override" '{"geist":"off"}' "$(overrides "$d")"
assert_contains "python fallback preserves other keys" '"theme": "dark"' "$(cat "$d/settings.json")"
set +e
out="$(PATH="$bin" CLAUDE_CONFIG_DIR="$d" "$INSTALL" --set geist=on 2>&1 </dev/null)"; rc=$?
set -e
assert_eq "python fallback drops key" "null" "$(overrides "$d")"

# ---- summary ----
printf '1..%d\n' "$n"
if [ "$fail" -eq 0 ]; then printf '# all %d tests passed\n' "$n"; else printf '# %d of %d tests FAILED\n' "$fail" "$n"; exit 1; fi
```

Notes for the executor:

- The python fallback block builds a minimal `PATH` so `command -v jq` fails
  inside `install`. If `install` dies with `command not found` for some tool,
  add that tool's name to the `for t in ...` list — that is the only
  adjustment expected. `printf`, `command`, `[`, `read`, `pwd`, and `cd` are
  bash builtins and need no symlink.
- `overrides()` uses the real `jq` from the test's own PATH; that is fine
  because the test process is not the one being restricted.
- The `.bak` and `.tmp` files are written next to `settings.json`, inside the
  temp dir.

**Verify**: `bash test/install.test.sh` → exit 0, final line
`# all N tests passed` where N is at least 30. Every line starts with `ok`.

### Step 2: Add `test` and `lint` scripts to `package.json`

Change the `scripts` block to:

```json
"scripts": {
  "prepare": "husky",
  "test": "bash test/install.test.sh",
  "lint": "shellcheck -s bash install"
},
```

Keep every other key in the file as it is.

**Verify**: `pnpm test` → exit 0 with the same summary line. `pnpm lint` →
at this point it FAILS with SC2221/SC2222 pointing at the `case` in
`pick_with_menu`; that is expected and Step 4 fixes it.

### Step 3: Run the pre-existing prettier check on the new file

`.lintstagedrc` runs Prettier on every staged file with `--ignore-unknown`,
so `test/install.test.sh` (a `.sh` file) is skipped by Prettier — it has no
shell parser. `package.json` will be reformatted by Prettier on commit; that
is fine.

**Verify**: `pnpm exec prettier --check .` → `All matched files use Prettier code style!`

### Step 4: Remove the dead `""` pattern so `shellcheck` passes

In `install`, in `pick_with_menu`, change

```sh
      *[!0-9]*|"") printf 'not a number\n'; continue ;;
```

to

```sh
      *[!0-9]*) printf 'not a number\n'; continue ;;
```

The `""` alternative can never match because the `""` arm two lines above
already caught it. Behaviour is identical.

**Verify**: `pnpm lint` → exit 0, no output. `pnpm test` → still all pass.

### Step 5: Run tests and lint from the pre-commit hook

Change `.husky/pre-commit` to:

```sh
pnpm exec lint-staged
pnpm lint
pnpm test
```

Order matters and matches `skills/new-project/SKILL.md:104-112`: lint-staged
first because it only touches staged files and is fast.

**Verify**: stage the changes and run the hook directly without committing:
`bash .husky/pre-commit` → exit 0; you see lint-staged's output, then the
test summary line.

### Step 6: Document it in `README.md`

Under the existing `## Notes` section, add one bullet:

```md
- `pnpm test` runs the installer's test suite (`test/install.test.sh`) against a throwaway config directory, and `pnpm lint` runs `shellcheck` on `install`. Both run on every commit. `shellcheck` is a system tool: `brew install shellcheck`.
```

**Verify**: `pnpm exec prettier --check README.md` → passes.

## Test plan

This plan IS the test plan for `install`. Coverage after it lands:

- `--list`, `--set` (one, two, invalid State, unknown skill), `--sync`,
  `--unlink` (present, absent), unknown option, tty refusal.
- Both JSON backends: `jq` (default on this machine) and `python3` (via the
  restricted PATH).
- Preservation of unrelated `settings.json` keys and of existing Overrides.
- Backup file creation.

Not covered (interactive): `pick_with_fzf`, `pick_with_menu`. These need a
pty; leave them for a follow-up and say so in the README bullet if you feel
it matters. Do not attempt to drive `fzf` from the test.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `test/install.test.sh` exists, is executable or run via `bash`, and `pnpm test` exits 0 with `# all N tests passed`, N ≥ 30
- [ ] `pnpm lint` exits 0 with no output
- [ ] `grep -c '\*\[!0-9\]\*|""' install` prints `0`
- [ ] `.husky/pre-commit` contains the three lines from Step 5, in that order
- [ ] `pnpm exec prettier --check .` passes
- [ ] `ls ~/.claude/settings.json*` shows no new `.bak` newer than the start of your run (the tests never touched the real config)
- [ ] `git status` shows changes only in: `test/install.test.sh`, `package.json`, `.husky/pre-commit`, `install`, `README.md`
- [ ] `plans/README.md` status row for 004 updated

## STOP conditions

Stop and report back (do not improvise) if:

- Any test in Step 1 fails for a reason that looks like a real bug in
  `install` rather than a mistake in the test. Report the failing assertion
  and the `install` lines involved; do NOT fix `install` beyond Step 4.
- `shellcheck` reports anything other than SC2221/SC2222 before Step 4.
- The python fallback test needs more than three extra tools added to its
  PATH list — that suggests `install` depends on something unexpected.
- `pnpm test` takes longer than 10 seconds. Something is hanging (most likely
  an interactive prompt: confirm `</dev/null` is on every invocation).

## Maintenance notes

- Every future change to `install` should add or adjust a case here first.
  Plan 005 (Link ownership) and plan 008 (CI) extend this file.
- If a skill is added to `skills/`, the `for s in geist new-project pr-draft`
  loops still pass (they check presence, not count). Add the new name to the
  loops if you want it covered.
- The restricted-PATH trick is the seam for testing the python backend; if
  `install` grows a dependency on another tool, the test's tool list must
  grow with it — a failing test is the signal.
