package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newRepo(t *testing.T) Repo {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "repo with spaces")
	for _, name := range []string{"beta", "alpha"} {
		src := filepath.Join(dir, "skills", name)
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n"
		if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A directory without a manifest is not a skill.
	if err := os.MkdirAll(filepath.Join(dir, "skills", "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNamesAreSortedAndNeedManifest(t *testing.T) {
	r := newRepo(t)
	names, err := r.Names()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alpha", "beta"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if r.Has("notes") || r.Has("../alpha") || !r.Has("alpha") {
		t.Fatal("Has is wrong")
	}
}

func TestDescribeCutsAt90Characters(t *testing.T) {
	r := newRepo(t)
	long := strings.Repeat("é", 120)
	body := "---\ndescription:   " + long + "\n---\n"
	if err := os.WriteFile(filepath.Join(r.SrcDir, "alpha", "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := r.Describe("alpha"); got != strings.Repeat("é", 90) {
		t.Fatalf("Describe = %q (%d runes)", got, len([]rune(got)))
	}
	if got := r.Describe("beta"); got != "beta skill" {
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
	if err := os.MkdirAll(filepath.Join(home, "src", "skills", "skills"), 0o755); err != nil {
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
	if err == nil || !strings.Contains(err.Error(), "no skills/ directory in "+dir) || !strings.Contains(err.Error(), "SKILLS_REPO") {
		t.Fatalf("err = %v", err)
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
	want := []Entry{{"alpha", NameOnly}, {"beta", Off}}
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
