package refresh

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jorden-parker/skills/internal/skills"
)

var oldInstructions = strings.Repeat("Follow these old instructions.\n", 10)

func newRepo(t *testing.T) (skills.Repo, skills.Paths) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "repo with spaces")
	flat := filepath.Join(dir, "skills", "example")
	if err := os.MkdirAll(flat, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flat, "SKILL.md"), []byte(oldInstructions), 0o644); err != nil {
		t.Fatal(err)
	}
	commit := []string{"-c", "user.name=T", "-c", "user.email=t@example.test", "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m"}
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, append(commit, "initial")...)
	if err := os.MkdirAll(filepath.Join(dir, "skills", "me"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "mv", "skills/example", "skills/me/example")
	src := filepath.Join(dir, "skills", "me", "example")
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(oldInstructions+"moved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, append(commit, "group by author")...)
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("current instructions"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := skills.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(base, "claude"))
	t.Setenv("CODEX_HOME", filepath.Join(base, "codex"))
	return r, skills.DefaultPaths()
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRootsAreDeduplicated(t *testing.T) {
	_, p := newRepo(t)
	p.CodexDir = filepath.Join(p.Home, ".agents")
	got := Roots(p)
	want := []string{p.LinkDir, p.ClaudeLinkDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %v, want %v", got, want)
	}
}

func TestKnownVersionsFollowTheMoveIntoAnAuthorFolder(t *testing.T) {
	r, _ := newRepo(t)
	v := KnownVersions(r.Dir, filepath.Join("skills", "me", "example", "SKILL.md"))
	for _, body := range []string{oldInstructions, oldInstructions + "moved\n", "current instructions"} {
		if _, ok := v[body]; !ok {
			t.Fatalf("missing %q in %v", body, v)
		}
	}
	if _, ok := v["never"]; ok || len(v) != 3 {
		t.Fatalf("versions = %v", v)
	}
}

func TestRunUsesTheClockForBackups(t *testing.T) {
	r, p := newRepo(t)
	target := filepath.Join(p.LinkDir, "example")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte(oldInstructions), 0o644); err != nil {
		t.Fatal(err)
	}
	now := func() time.Time { return time.Date(2026, 10, 8, 13, 57, 12, 330825000, time.UTC) }
	results, failed := Run(r, p, nil, false, now)
	if failed || len(results) != 1 || results[0].Status != Linked {
		t.Fatalf("results = %+v failed = %v", results, failed)
	}
	dir := filepath.Base(filepath.Dir(results[0].Backup))
	if len(dir) != len("20261008-135712-330825-c10318b8") || dir[:22] != "20261008-135712-330825" {
		t.Fatalf("backup dir = %s", dir)
	}
	if results[0].Target != target || results[0].Source != r.Source("me/example") {
		t.Fatalf("result = %+v", results[0])
	}
	results, failed = Run(r, p, []string{"me/example"}, true, nil)
	if failed || results[0].Status != Current {
		t.Fatalf("second run = %+v", results)
	}
}

func TestRunAcceptsALinkToAnySiblingAndRefreshesCopiesToTheMatchingOne(t *testing.T) {
	r, p := newRepo(t)
	writeManifest(t, filepath.Join(r.SrcDir, "other", "example"), "sibling instructions")
	if err := os.MkdirAll(p.LinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(r.Source("other/example"), filepath.Join(p.LinkDir, "example")); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(p.ClaudeLinkDir, "example")
	writeManifest(t, copied, oldInstructions)
	results, failed := Run(r, p, nil, false, nil)
	if failed || len(results) != 2 {
		t.Fatalf("results = %+v failed = %v", results, failed)
	}
	if results[0].Status != Current || results[1].Status != Linked || results[1].Source != r.Source("me/example") {
		t.Fatalf("results = %+v", results)
	}
}

func TestRunRelinksADanglingFlatLink(t *testing.T) {
	r, p := newRepo(t)
	if err := os.MkdirAll(p.LinkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(p.LinkDir, "example")
	if err := os.Symlink(filepath.Join(r.SrcDir, "example"), target); err != nil {
		t.Fatal(err)
	}
	results, failed := Run(r, p, nil, true, nil)
	if !failed || len(results) != 1 || results[0].Status != Refresh {
		t.Fatalf("check = %+v failed = %v", results, failed)
	}
	results, failed = Run(r, p, nil, false, nil)
	if failed || results[0].Status != Linked {
		t.Fatalf("run = %+v failed = %v", results, failed)
	}
	if !skills.ResolvesTo(target, r.Source("me/example")) {
		t.Fatal("not relinked")
	}
}
