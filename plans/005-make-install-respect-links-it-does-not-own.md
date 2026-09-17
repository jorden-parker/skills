# Plan 005: Stop `./install` from overwriting or deleting Links it did not create

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- install test/install.test.sh README.md`
> Plans 003 and 004 are expected to have changed `install` (the `mktemp`
> lines near the top, and one `case` pattern in `pick_with_menu`). If
> `link_skill` (lines ~128–140) or the `unlink` branch (lines ~290–300) differ
> from the excerpts below, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/004-add-install-test-harness-and-lint.md (adds tests to that file)
- **Category**: bug
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

`~/.claude/skills` is shared. On this machine it holds about forty symlinks
that point elsewhere (`../../.agents/skills/<name>`) and a few real
directories; only three entries point into this repo. `install` treats every
symlink there as its own:

1. `link_skill` replaces any existing symlink unconditionally. The day this
   repo gains a skill whose name collides with one of those forty (for
   example `code-review` or `frontend-design`), `./install --sync` silently
   re-points the user's link into this repo. Nothing is printed; the other
   skill just vanishes.
2. `--unlink NAME` removes any symlink of that name, and its error message
   for the non-symlink case says "not a symlink we manage" — but it never
   checks whether it manages the symlink it does find.
3. The one warning `link_skill` does print (a real directory is in the way)
   goes to stdout, and the interactive path redirects that stdout to
   `/dev/null`, so in the most common mode the user never sees it.
4. `--unlink` leaves the skill's Override in `settings.json`, so `--list`
   keeps reporting a State for a skill that is no longer linked.

The ADR at `docs/adr/0001-toggle-skills-with-overrides-not-presence.md`
decided that every skill in the repo is linked on every machine. It says
nothing about taking over links that belong to something else; this plan
keeps that decision intact and adds an ownership check.

## Current state

- `install` — the installer. Excerpts:

```sh
# install:128-140
link_skill() {
  local name="$1" target="$LINK_DIR/$1"
  mkdir -p "$LINK_DIR"
  if [ -L "$target" ]; then
    ln -sfn "$SRC_DIR/$name" "$target"
  elif [ -e "$target" ]; then
    printf '  \033[33mskip\033[0m %s - a real directory is already there, not touching it\n' "$name"
    return 1
  else
    ln -s "$SRC_DIR/$name" "$target"
  fi
  return 0
}
```

```sh
# install:290-300
  unlink)
    [ -n "${UNLINK_NAME:-}" ] || die "--unlink needs a skill name"
    if [ -L "$LINK_DIR/$UNLINK_NAME" ]; then
      rm "$LINK_DIR/$UNLINK_NAME"
      printf 'unlinked %s\n' "$UNLINK_NAME"
      dim "Start a new Claude Code session for this to take effect."
    else
      die "$LINK_DIR/$UNLINK_NAME is not a symlink we manage"
    fi
    exit 0
    ;;
```

```sh
# install:318  (--sync: link_skill's stdout is shown)
    while IFS=$'\t' read -r name _; do link_skill "$name" && printf '  linked %s\n' "$name"; done < "$STATES"
```

```sh
# install:331  (interactive: link_skill's stdout is discarded, so the skip warning is lost)
while IFS=$'\t' read -r name _; do link_skill "$name" >/dev/null && printf '  %s\n' "$name"; done < "$STATES"
```

- Helpers already defined near the top of the file:

```sh
# install:20-22
bold() { printf '\033[1m%s\033[0m\n' "$1"; }
dim()  { printf '\033[2m%s\033[0m\n' "$1"; }
die()  { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }
```

- `write_overrides` (`install:83-115`) takes `name<TAB>state` lines on stdin
  and drops any whose state is `on`. So setting a skill to `on` and calling
  `write_overrides` is how an Override is removed. `set_state NAME STATE`
  (`install:154-157`) updates the in-memory `$STATES` file.
- `readlink` exists on macOS and Linux and prints a symlink's target verbatim
  (no `-f` needed — we compare against the exact path we wrote).
- Vocabulary (`CONTEXT.md`): **Link**, **Override**, **State**. Use these in
  messages, matching existing ones like `"unlinked %s"`.
- Test harness: `test/install.test.sh` from plan 004. Its helpers:
  `fresh` (new config dir), `run_install DIR ARGS...` (sets `$out`, `$rc`),
  `assert_eq`, `assert_contains`, `assert_not_contains`, `overrides DIR`.

## Commands you will need

| Purpose | Command           | Expected on success            |
| ------- | ----------------- | ------------------------------ |
| Tests   | `pnpm test`       | exit 0, `# all N tests passed` |
| Lint    | `pnpm lint`       | exit 0, no output              |
| Syntax  | `bash -n install` | exit 0                         |

## Scope

**In scope** (the only files you should modify):

- `install` — `link_skill` and the `unlink)` branch only
- `test/install.test.sh` — append cases
- `README.md` — one sentence under `## Commands` describing the new refusal

**Out of scope** (do NOT touch):

- `write_overrides`, `read_overrides`, the pickers, `show_list`.
- Any change to which skills get linked (ADR 0001 stands: all of them).
- The user's real `~/.claude/skills`. Tests use `CLAUDE_CONFIG_DIR`.

## Git workflow

- Branch: `fix/install-link-ownership`
- Conventional Commits (hook-enforced). Suggested:
  `fix: refuse to overwrite or remove links the installer did not create`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add an ownership helper and use it in `link_skill`

Directly above `link_skill`, add:

```sh
owns_link() { # $1 name -> 0 when the Link at $LINK_DIR/$1 points into this repo
  [ -L "$LINK_DIR/$1" ] && [ "$(readlink "$LINK_DIR/$1")" = "$SRC_DIR/$1" ]
}
```

Rewrite `link_skill` so that:

- an existing symlink that already points to `$SRC_DIR/$name` is left alone
  (return 0; still print nothing — callers print "linked"/name);
- an existing symlink pointing anywhere else is NOT replaced: print a `skip`
  line to **stderr** naming where it points, return 1;
- a real file/directory is skipped as today, but the message goes to
  **stderr**;
- a missing target gets `ln -s` as today.

Target shape:

```sh
link_skill() {
  local name="$1" target="$LINK_DIR/$1"
  mkdir -p "$LINK_DIR"
  if [ -L "$target" ]; then
    if owns_link "$name"; then return 0; fi
    printf '  \033[33mskip\033[0m %s - a Link is already there pointing at %s, not touching it\n' \
      "$name" "$(readlink "$target")" >&2
    return 1
  elif [ -e "$target" ]; then
    printf '  \033[33mskip\033[0m %s - a real directory is already there, not touching it\n' "$name" >&2
    return 1
  fi
  ln -s "$SRC_DIR/$name" "$target"
}
```

Note the old code used `ln -sfn` to refresh a link that already pointed at
the repo (harmless but now unnecessary because `owns_link` compares the exact
target). If the repo is moved, the old link points at the old path and is
therefore "foreign" — that is acceptable and the skip message tells the user
where it points; `--unlink` then... see Step 2, which must allow removing a
dangling link.

**Verify**: `bash -n install` → exit 0. `pnpm test` → existing cases still
pass (the `sync links` and `set off creates Link` cases exercise this path).

### Step 2: Make `--unlink` check ownership and drop the Override

Replace the `unlink)` branch body with logic that:

1. dies if no name given (unchanged);
2. if `$LINK_DIR/$NAME` is not a symlink → die with the existing message;
3. if it is a symlink but `owns_link` fails → die:
   `"$LINK_DIR/$NAME points at <target>, not into this repo - remove it yourself if that is what you want"`;
4. otherwise `rm` it, then remove the Override: `set_state "$NAME" on` and
   `write_overrides < "$STATES"`, then print `unlinked NAME` and the "new
   session" hint as today.

Exception to (3): a **dangling** link (target no longer exists) whose target
path ends in `/skills/$NAME` may be removed — this is the "repo was moved"
case. Implement as: if `[ ! -e "$LINK_DIR/$NAME" ]` (dangling) and
`case "$(readlink ...)" in */skills/"$NAME") ;; *) die ...;; esac`.

**Verify**: `bash -n install` → exit 0; `pnpm lint` → clean.

### Step 3: Add tests

Append to `test/install.test.sh`, before the `# ---- summary ----` line:

```sh
# ---- Link ownership (plan 005) ----

# a foreign symlink is not replaced and the skip goes to stderr, even in --sync
d="$(fresh)"; mkdir -p "$d/skills"; foreign="$(mktemp -d "${TMPDIR:-/tmp}/foreign.XXXXXX")"
ln -s "$foreign" "$d/skills/geist"
run_install "$d" --sync
assert_eq "sync with foreign Link exits 0" 0 "$rc"
assert_eq "foreign Link untouched" "$foreign" "$(readlink "$d/skills/geist")"
assert_contains "foreign Link reported" "pointing at $foreign" "$out"
assert_eq "other skills still linked" "$REPO/skills/pr-draft" "$(readlink "$d/skills/pr-draft")"

# --unlink refuses a foreign Link
run_install "$d" --unlink geist
assert_eq "unlink foreign exits 1" 1 "$rc"
assert_contains "unlink foreign message" "not into this repo" "$out"
assert_eq "unlink foreign leaves it" "$foreign" "$(readlink "$d/skills/geist")"

# a real directory in the way is reported on stderr (stderr is captured by run_install)
d="$(fresh)"; mkdir -p "$d/skills/geist"
run_install "$d" --sync
assert_contains "real dir skip reported" "real directory is already there" "$out"

# --unlink of an owned Link also removes its Override
d="$(fresh)"; run_install "$d" --set geist=off
run_install "$d" --unlink geist
assert_eq "unlink owned exits 0" 0 "$rc"
assert_eq "unlink drops Override" "null" "$(overrides "$d")"

# a dangling Link into a moved repo can still be unlinked
d="$(fresh)"; mkdir -p "$d/skills"; ln -s "/nonexistent/skills/geist" "$d/skills/geist"
run_install "$d" --unlink geist
assert_eq "unlink dangling exits 0" 0 "$rc"
[ -L "$d/skills/geist" ] && nok "dangling Link removed" "still there" || ok "dangling Link removed"
```

The interactive path's stderr is not exercised (needs a tty); the `--sync`
case proves the message reaches stderr, and the interactive loop at
`install:331` only redirects stdout, so the same message now survives there.

**Verify**: `pnpm test` → all pass, N increased by 13.

### Step 4: README

Under `## Commands`, after the sentence about `--set`, add:

```md
`./install` only ever creates, replaces, or removes a symlink that points into this repo's `skills/`. A symlink to anywhere else, or a real directory, is reported and left alone; remove those by hand.
```

**Verify**: `pnpm exec prettier --check README.md` → passes.

## Test plan

Cases listed in Step 3, in `test/install.test.sh`, following the harness's
existing `assert_*` style. `pnpm test` → all pass.

## Done criteria

- [ ] `grep -c 'ln -sfn' install` prints `0`
- [ ] `grep -c 'owns_link' install` prints at least `3` (definition + 2 uses)
- [ ] `grep -n "skip" install` shows both skip `printf`s end with `>&2`
- [ ] `pnpm test` exits 0; the five new groups above pass
- [ ] `pnpm lint` exits 0
- [ ] `git status` shows changes only in `install`, `test/install.test.sh`, `README.md`
- [ ] `plans/README.md` status row for 005 updated

## STOP conditions

Stop and report back (do not improvise) if:

- `link_skill` or the `unlink)` branch does not match the excerpts.
- `test/install.test.sh` does not exist (plan 004 not done) — do not write
  the tests inline elsewhere.
- Any pre-existing test fails after Step 1. The likely cause is the removed
  `ln -sfn` refresh; report rather than restoring it.
- `readlink` output on this platform includes a trailing newline or is
  otherwise not byte-equal to `$SRC_DIR/$name` for a link the script itself
  created. (Not expected; `$(...)` strips the newline.)

## Maintenance notes

- `owns_link` compares exact strings. If `REPO_DIR` is ever computed
  differently (for example via `realpath`), links created by the old
  computation will read as foreign until re-created. Reviewer: check that
  `REPO_DIR` (`install:9`) is unchanged.
- If a future `--doctor`/`--check` mode is added (see the direction notes in
  `plans/README.md`), `owns_link` is the primitive it should report on.
