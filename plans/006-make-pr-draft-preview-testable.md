# Plan 006: Fix the stale `npm ci` hint in `preview`, add `--no-open`, and smoke-test it

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- skills/pr-draft/ package.json test/ README.md`
> `package.json` and `test/` are expected to have changed in plan 004. If
> anything under `skills/pr-draft/` changed, compare the "Current state"
> excerpts against the live code; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/004-add-install-test-harness-and-lint.md (for the `pnpm test` script)
- **Category**: bug + tests
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

`skills/pr-draft/preview` is the only executable a skill in this repo ships
besides the installer. Two problems:

1. When `node_modules` is missing it tells the user to `run npm ci`. The repo
   switched to pnpm (commit `493dc14`), the README says `pnpm install`, and
   the `new-project` skill says "Never `npm`; a `package-lock.json` in a new
   project is a mistake". Following the hint creates exactly that lockfile.
   An agent running `/pr-draft` will follow the hint verbatim.
2. The script always opens a browser, so it cannot be run in a test or in CI
   without side effects. That is why it has no tests today, even though its
   template lookup and HTML escaping are the parts that go wrong quietly.

After this plan the hint is correct, `--no-open` exists, and `pnpm test`
covers template discovery, the bundled fallback, the multiple-templates
refusal, the refuse-to-overwrite behaviour, and title/body escaping.

## Current state

- `skills/pr-draft/preview` — Python 3 script, ~84 lines, no dependencies
  beyond the standard library plus two files from `node_modules`
  (`marked/lib/marked.umd.js`, `dompurify/dist/purify.min.js`).

```py
# skills/pr-draft/preview:44-46
    parser.add_argument("--title", default="Draft pull request", help="PR title")
    parser.add_argument("--output", required=True, type=Path, help="New HTML file to write")
    args = parser.parse_args()
```

```py
# skills/pr-draft/preview:52-53
        if not (modules / "marked/lib/marked.umd.js").is_file() or not (modules / "dompurify/dist/purify.min.js").is_file():
            parser.exit(1, f"preview: run npm ci in {skill.parent.parent} first\n")
```

```py
# skills/pr-draft/preview:66-80
        with args.output.open("x", encoding="utf-8") as output:
            output.write(document)
    except (OSError, UnicodeError, ValueError) as error:
        parser.exit(1, f"preview: {error}\n")
    path = args.output.resolve()
    print(path)
    try:
        if sys.platform == "darwin":
            opened = subprocess.run(["open", path.as_uri()], check=False).returncode == 0
        else:
            opened = webbrowser.open(path.as_uri(), new=2)
    except (OSError, webbrowser.Error):
        opened = False
    if not opened:
        print(f"preview: browser could not open {path}", file=sys.stderr)
```

- `repository_template(repo)` (`preview:13-34`) searches `<root>`,
  `<root>/docs`, `<root>/.github` for a file whose stem is
  `pull_request_template` (any case) or a directory of that name; raises
  `ValueError` listing candidates when more than one is found; returns `None`
  when none.
- `skills/pr-draft/SKILL.md:81` documents the requirement: "It requires
  Python 3 and a one-time `pnpm install` in this skills repository". The
  README's "PR description preview" section says the same. Both already say
  pnpm; only the code is stale.
- `skills/pr-draft/SKILL.md:73-81` is the "Local preview" section that tells
  the agent how to invoke the command; it must mention the new flag.
- `package.json` after plan 004 has `"test": "bash test/install.test.sh"`.
- `test/install.test.sh` after plan 004 defines `ok`, `nok`, `assert_eq`,
  `assert_contains`, `assert_not_contains` — copy those five helpers into
  the new test file rather than sourcing (each file stays runnable alone),
  except that the two `contains` helpers below read a here-string instead of
  a `printf | grep -q` pipe: the rendered page is ~100 KB, `grep -q` exits at
  the first match, and under `pipefail` the resulting SIGPIPE turns a hit
  into a false "missing" (reproduced in `artifact/test.sh`, 2026-09-17).
- Vocabulary (`CONTEXT.md`): **Draft PR**, **Managed Block** (the
  `<!-- pr-draft:start -->` … `<!-- pr-draft:end -->` region). The preview
  strips HTML comments via DOMPurify, so markers are invisible in the render;
  the test below checks they are still present in the embedded source.
- Verified during planning (2026-09-15): running the script against a temp
  git repo containing `.github/PULL_REQUEST_TEMPLATE.md` renders that
  template; a title of `x</title><script>` is emitted as
  `x&lt;/title&gt;&lt;script&gt;`.

## Commands you will need

| Purpose      | Command                                                                              | Expected on success |
| ------------ | ------------------------------------------------------------------------------------ | ------------------- |
| Deps present | `ls node_modules/marked/lib/marked.umd.js node_modules/dompurify/dist/purify.min.js` | both listed         |
| Compile      | `python3 -m py_compile skills/pr-draft/preview`                                      | exit 0              |
| Tests        | `pnpm test`                                                                          | exit 0              |
| Format       | `pnpm exec prettier --check .`                                                       | passes              |

## Scope

**In scope**:

- `skills/pr-draft/preview`
- `skills/pr-draft/SKILL.md` (Local preview section only)
- `test/preview.test.sh` (create)
- `package.json` (`test` script only)
- `README.md` (one sentence in "PR description preview")

**Out of scope**:

- `skills/pr-draft/assets/preview.html` and `assets/body.md` — no rendering
  changes. Read `skills/geist/SKILL.md` before touching the HTML; this plan
  does not.
- The DOMPurify configuration.
- Any change to what `gh` is told to do.

## Git workflow

- Branch: `fix/pr-draft-preview-hint`
- Conventional Commits (hook-enforced). Suggested:
  - `fix: point the preview's missing-modules hint at pnpm`
  - `feat: add --no-open to the pr-draft preview`
  - `test: smoke-test the pr-draft preview`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Fix the hint

Change `preview:53` so the message reads
`preview: run pnpm install in {skill.parent.parent} first`.

**Verify**: `grep -n 'npm ci' skills/pr-draft/preview` → no output.
`grep -n 'pnpm install' skills/pr-draft/preview` → one line.

### Step 2: Add `--no-open`

After the `--output` argument (`preview:45`), add:

```py
    parser.add_argument("--no-open", action="store_true",
                        help="Write the file and print its path without opening a browser")
```

Then wrap the browser block (`preview:72-80`) so it runs only when
`not args.no_open`. The `print(path)` on line 71 must still run in both
cases — it is the contract callers rely on.

**Verify**: `python3 -m py_compile skills/pr-draft/preview` → exit 0.
Then, in a temp dir:

```sh
T="$(mktemp -d)"; ./skills/pr-draft/preview --repo "$T" --output "$T/o.html" --no-open; echo "rc=$?"
```

→ prints the absolute path of `o.html`, `rc=0`, and no browser window
appears. (Without `--no-open` the bundled template would open in a tab.)

### Step 3: Write `test/preview.test.sh`

```sh
#!/usr/bin/env bash
# Smoke tests for skills/pr-draft/preview. Never opens a browser.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREVIEW="$REPO/skills/pr-draft/preview"
pass=0; fail=0; n=0
ok()   { n=$((n+1)); pass=$((pass+1)); printf 'ok %d - %s\n' "$n" "$1"; }
nok()  { n=$((n+1)); fail=$((fail+1)); printf 'not ok %d - %s\n    %s\n' "$n" "$1" "$2"; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1"; else nok "$1" "expected [$2] got [$3]"; fi; }
assert_contains() { if grep -qF -- "$2" <<<"$3"; then ok "$1"; else nok "$1" "missing [$2]"; fi; }
assert_not_contains() { if grep -qF -- "$2" <<<"$3"; then nok "$1" "unexpected [$2]"; else ok "$1"; fi; }

run_preview() { set +e; out="$("$PREVIEW" "$@" --no-open 2>&1)"; rc=$?; set -e; }
tmp() { mktemp -d "${TMPDIR:-/tmp}/preview-test.XXXXXX"; }

# bundled fallback when the repo has no template
T="$(tmp)"; git -C "$T" init -q
run_preview --repo "$T" --output "$T/o.html"
assert_eq "fallback exits 0" 0 "$rc"
assert_contains "prints output path" "$T/o.html" "$out"
assert_contains "fallback body embedded" "## What &amp; why" "$(cat "$T/o.html")"

# a repository template wins over the bundled one
T="$(tmp)"; git -C "$T" init -q; mkdir "$T/.github"; printf '## Summary\n' > "$T/.github/PULL_REQUEST_TEMPLATE.md"
run_preview --repo "$T" --output "$T/o.html"
assert_eq "repo template exits 0" 0 "$rc"
assert_contains "repo template embedded" "## Summary" "$(cat "$T/o.html")"
assert_not_contains "bundled template not used" "What &amp; why" "$(cat "$T/o.html")"

# several templates: refuse and list them
mkdir "$T/docs"; printf '## Other\n' > "$T/docs/pull_request_template.md"
run_preview --repo "$T" --output "$T/o2.html"
assert_eq "multiple templates exit 1" 1 "$rc"
assert_contains "multiple templates listed" "multiple PR templates found" "$out"
[ -e "$T/o2.html" ] && nok "no file on refusal" "o2.html exists" || ok "no file on refusal"

# explicit body and title are HTML-escaped; Managed Block markers survive in the source
T="$(tmp)"; printf '<!-- pr-draft:start -->\n# hi\n<script>alert(1)</script>\n<!-- pr-draft:end -->\n' > "$T/b.md"
run_preview "$T/b.md" --title 'x</title><script>' --output "$T/o.html"
assert_eq "body exits 0" 0 "$rc"
assert_contains "title escaped" "x&lt;/title&gt;&lt;script&gt;" "$(cat "$T/o.html")"
assert_contains "body escaped" "&lt;script&gt;alert(1)&lt;/script&gt;" "$(cat "$T/o.html")"
assert_contains "markers kept in source" "&lt;!-- pr-draft:start --&gt;" "$(cat "$T/o.html")"
assert_contains "geist tokens inlined" "--ds-gray-1000" "$(cat "$T/o.html")"

# existing output is never overwritten
run_preview "$T/b.md" --output "$T/o.html"
assert_eq "existing output exits 1" 1 "$rc"
assert_contains "existing output message" "File exists" "$out"

# missing repo dir
run_preview --repo "$T/nope" --output "$T/o3.html"
assert_eq "missing repo exits 1" 1 "$rc"
assert_contains "missing repo message" "does not exist" "$out"

printf '1..%d\n' "$n"
if [ "$fail" -eq 0 ]; then printf '# all %d tests passed\n' "$n"; else printf '# %d of %d tests FAILED\n' "$fail" "$n"; exit 1; fi
```

The `"File exists"` string comes from Python's `FileExistsError` message for
`open(..., "x")`; if your Python words it differently, match on the actual
message and note it in your report.

**Verify**: `bash test/preview.test.sh` → exit 0, `# all 18 tests passed`.
No browser window opened during the run.

### Step 4: Run both suites from `pnpm test`

Change the `test` script in `package.json` to:

```json
"test": "bash test/install.test.sh && bash test/preview.test.sh"
```

**Verify**: `pnpm test` → exit 0, two summary lines.

### Step 5: Document the flag

In `skills/pr-draft/SKILL.md`, in the "Local preview" section, after the
sentence "It prints the generated file's absolute path and automatically
opens it in the default browser.", add: `Pass --no-open to skip the browser,
for example when the user only wants the file path.` (Use backticks around
the flag.)

In `README.md`, in the "PR description preview" section, after "and opens it
in your default browser.", add: `--no-open skips the browser.` (backticked).

**Verify**: `pnpm exec prettier --check .` → passes.

## Test plan

`test/preview.test.sh` as written above: fallback template, repo template,
multiple-template refusal, escaping of title and body, marker survival,
Geist inlining, refuse-to-overwrite, missing repo dir. `pnpm test` → both
suites pass.

## Done criteria

- [ ] `grep -c 'npm ci' skills/pr-draft/preview` prints `0`
- [ ] `grep -c 'no-open' skills/pr-draft/preview` prints at least `2`
- [ ] `pnpm test` exits 0 and prints `# all 18 tests passed` for the preview suite
- [ ] `git status` shows changes only in the five in-scope files
- [ ] `pnpm exec prettier --check .` passes
- [ ] `plans/README.md` status row for 006 updated

## STOP conditions

Stop and report back (do not improvise) if:

- `node_modules/marked/lib/marked.umd.js` is missing — run `pnpm install`
  once (that is the documented setup), and if it is still missing, stop.
- Any test in Step 3 fails in a way that points at `preview` rather than at
  the test (for example the repo template is not found). Report the case.
- `package.json`'s `test` script is not `bash test/install.test.sh` when you
  start (plan 004 not done, or done differently).

## Maintenance notes

- `marked` is pinned `^18`; the script reads its UMD build from
  `lib/marked.umd.js`. If a future major drops the UMD file, the
  `run pnpm install` hint will fire misleadingly — the test
  "fallback exits 0" is what will catch it.
- The tests use `git init` in temp dirs so `--show-toplevel` resolution is
  exercised; keep that.
