# Plan 008: Run format, lint and tests on GitHub Actions for every push (Ubuntu and macOS)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 067367d..HEAD -- package.json test/ .github/`
> `package.json` and `test/` are expected to have changed in plans 004 and 006. `.github/` must not exist yet; if it does, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/004-add-install-test-harness-and-lint.md, plans/006-make-pr-draft-preview-testable.md
- **Category**: dx
- **Planned at**: commit `067367d`, 2026-09-15

## Why this matters

The repo is pushed to GitHub (`origin` is
`https://github.com/jorden-parker/skills.git`) and is meant to be cloned
onto more than one machine, yet every check runs only on the author's Mac.
The Linux-only crash fixed in plan 003 is the concrete cost: it could not
have been seen locally. A two-OS matrix running the same three commands the
pre-commit hook runs (`prettier --check`, `pnpm lint`, `pnpm test`) closes
that gap at no ongoing cost.

## Current state

- No `.github/` directory exists.
- After plans 004 and 006, `package.json` scripts are:

```json
"scripts": {
  "prepare": "husky",
  "test": "bash test/install.test.sh && bash test/preview.test.sh",
  "lint": "shellcheck -s bash install"
},
```

- `packageManager` is NOT set in `package.json`; `pnpm-lock.yaml` is present
  (lockfile version from pnpm 10). The workflow pins pnpm via the action
  input instead.
- `.husky/pre-commit` runs `pnpm exec lint-staged`, `pnpm lint`, `pnpm test`.
  The `prepare` script installs husky hooks on `pnpm install`; in CI that is
  harmless (hooks are never triggered) but `husky` prints a warning when
  `.git` is absent — it is present in Actions checkouts, so nothing to do.
- Tests need: `bash`, `jq`, `python3`, `git`, `readlink`. `pnpm lint` needs
  `shellcheck`. Ubuntu runners ship `jq`, `python3`, `git`; `shellcheck` is
  preinstalled on `ubuntu-latest` images. macOS runners ship `jq`, `git`,
  `python3`; `shellcheck` needs `brew install shellcheck`.
- `test/preview.test.sh` needs `node_modules` (marked, dompurify) — so
  `pnpm install` must run before tests. It never opens a browser (`--no-open`).
- Commit convention: Conventional Commits, enforced locally by
  `commitlint`. The workflow also runs commitlint on the pushed range so a
  `--no-verify` commit is caught.

## Commands you will need

| Purpose          | Command                                                                                    | Expected on success                                                          |
| ---------------- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| Validate YAML    | `python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' .github/workflows/ci.yml` | exit 0 (skip if PyYAML absent; `gh workflow view` after push also validates) |
| Local equivalent | `pnpm exec prettier --check . && pnpm lint && pnpm test`                                   | exit 0                                                                       |
| Repo visibility  | `gh repo view --json visibility -q .visibility`                                            | `PUBLIC` or `PRIVATE` (informational)                                        |

## Scope

**In scope**:

- `.github/workflows/ci.yml` (create)
- `README.md` — one sentence in `## Notes`

**Out of scope**:

- `package.json` — do not add `packageManager`; do not change scripts.
- `.husky/*`.
- Branch protection or repository settings — the operator's call.
- Pushing. Creating the file is the deliverable; the operator pushes.

## Git workflow

- Branch: `ci/github-actions`
- Conventional Commits (hook-enforced). Suggested: `ci: run format, lint and tests on push`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Create the workflow

`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  check:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: pnpm/action-setup@v4
        with:
          version: 10
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
      - name: Install shellcheck (macOS)
        if: runner.os == 'macOS'
        run: brew install shellcheck
      - run: pnpm install --frozen-lockfile
      - run: pnpm exec prettier --check .
      - run: pnpm lint
      - run: pnpm test
      - name: Lint commit messages
        if: github.event_name == 'pull_request'
        run: pnpm exec commitlint --from ${{ github.event.pull_request.base.sha }} --to ${{ github.event.pull_request.head.sha }}
```

Notes:

- `fetch-depth: 0` is needed for the commitlint range.
- The commitlint step runs only on pull requests, where a base SHA exists.
  Pushes rely on the local hook.
- If `pnpm install --frozen-lockfile` fails because the lockfile is out of
  date, that is a real finding — STOP and report, do not drop the flag.

**Verify**: the YAML parses (command above), and
`pnpm exec prettier --check .github/workflows/ci.yml` passes (Prettier
formats YAML; run `pnpm exec prettier --write` on the file if needed and
re-check).

### Step 2: Run the same three commands locally

`pnpm exec prettier --check . && pnpm lint && pnpm test` → exit 0. This is
what the workflow does; if it fails here it will fail there.

### Step 3: README note

Under `## Notes`, add:

```md
- CI (`.github/workflows/ci.yml`) runs the same format, lint and test commands as the pre-commit hook on Ubuntu and macOS for every push and pull request.
```

**Verify**: `pnpm exec prettier --check README.md` → passes.

## Test plan

No new tests. The workflow is validated by parsing locally and, after the
operator pushes, by a green run on both OSes. Record the run URL in
`plans/README.md`'s execution log if you have it.

## Done criteria

- [ ] `.github/workflows/ci.yml` exists and parses as YAML
- [ ] The file contains `ubuntu-latest` and `macos-latest`, and the three commands `prettier --check`, `pnpm lint`, `pnpm test`
- [ ] `pnpm exec prettier --check .` passes (includes the new YAML)
- [ ] `pnpm exec prettier --check . && pnpm lint && pnpm test` exits 0 locally
- [ ] `git status` shows only `.github/workflows/ci.yml` and `README.md`
- [ ] `plans/README.md` status row for 008 updated

## STOP conditions

Stop and report back (do not improvise) if:

- `.github/` already exists.
- `pnpm install --frozen-lockfile` fails locally.
- `pnpm test` is not the two-suite command shown in "Current state" (plans
  004/006 not done as specified).

## Maintenance notes

- When a new system tool becomes a test dependency (see plan 004's
  restricted-PATH list), the macOS install step is where it gets added.
- If the repo goes private and Actions minutes matter, drop `macos-latest`
  first; the Linux run is the one that catches portability bugs.
