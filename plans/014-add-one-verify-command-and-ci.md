# Plan 014: Add one `pnpm test` that checks everything, run it on GitHub Actions, and update DOMPurify

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- package.json pnpm-lock.yaml .husky .github artifact/test.sh`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none. It supersedes plans 004 (lint gate part) and 008, which targeted the deleted Bash installer.
- **Category**: dx
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Why this matters

Only one gate runs today: the husky pre-commit hook. It is installed only
after `pnpm install`, and `--no-verify` skips it. The hook runs Go checks but
never runs the renderer smoke tests (`artifact/test.sh`, 21 cases). `pnpm test`
fails because no `test` script exists. The user's global Stop hook looks for
a repo test command, so it finds nothing useful. Nothing has ever run the Go
port on Linux, though the README's install line invites it. The bundled
DOMPurify (3.4.15) has 5 advisories fixed in 3.4.17 (`pnpm audit`). Its call
sites don't use the affected options, but it sanitises Markdown that comes
from the open web (`web-research`), so it should stay current.

## Current state

`package.json:7-9`:

```json
  "scripts": {
    "prepare": "husky"
  },
```

`.husky/pre-commit`:

```sh
pnpm exec lint-staged
go build ./...
go vet ./...
go test ./...
```

`.lintstagedrc`:

```json
{
  "*": "prettier --ignore-unknown --write",
  "*.go": "gofmt -w"
}
```

Baseline at `b9b4bfe`, measured when this plan was written:

- `bash artifact/test.sh`: `# all 21 tests passed`, exit 0.
- `pnpm exec prettier --check .`: exit 0.
- `shellcheck -S warning install.sh artifact/test.sh`: exit 0. Without `-S warning` there are two intentional SC2016 infos in `install.sh:57,76`, which write literal `$PATH` text into shell rc files.
- `go vet ./...` and `go test ./...`: clean.
- `go.mod:3` is `go 1.27.1`; there is no `toolchain` line.
- There is no `.github/` directory.
- Vendored skills (listed in `upstream-skills.json`) contain shell scripts such as `skills/wizard/template.sh`. Do **not** shellcheck vendored files, since they are upstream's.

The GitHub remote is `jorden-parker/skills` (`README.md:6`).

## Commands you will need

| Purpose         | Command                                                           | Expected                        |
| --------------- | ----------------------------------------------------------------- | ------------------------------- |
| Install JS deps | `pnpm install --frozen-lockfile`                                  | exit 0                          |
| Everything      | `pnpm test`                                                       | exit 0 (after step 1)           |
| Audit           | `pnpm audit --prod`                                               | no critical/high (after step 3) |
| Workflow lint   | `actionlint .github/workflows/verify.yml` if installed, else skip | exit 0                          |

## Scope

**In scope**: `package.json`, `pnpm-lock.yaml` (only via `pnpm update`), `.husky/pre-commit`, `.github/workflows/verify.yml` (create), `README.md` (one line naming `pnpm test`).

**Out of scope**: Go code; `artifact/render`, `skills/pr-draft/preview`; vendored skills; adding golangci-lint (deferred: needs a config decision); Go dependency upgrades (deferred, see Maintenance).

## Git workflow

Branch `advisor/014-verify-and-ci`; Conventional Commits, e.g. `ci: run pnpm test on Ubuntu and macOS`, `fix(deps): update dompurify to 3.4.17`. End each message with your operator's attribution trailer. Don't push or open a PR unless told; if you're told to push, the workflow runs on the PR.

## Steps

### Step 1: Add the `test` script

Add these to `package.json` `scripts`, keeping `prepare`:

```json
    "test": "pnpm run test:go && pnpm run test:render && pnpm run test:lint",
    "test:go": "go vet ./... && go test ./...",
    "test:render": "bash artifact/test.sh",
    "test:lint": "test -z \"$(gofmt -l .)\" && prettier --check . && shellcheck -S warning install.sh artifact/test.sh"
```

Also change `"private": "true"` to `"private": true`, which the 2026-09-15 audit flagged as harmless cruft.

**Verify**: `pnpm test` → exit 0. The output contains `# all 21 tests passed` and `ok` lines for every Go package.

### Step 2: Make the pre-commit hook call it

Replace the last three lines of `.husky/pre-commit` (the `go build`, `go vet` and `go test` lines) with `pnpm run test:go && pnpm run test:render`. Keep `pnpm exec lint-staged` as the first line. `test:lint` is left out on purpose, because lint-staged already formats the staged files.

**Verify**: `sh .husky/pre-commit` → exit 0. It may print lint-staged's "No staged files" message.

### Step 3: Update the renderer libraries

Run `pnpm update dompurify marked`, then `pnpm audit --prod`.

**Verify**: `pnpm audit --prod` reports no `critical` or `high` advisories. `grep -A2 "^  dompurify@" pnpm-lock.yaml` shows 3.4.17 or later. `pnpm run test:render` → `# all 21 tests passed`. If a render test fails after the update, STOP and report: marked or DOMPurify output changed.

### Step 4: Add the workflow

Create `.github/workflows/verify.yml`:

- Triggers: `push` to `main`, and `pull_request`.
- `permissions: contents: read`.
- Job `verify` with `strategy.matrix.os: [ubuntu-latest, macos-latest]`.
- Steps, in this order:
  - `actions/checkout@v5`.
  - `actions/setup-go@v6` with `go-version-file: go.mod`.
  - `pnpm/action-setup@v4` with no version input; add `"packageManager": "pnpm@<the version pnpm --version prints locally>"` to `package.json` so the action can read it.
  - `actions/setup-node@v5` with `node-version: lts/*` and `cache: pnpm`.
  - `pnpm install --frozen-lockfile`.
  - On macOS, install shellcheck with `brew install shellcheck`; Ubuntu runners already have it.
  - Install govulncheck with `go install golang.org/x/vuln/cmd/govulncheck@latest`, then run `govulncheck ./...`.
  - `pnpm test`.
  - `pnpm audit --prod --audit-level high`.
- Set `env: GIT_CONFIG_GLOBAL: /dev/null` and `GIT_CONFIG_NOSYSTEM: "1"` at job level. Some Go tests run `git commit`, and these keep them independent of runner git config.

Check the action major versions are current before committing: look at each action's GitHub releases page, or run `gh api repos/actions/checkout/releases/latest --jq .tag_name` (and the same for the others). If a newer major version exists, use it and say so in your report.

**Verify**: `actionlint .github/workflows/verify.yml` → exit 0 if actionlint is installed. If it isn't, run `python3 -c "import sys,yaml;yaml.safe_load(open(sys.argv[1]))" .github/workflows/verify.yml` → exit 0. If PyYAML is missing too, say so in your report.

### Step 5: Document it

In `README.md`, under the build sentence in the intro paragraph (the one that mentions `go build -o bin/skills .`), add one sentence: `pnpm test` runs every check CI runs.

**Verify**: `pnpm test` → exit 0 (prettier checks the README too).

## Test plan

This plan adds no new tests. It wires up the existing Go tests and the 21 renderer tests, and adds `gofmt`, prettier, shellcheck, govulncheck and audit gates. The first CI run on both operating systems is the real test; if the operator pushes, report both jobs' results.

## Done criteria

- [ ] `pnpm test` exits 0 locally
- [ ] `.husky/pre-commit` calls `pnpm run test:go` and `pnpm run test:render`
- [ ] `pnpm audit --prod --audit-level high` exits 0
- [ ] `.github/workflows/verify.yml` exists with the Ubuntu + macOS matrix
- [ ] `git status --short` lists only in-scope files
- [ ] `plans/README.md` row 014 updated, and rows 004 and 008 marked `REJECTED (superseded by 014)`

## STOP conditions

- `pnpm test` fails in step 1 on a check that passed in the baseline above. Something has drifted; report the failing command.
- `govulncheck ./...` reports a vulnerability that is called from this module's code. Report it with the IDs; don't upgrade Go modules in this plan.
- `pnpm update dompurify marked` changes any package other than those two and their own dependencies in the lockfile.

## Maintenance notes

- Go module updates are deferred: Bubble Tea 2.0.10 → 2.1.x, and `golang.org/x/text` is about 19 minors behind. A Bubble Tea minor can change rendering, and `TestViewFitsTheTerminal` only partly guards that. Do it as its own change with CI green first.
- golangci-lint (staticcheck, errcheck, ineffassign) would have caught the dead assignment at `internal/tui/refresh.go:83` and the ignored `scanOrder` error. Adding it needs a `.golangci.yml`; deferred.
- If more first-party shell scripts appear, add them to `test:lint`'s shellcheck list. Keep vendored scripts out.
