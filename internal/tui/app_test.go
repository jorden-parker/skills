package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jorden-parker/skills/internal/links"
	"github.com/jorden-parker/skills/internal/settings"
	"github.com/jorden-parker/skills/internal/skills"
)

type fixture struct {
	t     *testing.T
	base  string
	repo  skills.Repo
	paths skills.Paths
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "repo with spaces")
	for _, id := range []string{"me/alpha", "me/beta"} {
		writeSkill(t, dir, id)
	}
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(base, "claude"))
	t.Setenv("CODEX_HOME", filepath.Join(base, "codex"))
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("SKILLS_REGISTRY_URL", "http://127.0.0.1:1")
	r, err := skills.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, base: base, repo: r, paths: skills.DefaultPaths()}
}

func writeSkill(t *testing.T, dir, id string) {
	t.Helper()
	name := filepath.Base(id)
	src := filepath.Join(dir, "catalog", id)
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n# " + name + "\n\nBody of " + name + ".\n"
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// set applies states the way the CLI does, so the garden starts from them.
func (f *fixture) set(states map[string]skills.State) {
	f.t.Helper()
	entries, _ := skills.LoadStates(f.repo, f.paths)
	for i := range entries {
		if st, ok := states[entries[i].Name]; ok {
			entries[i].State = st
		}
	}
	after, err := links.Apply(f.repo, f.paths, entries, func(string) {})
	if err != nil {
		f.t.Fatal(err)
	}
	if err := settings.WriteOverrides(f.paths.Settings, skills.Overrides(after)); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) states() map[string]skills.State {
	f.t.Helper()
	entries, err := skills.LoadStates(f.repo, f.paths)
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[string]skills.State{}
	for _, e := range entries {
		out[e.Name] = e.State
	}
	return out
}

const registryIndexXML = `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://www.skills.sh/sitemap-skills-1.xml</loc></sitemap>
</sitemapindex>`

const registryUrlsetXML = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://www.skills.sh/acme/tools/lint</loc></url>
  <url><loc>https://www.skills.sh/acme/tools/me-alpha</loc></url>
</urlset>`

const lintRegistryBody = "---\nname: lint\ndescription: lint skill\n---\n# lint\n\nRegistry body of lint.\n"

func serveRegistry(t *testing.T, failIndex bool) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case failIndex:
			http.Error(w, "down", http.StatusInternalServerError)
		case r.URL.Path == "/sitemap.xml":
			_, _ = w.Write([]byte(registryIndexXML))
		case r.URL.Path == "/sitemap-skills-1.xml":
			_, _ = w.Write([]byte(registryUrlsetXML))
		case r.URL.Path == "/api/download/acme/tools/lint":
			files := []map[string]string{{"path": "SKILL.md", "contents": lintRegistryBody}}
			_ = json.NewEncoder(w).Encode(map[string]any{"files": files, "hash": "h"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("SKILLS_REGISTRY_URL", server.URL)
	return &hits
}

func rowNames(m *model) []string {
	var names []string
	for _, row := range m.items() {
		names = append(names, row.(item).name)
	}
	return names
}

func stateOf(m *model, name string) skills.State {
	for _, e := range m.staged {
		if e.Name == name {
			return e.State
		}
	}
	return ""
}

func TestBrowseShowsRegistrySkillsThenReturnsToCatalog(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, false)
	m := f.open()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if m.view != registryView || cmd == nil {
		t.Fatalf("view = %v, cmd nil = %v", m.view, cmd == nil)
	}
	drain(m, cmd, 0)
	if got := rowNames(m); !reflect.DeepEqual(got, []string{"acme/tools/lint", "acme/tools/me-alpha"}) {
		t.Fatalf("registry rows = %v", got)
	}
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "skills.sh") {
		t.Fatalf("header:\n%s", view)
	}
	press(m, "b")
	if m.view != catalogView || !reflect.DeepEqual(rowNames(m), []string{"me/alpha", "me/beta"}) {
		t.Fatalf("catalog rows = %v", rowNames(m))
	}
}

func TestBrowseLoadsTheIndexOnlyOnce(t *testing.T) {
	f := newFixture(t)
	hits := serveRegistry(t, false)
	m := f.open()
	press(m, "b")
	if err := os.Remove(f.paths.RegistryIndex); err != nil {
		t.Fatal(err)
	}
	before := hits.Load()
	press(m, "b", "b")
	if hits.Load() != before {
		t.Fatalf("requests after second browse = %d, want %d", hits.Load(), before)
	}
}

func TestSelectingRegistrySkillFetchesPreviewOnce(t *testing.T) {
	f := newFixture(t)
	hits := serveRegistry(t, false)
	m := f.open()
	press(m, "b")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "Registry body of lint.") || !strings.Contains(view, "skills.sh/acme/tools/lint") {
		t.Fatalf("preview:\n%s", view)
	}
	press(m, "down")
	afterDown := hits.Load()
	press(m, "up")
	if hits.Load() != afterDown {
		t.Fatalf("reselecting made a request: %d -> %d", afterDown, hits.Load())
	}
}

func TestSpaceThenSaveDownloadsAndLinksRegistrySkill(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, false)
	m := f.open()
	press(m, "b")
	press(m, "space", "s", "enter")
	if m.mode != modeGarden || m.pending() != 0 {
		t.Fatalf("mode %v, pending %d, status %q", m.mode, m.pending(), m.status)
	}
	if _, err := os.Stat(filepath.Join(f.paths.RegistryDir, "acme", "tools", "lint", "SKILL.md")); err != nil {
		t.Fatalf("download missing: %v", err)
	}
	press(m, "b")
	if got := f.states()["acme/tools/lint"]; got != skills.On {
		t.Fatalf("lint = %s, want on", got)
	}
}

func TestAllOnAndAllOffStayInTheCatalogView(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, false)
	m := f.open()
	press(m, "b")
	press(m, "a")
	if !strings.Contains(m.status, "catalog view") {
		t.Fatalf("status = %q", m.status)
	}
	press(m, "n")
	if !strings.Contains(m.status, "catalog view") {
		t.Fatalf("status = %q", m.status)
	}
	press(m, "b")
	press(m, "a")
	if stateOf(m, "me/alpha") != skills.On {
		t.Fatalf("all on did not turn on me/alpha")
	}
	for _, e := range m.staged {
		if skills.IsRegistryID(e.Name) && !m.repo.Has(e.Name) && e.State != skills.Off {
			t.Fatalf("%s turned %s on", e.Name, e.State)
		}
	}
}

func TestLoadingTheIndexLeavesCatalogCountsAlone(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, false)
	m := f.open()
	press(m, "b", "b")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "0 of 2 growing") {
		t.Fatalf("header:\n%s", view)
	}
}

func TestIndexFailureKeepsGardenUsable(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, true)
	m := f.open()
	press(m, "b")
	if m.view != registryView || !m.statusErr || !strings.Contains(m.status, "skills.sh") {
		t.Fatalf("status %q, err %v", m.status, m.statusErr)
	}
	press(m, "b")
	if !reflect.DeepEqual(rowNames(m), []string{"me/alpha", "me/beta"}) {
		t.Fatalf("catalog rows = %v", rowNames(m))
	}
}

func TestReloadInSkillsShViewKeepsRegistryRows(t *testing.T) {
	f := newFixture(t)
	serveRegistry(t, false)
	m := f.open()
	press(m, "b", "R")
	if m.view != registryView || !reflect.DeepEqual(rowNames(m), []string{"acme/tools/lint", "acme/tools/me-alpha"}) {
		t.Fatalf("rows after reload = %v", rowNames(m))
	}
}

func (f *fixture) open() *model {
	m := newModel(f.repo, f.paths)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

// drain runs a command and feeds its messages back, the way the program
// would, so forms that progress through commands reach their end state.
// Spinner ticks are dropped so a running refresh does not loop forever.
func drain(m *model, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 8 {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	if _, ok := msg.(spinner.TickMsg); ok {
		return
	}
	// tea.BatchMsg and tea.Sequence's message are both slices of commands.
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		for i := 0; i < v.Len(); i++ {
			drain(m, v.Index(i).Interface().(tea.Cmd), depth+1)
		}
		return
	}
	if _, ok := msg.(tea.QuitMsg); ok {
		return
	}
	_, next := m.Update(msg)
	drain(m, next, depth+1)
}

func press(m *model, keys ...string) tea.Cmd {
	var last tea.Cmd
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "up":
			msg = tea.KeyPressMsg{Code: tea.KeyUp}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		_, last = m.Update(msg)
		drain(m, last, 0)
	}
	return last
}

// quits reports whether a key's command is tea.Quit.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func selectSkill(t *testing.T, m *model, name string) {
	t.Helper()
	for i, e := range m.staged {
		if e.Name == name {
			m.list.Select(i)
			m.refreshPreview()
			return
		}
	}
	t.Fatalf("no skill %s", name)
}

func isLink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func TestStageThenSavePreservesPartialStates(t *testing.T) {
	f := newFixture(t)
	f.set(map[string]skills.State{"me/alpha": skills.NameOnly})
	m := f.open()
	selectSkill(t, m, "me/beta")
	press(m, "space")
	if m.pending() != 1 || m.staged[1].State != skills.On {
		t.Fatalf("staged = %v", m.staged)
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "1 pending") || !strings.Contains(view, "2 of 2 growing") {
		t.Fatalf("header:\n%s", view)
	}
	press(m, "s")
	if m.mode != modeReview {
		t.Fatal("s did not open review")
	}
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "beta  off -> on") {
		t.Fatalf("review:\n%s", view)
	}
	press(m, "enter")
	if m.mode != modeGarden || m.pending() != 0 {
		t.Fatalf("mode %v pending %d", m.mode, m.pending())
	}
	want := map[string]skills.State{"me/alpha": skills.NameOnly, "me/beta": skills.On}
	if got := f.states(); !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %v", got)
	}
	if !isLink(filepath.Join(f.paths.ClaudeLinkDir, "beta")) {
		t.Fatal("beta not linked for Claude")
	}
	if !strings.Contains(m.status, "Saved to") || !strings.Contains(m.status, ".bak") || !strings.Contains(m.status, "new Claude Code session") {
		t.Fatalf("status = %q", m.status)
	}
}

func (f *fixture) chdirToNewProject() string {
	f.t.Helper()
	dir := filepath.Join(f.base, "project")
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		f.t.Fatalf("git init: %v\n%s", err, out)
	}
	f.t.Chdir(dir)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		f.t.Fatal(err)
	}
	return resolved
}

func TestScopeKeyLinksIntoTheGitProject(t *testing.T) {
	f := newFixture(t)
	project := f.chdirToNewProject()
	m := f.open()
	press(m, "p")
	if m.paths.Scope != skills.ProjectScope || !strings.HasPrefix(m.paths.LinkDir, project) {
		t.Fatalf("paths = %+v", m.paths)
	}
	selectSkill(t, m, "me/alpha")
	press(m, "space", "s", "enter")
	if !isLink(filepath.Join(project, ".agents", "skills", "alpha")) {
		t.Fatal("alpha not linked in the project")
	}
	if isLink(filepath.Join(f.paths.LinkDir, "alpha")) {
		t.Fatal("alpha linked globally")
	}
	press(m, "p")
	if m.paths.Scope != skills.GlobalScope {
		t.Fatalf("scope = %s, want global", m.paths.Scope)
	}
	if stagedStates(m)["me/alpha"] != skills.Off {
		t.Fatalf("global alpha = %v, want off", stagedStates(m)["me/alpha"])
	}
}

func TestScopeKeyWaitsForStagedChanges(t *testing.T) {
	f := newFixture(t)
	f.chdirToNewProject()
	m := f.open()
	selectSkill(t, m, "me/alpha")
	press(m, "space", "p")
	if m.paths.Scope != skills.GlobalScope {
		t.Fatal("scope changed with changes staged")
	}
	if !strings.Contains(m.status, "Save or discard") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestRefreshDoesNotRunInProjectScope(t *testing.T) {
	f := newFixture(t)
	f.chdirToNewProject()
	m := f.open()
	press(m, "p", "r")
	if m.mode != modeGarden || !strings.Contains(m.status, "global") {
		t.Fatalf("mode %v status %q", m.mode, m.status)
	}
}

func TestScopeKeyOutsideGitStaysGlobal(t *testing.T) {
	f := newFixture(t)
	t.Chdir(t.TempDir())
	m := f.open()
	press(m, "p")
	if m.paths.Scope != skills.GlobalScope || !m.statusErr || !strings.Contains(m.status, "not inside a Git repository") {
		t.Fatalf("scope %s status %q", m.paths.Scope, m.status)
	}
}

func TestCanStartWithNoEnabledSkills(t *testing.T) {
	f := newFixture(t)
	m := f.open()
	if m.growing() != 0 {
		t.Fatal("something is on")
	}
	selectSkill(t, m, "me/beta")
	press(m, "tab", "s", "y")
	want := map[string]skills.State{"me/alpha": skills.Off, "me/beta": skills.On}
	if got := f.states(); !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %v", got)
	}
}

func TestDiscardWritesNothing(t *testing.T) {
	f := newFixture(t)
	m := f.open()
	selectSkill(t, m, "me/alpha")
	press(m, "t")
	if m.mode != modeTune || m.tune == nil {
		t.Fatal("t did not open the tune form")
	}
	// The form opens on the current state (off, second option); enter picks it
	// unchanged, so move up to "on" first.
	press(m, "up", "enter")
	if m.mode != modeGarden || m.staged[0].State != skills.On {
		t.Fatalf("mode %v staged %v", m.mode, m.staged)
	}
	press(m, "s", "d")
	if m.pending() != 0 || m.mode != modeGarden {
		t.Fatalf("discard left %d pending in mode %v", m.pending(), m.mode)
	}
	if _, err := os.Lstat(f.paths.Settings); err == nil {
		t.Fatal("settings.json was written")
	}
	if _, err := os.Lstat(f.paths.LinkDir); err == nil {
		t.Fatal("link directory was created")
	}
	if !quits(press(m, "q")) {
		t.Fatal("q did not quit")
	}
}

func TestQuitWithPendingAsksFirst(t *testing.T) {
	f := newFixture(t)
	m := f.open()
	press(m, "space")
	if quits(press(m, "q")) || m.mode != modeConfirmQuit {
		t.Fatal("q quit without asking")
	}
	press(m, "n")
	if m.mode != modeGarden || m.pending() != 1 {
		t.Fatal("n did not keep the staged change")
	}
	press(m, "esc")
	if !quits(press(m, "y")) {
		t.Fatal("y did not quit")
	}
	if _, err := os.Lstat(f.paths.Settings); err == nil {
		t.Fatal("quitting wrote settings")
	}
}

func TestSaveWithoutChangesLeavesSettingsAlone(t *testing.T) {
	f := newFixture(t)
	f.set(map[string]skills.State{"me/alpha": skills.On})
	before, err := os.ReadFile(f.paths.Settings)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(f.paths.Settings)
	m := f.open()
	press(m, "s")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "Your garden is already just so. No changes to save.") {
		t.Fatalf("review:\n%s", view)
	}
	press(m, "enter")
	if m.mode != modeGarden {
		t.Fatal("enter did not return to the garden")
	}
	after, _ := os.ReadFile(f.paths.Settings)
	info2, _ := os.Stat(f.paths.Settings)
	if string(after) != string(before) || !info2.ModTime().Equal(info.ModTime()) {
		t.Fatal("settings.json was rewritten")
	}
}

func TestTuneCanTurnOffLastSkill(t *testing.T) {
	f := newFixture(t)
	f.set(map[string]skills.State{"me/alpha": skills.On})
	m := f.open()
	selectSkill(t, m, "me/alpha")
	press(m, "enter")
	if m.mode != modeTune {
		t.Fatal("enter did not open the tune form")
	}
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "off — Put this one to bed") || !strings.Contains(view, "user-invocable-only — Only when you call it") {
		t.Fatalf("tune:\n%s", view)
	}
	press(m, "down", "enter")
	if m.staged[0].State != skills.Off {
		t.Fatalf("staged = %v", m.staged)
	}
	press(m, "s", "enter")
	want := map[string]skills.State{"me/alpha": skills.Off, "me/beta": skills.Off}
	if got := f.states(); !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %v", got)
	}
	if _, err := os.Lstat(filepath.Join(f.paths.ClaudeLinkDir, "alpha")); err == nil {
		t.Fatal("alpha still linked for Claude")
	}
}

func TestAllOnKeepsPartialStatesAndAllOffClears(t *testing.T) {
	f := newFixture(t)
	f.set(map[string]skills.State{"me/alpha": skills.UserInvocableOnly})
	m := f.open()
	press(m, "a")
	if m.staged[0].State != skills.UserInvocableOnly || m.staged[1].State != skills.On {
		t.Fatalf("staged = %v", m.staged)
	}
	press(m, "n")
	if m.growing() != 0 {
		t.Fatalf("staged = %v", m.staged)
	}
	press(m, "R")
	if m.pending() != 0 || m.staged[0].State != skills.UserInvocableOnly {
		t.Fatalf("reload: %v", m.staged)
	}
}

func TestRefreshScreenPlansThenApplies(t *testing.T) {
	f := newFixture(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", f.repo.Dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("add", ".")
	git("-c", "user.name=Test", "-c", "user.email=test@example.test", "-c", "core.hooksPath=/dev/null", "commit", "-m", "initial")
	old, _ := os.ReadFile(filepath.Join(f.repo.SrcDir, "me", "alpha", "SKILL.md"))
	if err := os.WriteFile(filepath.Join(f.repo.SrcDir, "me", "alpha", "SKILL.md"), []byte("newer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(f.paths.LinkDir, "alpha")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "SKILL.md"), old, 0o644); err != nil {
		t.Fatal(err)
	}
	m := f.open()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if m.mode != modeRefresh || !m.rf.running || cmd == nil {
		t.Fatal("r did not start a refresh")
	}
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "Looking through") {
		t.Fatalf("spinner view:\n%s", view)
	}
	drain(m, cmd, 0)
	if m.rf.running {
		t.Fatal("refresh did not finish")
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "refresh "+tildify(target)) || !strings.Contains(view, "Press enter to refresh 1") {
		t.Fatalf("plan:\n%s", view)
	}
	if isLink(target) {
		t.Fatal("planning changed files")
	}
	press(m, "enter")
	view = ansi.Strip(m.View().Content)
	if !strings.Contains(view, "linked  "+tildify(target)) || !strings.Contains(view, "backup at") {
		t.Fatalf("apply:\n%s", view)
	}
	if !isLink(target) {
		t.Fatal("apply did not link")
	}
	backups, _ := filepath.Glob(filepath.Join(f.base, "home", ".agents", "skill-backups", "*", "alpha"))
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	press(m, "esc")
	if m.mode != modeGarden || m.staged[0].State != skills.On {
		t.Fatalf("after refresh: mode %v staged %v", m.mode, m.staged)
	}
}

func TestViewFitsTheTerminal(t *testing.T) {
	f := newFixture(t)
	f.set(map[string]skills.State{"me/alpha": skills.NameOnly})
	m := f.open()
	for _, size := range [][2]int{{80, 24}, {120, 40}, {50, 20}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, open := range []string{"", "?", "s"} {
			if open != "" {
				press(m, open)
			}
			view := ansi.Strip(m.View().Content)
			if h, w := lipgloss.Height(view), lipgloss.Width(view); h > size[1] || w > size[0] {
				t.Fatalf("%dx%d mode %v: view is %dx%d\n%s", size[0], size[1], m.mode, w, h, view)
			}
			if open != "" {
				press(m, "esc")
			}
		}
		view := ansi.Strip(m.View().Content)
		for _, text := range []string{"The skill garden", "alpha", "name-only", "beta skill", "1 of 2 growing", "? keys", "q quit"} {
			if !strings.Contains(view, text) {
				t.Fatalf("%dx%d: missing %q\n%s", size[0], size[1], text, view)
			}
		}
		if size[0] >= 60 && !strings.Contains(view, "Body of alpha") {
			t.Fatalf("%dx%d: no preview\n%s", size[0], size[1], view)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	press(m, "ctrl+d")
	if view := ansi.Strip(m.View().Content); !m.details || !strings.Contains(view, "Body of alpha") {
		t.Fatalf("ctrl+d: details=%v\n%s", m.details, view)
	}
}

func TestToggleAndAllOnKeepOneSkillPerName(t *testing.T) {
	f := newFixture(t)
	writeSkill(t, f.repo.Dir, "other/alpha")
	f.set(map[string]skills.State{"me/alpha": skills.NameOnly})
	m := f.open()
	press(m, "a")
	want := map[string]skills.State{"me/alpha": skills.NameOnly, "me/beta": skills.On, "other/alpha": skills.Off}
	if got := stagedStates(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("after a: %v", got)
	}
	selectSkill(t, m, "other/alpha")
	press(m, "space")
	want = map[string]skills.State{"me/alpha": skills.Off, "me/beta": skills.On, "other/alpha": skills.On}
	if got := stagedStates(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("after toggle: %v", got)
	}
	press(m, "s", "enter")
	if got := f.states(); !reflect.DeepEqual(got, want) {
		t.Fatalf("saved: %v", got)
	}
}

func stagedStates(m *model) map[string]skills.State {
	out := map[string]skills.State{}
	for _, e := range m.staged {
		out[e.Name] = e.State
	}
	return out
}
