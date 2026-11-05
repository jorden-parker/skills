package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newRepo(t *testing.T) Repo {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo with spaces")
	for _, id := range []string{"me/beta", "me/alpha"} {
		writeSkill(t, dir, id)
	}
	if err := os.MkdirAll(filepath.Join(dir, "catalog", "me", "notes-without-manifest"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func writeSkill(t *testing.T, dir, id string) {
	t.Helper()
	name := filepath.Base(id)
	src := filepath.Join(dir, "catalog", id)
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n"
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNamesAreSortedAndNeedManifest(t *testing.T) {
	r := newRepo(t)
	names, err := r.Names()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"me/alpha", "me/beta"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for _, bad := range []string{"me/notes-without-manifest", "alpha", "me/../me/alpha", "../me/alpha", "me/alpha/x", "me", ""} {
		if r.Has(bad) {
			t.Fatalf("Has(%q) = true", bad)
		}
	}
	if !r.Has("me/alpha") {
		t.Fatal("Has(me/alpha) = false")
	}
}

func TestDescribeCutsAt90Characters(t *testing.T) {
	r := newRepo(t)
	long := strings.Repeat("é", 120)
	body := "---\ndescription:   " + long + "\n---\n"
	if err := os.WriteFile(filepath.Join(r.SrcDir, "me", "alpha", "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := r.Describe("me/alpha"); got != strings.Repeat("é", 90) {
		t.Fatalf("Describe = %q (%d runes)", got, len([]rune(got)))
	}
	if got := r.Describe("me/beta"); got != "beta skill" {
		t.Fatalf("Describe = %q", got)
	}
	if got := r.Describe("missing"); got != "" {
		t.Fatalf("Describe = %q", got)
	}
}

func TestFindOrder(t *testing.T) {
	r := newRepo(t)
	other := newRepo(t)
	t.Setenv("SKILLS_REPO", other.Dir)
	t.Setenv("HOME", t.TempDir())

	got, err := Find(r.Dir)
	if err != nil || got.Dir != r.Dir {
		t.Fatalf("flag: %v %v", got.Dir, err)
	}
	got, err = Find("")
	if err != nil || got.Dir != other.Dir {
		t.Fatalf("env: %v %v", got.Dir, err)
	}
	t.Setenv("SKILLS_REPO", "")
	old := DefaultDir
	DefaultDir = r.Dir
	t.Cleanup(func() { DefaultDir = old })
	got, err = Find("")
	if err != nil || got.Dir != r.Dir {
		t.Fatalf("baked: %v %v", got.Dir, err)
	}
	DefaultDir = ""
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "src", "skills", "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = Find("")
	if err != nil || got.Dir != filepath.Join(home, "src", "skills") {
		t.Fatalf("home: %v %v", got.Dir, err)
	}
}

func TestFindRejectsMissingSkillsDir(t *testing.T) {
	dir := t.TempDir()
	_, err := Find(dir)
	if err == nil || !strings.Contains(err.Error(), "no catalog/ directory in "+dir) || !strings.Contains(err.Error(), "SKILLS_REPO") {
		t.Fatalf("err = %v", err)
	}
}

func TestFindNamesTheCatalogAndTheOldSkillsFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Find(dir)
	if err != nil || r.SrcDir != filepath.Join(dir, "catalog") || r.OldSrcDir != filepath.Join(dir, "skills") {
		t.Fatalf("Find = %+v, %v", r, err)
	}
}

func TestParseState(t *testing.T) {
	if st, err := ParseState("name-only"); err != nil || st != NameOnly {
		t.Fatal(st, err)
	}
	_, err := ParseState("maybe")
	if err == nil || err.Error() != "bad state: maybe (valid: on off name-only user-invocable-only)" {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadStates(t *testing.T) {
	r := newRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude"))
	t.Setenv("CODEX_HOME", "")
	p := DefaultPaths()
	if p.ClaudeLinkDir != filepath.Join(home, "claude", "skills") || p.CodexDir != filepath.Join(home, ".codex") {
		t.Fatalf("paths = %+v", p)
	}
	if err := os.MkdirAll(p.LinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A dangling link still counts as linked; the override gives the state.
	if err := os.Symlink(filepath.Join(r.SrcDir, "gone"), filepath.Join(p.LinkDir, "alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.ClaudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"skillOverrides":{"alpha":"name-only","beta":"user-invocable-only"}}`
	if err := os.WriteFile(p.Settings, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadStates(r, p)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{"me/alpha", NameOnly}, {"me/beta", Off}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %v, want %v", got, want)
	}
	// An off override on a linked skill reads as on.
	if err := os.WriteFile(p.Settings, []byte(`{"skillOverrides":{"alpha":"off"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ = LoadStates(r, p)
	if got[0].State != On {
		t.Fatalf("alpha = %s", got[0].State)
	}
}

func TestLinkName(t *testing.T) {
	if got := LinkName("mattpocock/handoff"); got != "handoff" {
		t.Fatalf("LinkName = %q", got)
	}
}

func TestResolve(t *testing.T) {
	r := newRepo(t)
	writeSkill(t, r.Dir, "other/beta")
	cases := map[string]string{"me/alpha": "me/alpha", "alpha": "me/alpha", "other/beta": "other/beta"}
	for arg, want := range cases {
		if got, err := r.Resolve(arg); err != nil || got != want {
			t.Fatalf("Resolve(%q) = %q, %v", arg, got, err)
		}
	}
	if _, err := r.Resolve("beta"); err == nil || err.Error() != "ambiguous skill: beta (use one of: me/beta, other/beta)" {
		t.Fatalf("ambiguous err = %v", err)
	}
	if _, err := r.Resolve("gamma"); err == nil || err.Error() != "no such skill: gamma" {
		t.Fatalf("missing err = %v", err)
	}
}

func TestStageTurnsSiblingsOff(t *testing.T) {
	entries := []Entry{{"me/beta", On}, {"me/alpha", On}, {"other/beta", Off}}
	Stage(entries, "other/beta", NameOnly)
	want := []Entry{{"me/beta", Off}, {"me/alpha", On}, {"other/beta", NameOnly}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("staged = %v, want %v", entries, want)
	}
	Stage(entries, "other/beta", Off)
	if entries[0].State != Off || entries[2].State != Off {
		t.Fatalf("off staging touched siblings: %v", entries)
	}
}

func TestLoadStatesGivesTheLinkToTheSkillItPointsAt(t *testing.T) {
	r := newRepo(t)
	writeSkill(t, r.Dir, "other/beta")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	p := DefaultPaths()
	if err := os.MkdirAll(p.LinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(r.Source("other/beta"), filepath.Join(p.LinkDir, "beta")); err != nil {
		t.Fatal(err)
	}
	got, err := LoadStates(r, p)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{"me/alpha", Off}, {"me/beta", Off}, {"other/beta", On}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %v, want %v", got, want)
	}
}

func TestOverridesPreferTheLinkedSibling(t *testing.T) {
	got := Overrides([]Entry{{"me/beta", NameOnly}, {"other/beta", Off}, {"me/alpha", Off}})
	want := map[string]string{"beta": "name-only", "alpha": "off"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("overrides = %v, want %v", got, want)
	}
}

func TestParseScope(t *testing.T) {
	for _, s := range []string{"global", "project"} {
		if _, err := ParseScope(s); err != nil {
			t.Fatalf("ParseScope(%q): %v", s, err)
		}
	}
	if _, err := ParseScope("both"); err == nil || !strings.Contains(err.Error(), "valid: global project") {
		t.Fatalf("ParseScope(both) = %v", err)
	}
}

func TestDefaultPathsAreGlobal(t *testing.T) {
	if got := DefaultPaths().Scope; got != GlobalScope {
		t.Fatalf("Scope = %s, want global", got)
	}
}

func TestDefaultPathsRegistryLocations(t *testing.T) {
	t.Setenv("HOME", "/h")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	got := DefaultPaths()
	if got.RegistryDir != "/h/.local/share/skills/registry" || got.RegistryIndex != "/h/.cache/skills/skills-sh-index.txt" {
		t.Fatalf("registry paths = %q, %q", got.RegistryDir, got.RegistryIndex)
	}
}

func TestDefaultPathsRegistryHonoursXDG(t *testing.T) {
	t.Setenv("HOME", "/h")
	t.Setenv("XDG_DATA_HOME", "/d")
	t.Setenv("XDG_CACHE_HOME", "/c")
	got := DefaultPaths()
	if got.RegistryDir != "/d/skills/registry" || got.RegistryIndex != "/c/skills/skills-sh-index.txt" {
		t.Fatalf("registry paths = %q, %q", got.RegistryDir, got.RegistryIndex)
	}
}

func TestPathsForProjectKeepsSharedRegistryCache(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	t.Setenv("XDG_DATA_HOME", "/d")
	t.Setenv("XDG_CACHE_HOME", "/c")
	got, err := PathsFor(ProjectScope, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.RegistryDir != "/d/skills/registry" || got.RegistryIndex != "/c/skills/skills-sh-index.txt" {
		t.Fatalf("project registry paths = %q, %q", got.RegistryDir, got.RegistryIndex)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

func TestPathsForProjectUsesGitTopLevel(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	sub := filepath.Join(root, "pkg", "inner")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := PathsFor(ProjectScope, sub)
	if err != nil {
		t.Fatal(err)
	}
	top, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	global := DefaultPaths()
	if got.Scope != ProjectScope {
		t.Fatalf("Scope = %s", got.Scope)
	}
	if got.LinkDir != filepath.Join(top, ".agents", "skills") {
		t.Fatalf("LinkDir = %s", got.LinkDir)
	}
	if got.ClaudeLinkDir != filepath.Join(top, ".claude", "skills") {
		t.Fatalf("ClaudeLinkDir = %s", got.ClaudeLinkDir)
	}
	if got.Settings != filepath.Join(top, ".claude", "settings.local.json") {
		t.Fatalf("Settings = %s", got.Settings)
	}
	if got.Home != global.Home || got.ClaudeDir != global.ClaudeDir || got.CodexDir != global.CodexDir {
		t.Fatalf("Home, ClaudeDir and CodexDir should match the global paths: %+v", got)
	}
}

func TestPathsForProjectOutsideRepoFails(t *testing.T) {
	_, err := PathsFor(ProjectScope, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not inside a Git repository") {
		t.Fatalf("PathsFor outside a repo = %v", err)
	}
}

func TestPathsForGlobalIgnoresDirectory(t *testing.T) {
	got, err := PathsFor(GlobalScope, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, DefaultPaths()) {
		t.Fatalf("PathsFor(global) = %+v, want %+v", got, DefaultPaths())
	}
}
