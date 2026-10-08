package links

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jorden-parker/skills/internal/skills"
)

func setup(t *testing.T) (skills.Repo, skills.Paths) {
	t.Helper()
	base := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		src := filepath.Join(base, "repo", "skills", name)
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := skills.Find(filepath.Join(base, "repo"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(base, "claude"))
	return r, skills.DefaultPaths()
}

func isLink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func TestLinkCreatesBothAndRelinks(t *testing.T) {
	r, p := setup(t)
	var warned []string
	warn := func(s string) { warned = append(warned, s) }
	blocked, err := Link(r.Source("alpha"), "alpha", p.LinkDirs(), warn)
	if err != nil || blocked || len(warned) != 0 {
		t.Fatal(blocked, err, warned)
	}
	for _, dir := range p.LinkDirs() {
		if dest, _ := os.Readlink(filepath.Join(dir, "alpha")); dest != r.Source("alpha") {
			t.Fatalf("%s -> %s", dir, dest)
		}
	}
	// An existing link, even to somewhere else, is replaced.
	if err := os.Remove(filepath.Join(p.LinkDir, "alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/elsewhere", filepath.Join(p.LinkDir, "alpha")); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(r.Source("alpha"), "alpha", p.LinkDirs(), warn); err != nil {
		t.Fatal(err)
	}
	if dest, _ := os.Readlink(filepath.Join(p.LinkDir, "alpha")); dest != r.Source("alpha") {
		t.Fatalf("not relinked: %s", dest)
	}
}

func TestRealDirectoryBlocksAndSurvives(t *testing.T) {
	r, p := setup(t)
	real := filepath.Join(p.ClaudeLinkDir, "alpha")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "local.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	var warned []string
	warn := func(s string) { warned = append(warned, s) }
	blocked, err := Link(r.Source("alpha"), "alpha", p.LinkDirs(), warn)
	if err != nil || !blocked {
		t.Fatal(blocked, err)
	}
	if len(warned) != 1 || warned[0] != real+" - a real directory or file is already there, not touching it" {
		t.Fatalf("warned = %q", warned)
	}
	if !isLink(filepath.Join(p.LinkDir, "alpha")) || isLink(real) {
		t.Fatal("link dir not linked or real dir replaced")
	}
	warned = nil
	blocked, err = Unlink("alpha", p.LinkDirs(), warn)
	if err != nil || !blocked {
		t.Fatal(blocked, err)
	}
	if len(warned) != 1 || warned[0] != real+" - a real directory or file is there, not deleting it" {
		t.Fatalf("warned = %q", warned)
	}
	if isLink(filepath.Join(p.LinkDir, "alpha")) {
		t.Fatal("link dir not unlinked")
	}
	if b, _ := os.ReadFile(filepath.Join(real, "local.txt")); string(b) != "mine" {
		t.Fatal("real dir touched")
	}
}

func TestApplyKeepsBlockedUnlinkOn(t *testing.T) {
	r, p := setup(t)
	if err := os.MkdirAll(filepath.Join(p.LinkDir, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Apply(r, p, []skills.Entry{{Name: "alpha", State: skills.Off}, {Name: "beta", State: skills.NameOnly}}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	want := []skills.Entry{{Name: "alpha", State: skills.On}, {Name: "beta", State: skills.NameOnly}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if !isLink(filepath.Join(p.ClaudeLinkDir, "beta")) {
		t.Fatal("beta not linked")
	}
}

func TestPruneRemovesOnlyDanglingLinksIntoRepo(t *testing.T) {
	r, p := setup(t)
	for _, dir := range p.LinkDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		must := func(err error) {
			if err != nil {
				t.Fatal(err)
			}
		}
		must(os.Symlink(r.Source("alpha"), filepath.Join(dir, "alpha")))
		must(os.Symlink(r.Source("gone"), filepath.Join(dir, "gone")))
		must(os.Symlink(filepath.Join(r.Dir, "elsewhere"), filepath.Join(dir, "other")))
		must(os.Symlink(filepath.Join(r.Dir, "skills-old", "x"), filepath.Join(dir, "lookalike")))
	}
	removed, err := Prune(r.SrcDir, p.LinkDirs())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(p.LinkDir, "gone"), filepath.Join(p.ClaudeLinkDir, "gone")}
	if !reflect.DeepEqual(removed, want) {
		t.Fatalf("removed = %v", removed)
	}
	for _, dir := range p.LinkDirs() {
		for _, n := range []string{"alpha", "other", "lookalike"} {
			if !isLink(filepath.Join(dir, n)) {
				t.Fatalf("%s/%s was removed", dir, n)
			}
		}
	}
}
