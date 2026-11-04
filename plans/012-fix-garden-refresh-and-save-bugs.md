# Plan 012: Fix the garden's refresh and save bugs: lost staging, no exit while running, stale preview

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- internal/tui`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding. Plans 010
> and 011 change `internal/tui/review.go` `save()`; that is expected. Any other
> mismatch is a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW (local TUI state only)
- **Depends on**: none. If 010/011 are in flight on another branch, land this after them, because all three touch `review.go`.
- **Category**: bug
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Why this matters

The garden is the bare-`skills` Bubble Tea UI. Four bugs make it misreport
or lose work:

1. **Lost staging.** Stage toggles, press `r` (refresh), apply it: every
   staged change is silently dropped. Quitting asks first; refresh does not.
2. **Trapped.** While a refresh is running, every key is ignored, `ctrl+c`
   included. The hint still says `esc back`. A slow or hung `git` freezes the
   terminal.
3. **Stale preview.** After `space`, `a` or `n`, the right-hand preview still
   shows the old state until the cursor moves.
4. **Out of step after a failed save.** If `links.Apply` errors partway
   through, the model isn't reloaded, so the pending count shows changes that
   were already applied.

There is also a swallowed error. `refreshCmd` discards the error from
`refresh.Run`, so a git failure looks like an empty result.

## Current state

- `internal/tui/refresh.go` — refresh screen.
- `internal/tui/app.go` — model, `reload` (line 121), garden key handling (`updateGarden`, line 240).
- `internal/tui/review.go` — save dialog and `save()` (line 39).
- `internal/tui/app_test.go` — tests: `newFixture` (line 26), `press` (around line 120), `drain`, `selectSkill`, and `TestRefreshScreenPlansThenApplies` (line 340) as the pattern.

`internal/tui/refresh.go:37-54`:

```go
func (m *model) refreshCmd(names []string, check bool) tea.Cmd {
	m.rf.running, m.rf.names, m.rf.check = true, names, check
	repo, paths := m.repo, m.paths
	return func() tea.Msg {
		results, _ := refresh.Run(repo, paths, names, check, nil)
		return refreshMsg{names: names, check: check, results: results}
	}
}

func (m *model) refreshDone(msg refreshMsg) tea.Cmd {
	m.rf.running = false
	m.rf.results = msg.results
	m.rf.check = msg.check
	if !msg.check {
		m.reload()
		return m.resetList()
	}
	return nil
}
```

Note: the second return value of `refresh.Run` is `failed bool`, not an
error. "Swallowed error" means individual `Result.Err` values are already
displayed per row, so **only** the `failed` flag is dropped. Keep it in the
message as `failed` and use it for the summary line (step 4).

`internal/tui/refresh.go:77-84`:

```go
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || m.rf.running {
		return m, nil
	}
	switch k.String() {
	case "esc", "q", "ctrl+c":
		m.mode = modeRefresh
		m.mode = modeGarden
```

`internal/tui/app.go:121-126`:

```go
func (m *model) reload() {
	entries, err := skills.LoadStates(m.repo, m.paths)
	m.loadErr = err
	m.before = entries
	m.staged = append([]skills.Entry(nil), entries...)
}
```

`internal/tui/app.go:264-282`: the `toggle`, `allOn` and `allOff` cases
`return m, nil` without calling `m.refreshPreview()`. The tune dialog
(`tune.go:51`) and discard (`review.go:30`) do call it; match them.

`internal/tui/review.go:41-46`:

```go
	after, err := links.Apply(m.repo, m.paths, m.staged, func(s string) { warnings = append(warnings, s) })
	if err != nil {
		m.mode = modeGarden
		m.setStatus("Save failed: "+err.Error(), true)
		return nil
	}
```

Conventions: no comments except doc comments on exported identifiers (the
user's hooks enforce this). Status messages are friendly full sentences;
see the `setStatus` calls in `app.go:274-280`. Tests drive the model with
`press(m, "key")` and read `ansi.Strip(m.View().Content)`.

## Commands you will need

| Purpose      | Command                      | Expected          |
| ------------ | ---------------------------- | ----------------- |
| TUI tests    | `go test ./internal/tui/ -v` | all `PASS`        |
| All tests    | `go test ./...`              | all `ok`          |
| Vet / format | `go vet ./... && gofmt -l .` | exit 0, no output |

## Scope

**In scope**: `internal/tui/refresh.go`, `internal/tui/app.go`, `internal/tui/review.go`, `internal/tui/app_test.go`.

**Out of scope**: `internal/refresh/` (no change to `Run`'s signature or behaviour); `internal/links/`; key bindings and help text other than the refresh-running hint; the scroll-offset clamping and empty-list message nits (deferred, see Maintenance).

## Git workflow

Branch `advisor/012-garden-refresh-save`; Conventional Commits, e.g. `fix: keep staged changes across a garden refresh`. End each message with the operator's attribution trailer. No push or PR unless told.

## Steps

### Step 1: Write failing tests

In `internal/tui/app_test.go`, modelled on `TestRefreshScreenPlansThenApplies`:

1. `TestRefreshKeepsStagedChanges`: use `newFixture`, then `f.set({"alpha": On})`, then `f.open()`. `selectSkill(t, m, "beta")`, then `press(m, "space")` so beta is staged on. Then `press(m, "r")` and drain. The repo has no git history and no copies, so the plan is empty. Press `esc`. Assert beta is still staged on (`m.staged`) and `m.pending() == 1`. Then do the same with a real apply: set up a recognised copy as `TestRefreshScreenPlansThenApplies` does, stage beta, `r`, `enter`, `esc`, and assert beta is still staged.
2. `TestCtrlCQuitsDuringRefresh`: open, call `m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})` **without** draining so `m.rf.running` is true, then `_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})`. Assert `quits(cmd)`. Also assert that `esc` while running returns to `modeGarden`, and that a later `refreshMsg` delivered with `m.Update(...)` does not switch the mode back to refresh.
3. `TestToggleUpdatesPreview`: open, `selectSkill(t, m, "alpha")`, `press(m, "space")`. Assert the view's preview header shows the new state chip (`on`) and the pending marker, the same way the existing tune test checks them. Look at how `TestStageThenSavePreservesPartialStates` reads the view, and copy its assertion style.

**Verify**: `go test ./internal/tui/ -run 'TestRefreshKeeps|TestCtrlC|TestToggleUpdates' -v` → all three FAIL. Write down how each fails.

### Step 2: Keep staged changes across a refresh

In `refreshDone`, when `!msg.check`, replace `m.reload()` with a merge:

- Remember the staged states from before the refresh in a map keyed by name. Keep only entries where the staged state differs from `m.before`.
- Call `m.reload()`.
- For each remembered name still present in `m.staged`, restore its staged state.
- If anything was restored, set the status to `Refresh done. Your N staged change(s) are still waiting for s.`

**Verify**: `go test ./internal/tui/ -run TestRefreshKeeps -v` → PASS.

### Step 3: Let the user leave a running refresh

- In `updateRefresh`, handle keys **before** the `m.rf.running` guard: `ctrl+c` returns `tea.Quit` when `m.pending() == 0`, and otherwise goes to `modeConfirmQuit`, as the garden does in `app.go:255-258`. `esc` and `q` set `m.mode = modeGarden` and `m.rf.running = false`.
- Add an `int` generation counter to the refresh state struct (`m.rf`). Increment it in `refreshCmd`, carry it in `refreshMsg`, and ignore a `refreshMsg` whose generation doesn't match the current one, or that arrives while `m.mode != modeRefresh`.
- Remove the dead `m.mode = modeRefresh` line.
- `refreshHints()` already shows `esc back` while running; leave it.

The `git` child process is not killed; it finishes in the background and its result is ignored. That is acceptable (see Maintenance).

**Verify**: `go test ./internal/tui/ -run TestCtrlC -v` → PASS.

### Step 4: Carry `failed` through

Add `failed bool` to `refreshMsg` and fill it from `refresh.Run`'s second return. In `refreshDone`, when `failed` is true and `msg.check` is false, set the status with the error flag: `Refresh finished with problems; see the list.`

**Verify**: `go build ./...` → exit 0.

### Step 5: Refresh the preview after toggles

Call `m.refreshPreview()` before `return` in the `toggle`, `allOn` and `allOff` cases of `updateGarden`.

**Verify**: `go test ./internal/tui/ -run TestToggleUpdates -v` → PASS.

### Step 6: Reload after a failed save

In `review.go` `save()`, on a `links.Apply` error, call `m.reload()` before setting the status, and return `m.resetList()`. This matches the settings-failure branch below it. Add `TestSaveFailureReloads`. Make the agents link dir unwritable so `links.Apply` fails: create `f.paths.LinkDir` as a **file**, not a directory, before saving. Stage alpha on, save, and assert `m.pending()` equals what `skills.LoadStates` now reports as different from the staged state. In practice that is `0`, because nothing was applied and staging is dropped. Assert the status contains `Save failed`.

**Verify**: `go test ./...` → all `ok`.

## Test plan

Four new tests in `internal/tui/app_test.go`: `TestRefreshKeepsStagedChanges`, `TestCtrlCQuitsDuringRefresh`, `TestToggleUpdatesPreview`, `TestSaveFailureReloads`. All existing TUI tests must still pass unchanged.

## Done criteria

- [ ] `go test ./...` all `ok`, including the 4 new tests
- [ ] `grep -n 'm.mode = modeRefresh' internal/tui/refresh.go` → only the one in `startRefresh`
- [ ] `grep -n 'results, _ := refresh.Run' internal/tui/refresh.go` → no match
- [ ] `go vet ./...` exit 0; `gofmt -l .` empty
- [ ] Only in-scope files modified (`git status --short`)
- [ ] `plans/README.md` row 012 updated

## STOP conditions

- `tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}.String()` is not `"ctrl+c"` in this Bubble Tea version. Check with a one-off test assertion; if it differs, report the actual string rather than guessing.
- Restoring staged states needs changes outside `internal/tui`.
- An existing TUI test needs its assertions changed. Report which one and why.

## Maintenance notes

- If refresh ever gets slower, the next step is a `context.Context` passed into `refresh.Run` and `exec.CommandContext`, so that `esc` actually kills `git`. That would change `internal/refresh`'s API and is deferred.
- Deferred nits, each S: clamp `helpOffset`/`rf.offset` in the update handlers (`app.go:218`, `refresh.go:95`); show a load error or "no skills" instead of the filter message when the list is empty (`app.go:376`); `tildify` should match `home + "/"`, not a bare prefix (`app.go:572`); and save warnings are cut to one line (`review.go:60`).
- The test helper `press` sends `"ctrl+d"` as `Code: 'c'`. That passes only because `key.Matches` compares strings. Use `Mod: tea.ModCtrl` in new tests.
