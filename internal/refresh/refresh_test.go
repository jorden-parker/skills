package refresh

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jorden-parker/skills/internal/skills"
)

func newRepo(t *testing.T) (skills.Repo, skills.Paths) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "repo with spaces")
	src := filepath.Join(dir, "skills", "example")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("old instructions"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@example.test", "-c", "core.hooksPath=/dev/null", "commit", "-m", "initial"}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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

func TestRootsAreDeduplicated(t *testing.T) {
	_, p := newRepo(t)
	p.CodexDir = filepath.Join(p.Home, ".agents")
	got := Roots(p)
	want := []string{p.LinkDir, p.ClaudeLinkDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %v, want %v", got, want)
	}
}

func TestKnownVersionsIncludeHistory(t *testing.T) {
	r, _ := newRepo(t)
	v := KnownVersions(r.Dir, filepath.Join("skills", "example", "SKILL.md"))
	for _, body := range []string{"old instructions", "current instructions"} {
		if _, ok := v[body]; !ok {
			t.Fatalf("missing %q in %v", body, v)
		}
	}
	if _, ok := v["never"]; ok || len(v) != 2 {
		t.Fatalf("versions = %v", v)
	}
}

func TestUnknownIsSortedAndUnique(t *testing.T) {
	r, _ := newRepo(t)
	got := Unknown(r, []string{"zeta", "example", "alpha", "zeta"})
	if want := []string{"alpha", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown = %v", got)
	}
}

func TestRunUsesTheClockForBackups(t *testing.T) {
	r, p := newRepo(t)
	target := filepath.Join(p.LinkDir, "example")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("old instructions"), 0o644); err != nil {
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
	if results[0].Target != target || results[0].Source != r.Source("example") {
		t.Fatalf("result = %+v", results[0])
	}
	results, failed = Run(r, p, []string{"example"}, true, nil)
	if failed || results[0].Status != Current {
		t.Fatalf("second run = %+v", results)
	}
}
