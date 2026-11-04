# Plan 011: Make settings.json writes safe: validate first, touch only this repo's keys, keep formatting

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat b9b4bfe..HEAD -- internal/settings internal/cli/set.go internal/cli/sync.go internal/tui/review.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. Plan 010 also edits `set.go`,
> `sync.go` and `review.go`. If 010 has landed, the `links.Apply` call
> signatures will differ from the excerpts below. That is expected and is not
> a STOP; keep 010's signatures.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW (the file format stays the same; the tests pin the output)
- **Depends on**: none (can land before or after 010; see the drift note)
- **Category**: bug
- **Planned at**: commit `b9b4bfe`, 2026-11-04

## Why this matters

`skills` edits the user's `${CLAUDE_CONFIG_DIR:-~/.claude}/settings.json`.
That file holds hooks, permissions, environment variables and sometimes
credentials, and several other tools on this machine also manage it, including
`~/src/claude-custom`. Today the writer:

1. **Erases other tools' overrides.** It deletes every `"on"`/`"off"` entry in
   the whole `skillOverrides` map, not just entries for this repo's skills.
2. **Overwrites the last good backup with a broken file.** `.bak` is
   written _before_ the JSON is parsed, so an invalid settings.json replaces
   the previous good `.bak`.
3. **Changes links before it knows the settings write will work.** With an
   invalid settings.json, `skills set x name-only` links the skill, then fails
   to write the override. The skill ends up fully visible and the command
   exits 1.
4. **Rewrites the file on every command, even when nothing changed.** `.bak`
   then equals the live file, and the real previous version is lost.
5. **Escapes `&`, `<`, `>` as `&` etc.** and turns numbers into float64.
   Hook commands such as `a && b 2>&1` get rewritten into escaped text.
6. **Creates `.bak`, and a new settings.json, as world-readable 0644.**

## Current state

Files:

- `internal/settings/settings.go` — `ReadOverrides` (line 20), `WriteOverrides` (line 44), `scanOrder` (line 96), `encode` (line 130).
- `internal/settings/settings_test.go` — 4 tests; pattern for new ones.
- `internal/atomicfile/write.go` — `atomicfile.Write(path, data)`. It writes through a temp file and rename, keeps the existing file's mode, creates new files 0600, follows a symlinked destination, and refuses a dangling symlink (lines 66-90).
- Callers: `internal/cli/set.go` `save` (line 105) and `applyStates` (line 62), `internal/cli/sync.go` (line 51), `internal/tui/review.go` `save()` (line 39).

`internal/settings/settings.go:44-92` (abridged):

```go
func WriteOverrides(path string, states map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b = []byte("{}\n")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.WriteFile(path+".bak", b, 0o644); err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	...
	ov, _ := doc[key].(map[string]any)
	if ov == nil {
		ov = map[string]any{}
	}
	for k, v := range states {
		ov[k] = v
	}
	for k, v := range ov {
		if s, ok := v.(string); !ok || s == "on" || s == "off" {
			delete(ov, k)
		}
	}
	...
	return atomicfile.Write(path, buf.Bytes())
}
```

`encode` default branch (`settings.go:186-191`) uses `json.MarshalIndent(v, indent, "  ")`
for scalars, and the object branch uses `json.Marshal(k)` for keys. Both
HTML-escape by default.

Callers pass a map holding **every repo skill** (`overrides(after)` in
`internal/cli/set.go:113`; the loop at `internal/tui/review.go:47-50`). So
`states` holds exactly this repo's skill names. That is the set the writer may
change.

Conventions: doc comments on exported identifiers only, **no other
comments** (user hooks reject them). Tests use `t.TempDir()` and
`t.Fatalf("got %q", ...)` (see `settings_test.go:10-27`).

## Commands you will need

| Purpose       | Command                           | Expected   |
| ------------- | --------------------------------- | ---------- |
| Tests         | `go test ./...`                   | all `ok`   |
| Settings only | `go test ./internal/settings/ -v` | all `PASS` |
| Vet           | `go vet ./...`                    | exit 0     |
| Format        | `gofmt -l .`                      | no output  |

## Scope

**In scope**: `internal/settings/settings.go`, `internal/settings/settings_test.go`, `internal/cli/set.go`, `internal/cli/sync.go`, `internal/tui/review.go`, `internal/cli/cli_test.go`, `internal/tui/app_test.go`.

**Out of scope**: `internal/atomicfile/` (reuse it, don't change it); `internal/links/` (plan 010); the JSON key-order algorithm in `scanOrder`/`encode` beyond the escaping change; JSONC/comment support (report an error instead).

## Git workflow

- Branch `advisor/011-safe-settings-writes`; Conventional Commits, e.g. `fix: keep other tools' skill overrides in settings.json`; end each message with your operator's attribution trailer. No push/PR unless told.

## Steps

### Step 1: Write the failing tests first

Add these to `internal/settings/settings_test.go`:

1. `TestWriteKeepsForeignOverrides`: the file holds `{"skillOverrides": {"plugin:x": "off", "other": "on", "alpha": "name-only"}}`. Call `WriteOverrides(path, {"alpha": "on"})`. Expect `skillOverrides` to equal `{"plugin:x": "off", "other": "on"}`: `alpha` is removed and the others are kept verbatim.
2. `TestWriteRejectsNonObjectOverrides`: the file holds `{"skillOverrides": []}`. `WriteOverrides` returns an error containing `skillOverrides`, and the file is unchanged.
3. `TestInvalidJSONKeepsLastBackup`: write `path.bak` = `"good\n"` and `path` = `"{not json"`. `WriteOverrides` returns an error, and `path.bak` is still `"good\n"`.
4. `TestNoChangeWritesNothing`: the file holds the writer's canonical form, `"{\n  \"skillOverrides\": {\n    \"alpha\": \"name-only\"\n  }\n}\n"`. A one-line file would be reformatted, so it would legitimately change. Record its mod time and `.bak` absence. Call `WriteOverrides(path, {"alpha": "name-only", "beta": "off"})`. The file is byte-identical, the mod time is unchanged, and no `.bak` was created.
5. `TestWriteKeepsTextAndNumbersVerbatim`: the file holds `{"hooks": {"cmd": "a && b 2>&1 <x"}, "n": 1.0, "big": 12345678901234567890}`. After `WriteOverrides(path, {"alpha": "name-only"})`, the output contains `"a && b 2>&1 <x"`, `1.0` and `12345678901234567890` literally.
6. `TestBackupAndNewFileArePrivate`: on a missing file, after `WriteOverrides(path, {"alpha": "name-only"})`, both `path` and `path.bak` have mode `0o600`.
7. `TestCheck`: `Check(missing)` is nil, `Check(valid)` is nil, `Check("{not json")` is an error, and `Check('{"skillOverrides": 3}')` is an error.

**Verify**: `go test ./internal/settings/` → it fails to compile because `Check` is missing, or tests 1–7 fail. Note which, then continue.

### Step 2: Decode with numbers kept verbatim, and validate in one place

In `settings.go`, add an unexported `parse(b []byte) (map[string]any, error)`.
It uses `json.NewDecoder(bytes.NewReader(b))` with `UseNumber()`, and returns
an error when the top level isn't an object. It also returns an error naming
`skillOverrides` when that key is present but isn't an object. Add an exported:

```go
// Check reports whether the settings file at path can be updated: a missing
// file is fine; an unreadable file, invalid JSON, a non-object document or a
// non-object skillOverrides is an error naming path.
func Check(path string) error
```

Use `parse` in `WriteOverrides`, `ReadOverrides` and `Check`.
`ReadOverrides` keeps returning an empty map on any error.

### Step 3: Rewrite `WriteOverrides`

Order:

1. Read the file. If it's missing, treat its content as `{}\n`, write nothing yet, and remember that the file was missing.
2. `parse`, returning any error. Nothing has been written at this point.
3. Update only keys in `states`: when the value is `"on"` or `"off"`, `delete(ov, k)`; otherwise `ov[k] = v`. Leave all other keys untouched, whatever their value. Remove the second loop.
4. Encode as today (`encode` plus the trailing newline).
5. If the file existed and the new bytes equal the old bytes, return nil.
6. Write the backup with `atomicfile.Write(path+".bak", old)` (0600 for a new file, the existing mode otherwise), then write the file with `atomicfile.Write(path, new)`. Keep `os.MkdirAll(filepath.Dir(path), 0o755)` at the top.

Keep the doc comment accurate. It must now say that only the given keys
change, that nothing is written when the result is unchanged, and that `.bak`
holds the previous file.

### Step 4: Stop HTML escaping

Add an unexported `marshal(v any) ([]byte, error)`. It uses a `json.Encoder`
on a buffer with `SetEscapeHTML(false)` and trims the trailing `\n`. Use it in
`encode` for object keys and for scalars; scalars never need indenting, so
drop `MarshalIndent`. `json.Number` values encode as their literal text.

**Verify** (after steps 2–4): `go test ./internal/settings/ -v` → all `PASS`, including the 4 old tests. If an old test asserts behaviour this plan deliberately changes, update that test and say so in the commit message. The likely one is `TestWriteDropsOnOffAndEmptyKey`'s `.bak` assertion; its second write still changes the file, so it should pass unchanged. Any other old-test change is a STOP.

### Step 5: Validate before any link changes

At the start of the write path, before calling `links.Apply` or `links.Link`, call `settings.Check(paths.Settings)` and return its error. Do this in:

- `internal/cli/set.go` `applyStates`, before `save`.
- `internal/cli/sync.go` `RunE`, before the link loop.
- `internal/tui/review.go` `save()`. On error, set the status to `"settings.json can't be updated: " + err.Error()` with the error flag, go back to `modeGarden`, and return without touching links.

Tests:

- `internal/cli/cli_test.go`: add `TestInvalidSettingsChangesNoLinks`. Write `{not json` to `h.settings`, then run `h.run("set", "alpha", "name-only")`. Expect a non-zero code, stderr containing `settings.json`, and no link at `filepath.Join(h.links, "alpha")`.
- `internal/tui/app_test.go`: add `TestSaveWithInvalidSettingsChangesNoLinks` using the fixture (`newFixture`, `press`). Write invalid JSON to `f.paths.Settings`, toggle alpha on (`space`), then `s` and `enter`. Assert no link at `filepath.Join(f.paths.LinkDir, "alpha")` and a view containing `can't be updated`.

**Verify**: `go test ./...` → all `ok`.

## Test plan

There are 7 new settings tests (step 1), 1 new CLI test and 1 new TUI test (step 5). Pattern: `internal/settings/settings_test.go:47-62` for file-content assertions and `internal/cli/cli_test.go` `TestBadInputIsRejected` for exit codes.

## Done criteria

- [ ] `go test ./...` all `ok`; the 9 new tests exist and pass
- [ ] `grep -n 'os.WriteFile' internal/settings/settings.go` → no matches
- [ ] `grep -n 'MarshalIndent' internal/settings/settings.go` → no matches
- [ ] `grep -n 'settings.Check' internal/cli/set.go internal/cli/sync.go internal/tui/review.go` → one match per file
- [ ] `go vet ./...` exit 0, `gofmt -l .` empty
- [ ] `git status --short` lists only in-scope files
- [ ] `plans/README.md` row 011 updated

## STOP conditions

- The excerpts don't match the live code (beyond plan 010's signature changes).
- Making the old `TestWritePreservesOtherKeysAndOrder` pass needs changes to `scanOrder` or to key ordering.
- `atomicfile.Write` refuses `path+".bak"` in a case the tests need. Report it; don't change atomicfile.
- You find that a caller passes keys that are not repo skill names to `WriteOverrides`.

## Maintenance notes

- `WriteOverrides` now owns only the keys it is given. Any new caller must pass exactly the repo's skill names, or it will leave stale entries behind.
- Claude Code may write settings.json concurrently, for example when the user accepts a permission. There is still no lock; the window is short. Deferred.
- If the user ever adds comments to settings.json (JSONC), `Check` refuses it with a clear error, which is the intended behaviour.
