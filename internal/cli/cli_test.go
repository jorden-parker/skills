package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type harness struct {
	t                  *testing.T
	base, repo         string
	links, claudeLinks string
	settings           string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	base := t.TempDir()
	h := &harness{t: t, base: base, repo: filepath.Join(base, "repo with spaces")}
	for _, name := range []string{"alpha", "beta"} {
		src := filepath.Join(h.repo, "skills", name)
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + name + "\ndescription: " + name + " skill\n---\n"
		if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	claude := filepath.Join(base, "claude")
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	t.Setenv("CODEX_HOME", filepath.Join(base, "codex"))
	t.Setenv("SKILLS_REPO", "")
	t.Setenv("NO_COLOR", "1")
	h.links = filepath.Join(base, "home", ".agents", "skills")
	h.claudeLinks = filepath.Join(claude, "skills")
	h.settings = filepath.Join(claude, "settings.json")
	return h
}

// run executes the command the way a script would and returns stdout,
// stderr and the exit status.
func (h *harness) run(args ...string) (string, string, int) {
	h.t.Helper()
	var out, errOut bytes.Buffer
	root := Root("skills")
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(bytes.NewReader(nil))
	root.SetArgs(append([]string{"--repo", h.repo}, args...))
	err := root.Execute()
	code := 0
	if err != nil {
		code = 1
		if ex, ok := err.(exitError); ok {
			code = ex.code
		} else {
			errOut.WriteString("error: " + err.Error() + "\n")
		}
	}
	return out.String(), errOut.String(), code
}

func (h *harness) ok(args ...string) string {
	h.t.Helper()
	out, errOut, code := h.run(args...)
	if code != 0 {
		h.t.Fatalf("%v: exit %d\n%s%s", args, code, out, errOut)
	}
	return out
}

func (h *harness) states() map[string]string {
	h.t.Helper()
	found := map[string]string{}
	for _, line := range strings.Split(h.ok("list"), "\n") {
		for _, name := range []string{"alpha", "beta"} {
			if !strings.Contains(line+" ", " "+name+" ") {
				continue
			}
			for _, st := range []string{"user-invocable-only", "name-only", "off", "on"} {
				if strings.Contains(line, st) {
					found[name] = st
					break
				}
			}
		}
	}
	return found
}

func (h *harness) overrides() map[string]string {
	h.t.Helper()
	b, err := os.ReadFile(h.settings)
	if err != nil {
		return nil
	}
	var doc struct {
		Overrides map[string]string `json:"skillOverrides"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		h.t.Fatal(err)
	}
	return doc.Overrides
}

func isLink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func resolves(t *testing.T, path, want string) {
	t.Helper()
	got, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := filepath.EvalSymlinks(want)
	if got != w {
		t.Fatalf("%s -> %s, want %s", path, got, w)
	}
}

func symlink(t *testing.T, old, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(old, link); err != nil {
		t.Fatal(err)
	}
}

func TestListShowsStates(t *testing.T) {
	h := newHarness(t)
	out := h.ok("list")
	if !strings.Contains(out, "Skills in "+filepath.Join(h.repo, "skills")) || !strings.Contains(out, "alpha skill") {
		t.Fatalf("list:\n%s", out)
	}
	if !strings.Contains(out, h.links+" and "+h.claudeLinks+"; off is not") {
		t.Fatalf("list:\n%s", out)
	}
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "off", "beta": "off"}) {
		t.Fatalf("states = %v", got)
	}
}

func TestAddCreatesLinkAndRemoveDeletesIt(t *testing.T) {
	h := newHarness(t)
	out := h.ok("add", "alpha")
	for _, dir := range []string{h.links, h.claudeLinks} {
		if !isLink(filepath.Join(dir, "alpha")) {
			t.Fatalf("%s/alpha not linked", dir)
		}
		resolves(t, filepath.Join(dir, "alpha"), filepath.Join(h.repo, "skills", "alpha"))
	}
	if isLink(filepath.Join(h.links, "beta")) {
		t.Fatal("beta linked")
	}
	if !strings.Contains(out, "Changed\n  alpha          off                 -> on") || !strings.Contains(out, "Start a new Claude Code session") {
		t.Fatalf("add:\n%s", out)
	}
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "on", "beta": "off"}) {
		t.Fatalf("states = %v", got)
	}
	if h.overrides() != nil {
		t.Fatalf("overrides = %v", h.overrides())
	}
	out = h.ok("remove", "alpha")
	if isLink(filepath.Join(h.links, "alpha")) || isLink(filepath.Join(h.claudeLinks, "alpha")) {
		t.Fatal("alpha still linked")
	}
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "off", "beta": "off"}) {
		t.Fatalf("states = %v", got)
	}
	out = h.ok("rm", "alpha")
	if !strings.Contains(out, "No states changed.") {
		t.Fatalf("rm:\n%s", out)
	}
}

func TestPartialStatesKeepLinkAndWriteOverride(t *testing.T) {
	h := newHarness(t)
	h.ok("set", "alpha=name-only", "beta=user-invocable-only")
	for _, p := range []string{filepath.Join(h.links, "alpha"), filepath.Join(h.links, "beta"), filepath.Join(h.claudeLinks, "beta")} {
		if !isLink(p) {
			t.Fatalf("%s not linked", p)
		}
	}
	if got := h.overrides(); !reflect.DeepEqual(got, map[string]string{"alpha": "name-only", "beta": "user-invocable-only"}) {
		t.Fatalf("overrides = %v", got)
	}
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "name-only", "beta": "user-invocable-only"}) {
		t.Fatalf("states = %v", got)
	}
	h.ok("set", "alpha", "off")
	h.ok("set", "beta=on")
	if isLink(filepath.Join(h.links, "alpha")) || isLink(filepath.Join(h.claudeLinks, "alpha")) {
		t.Fatal("alpha still linked")
	}
	if !isLink(filepath.Join(h.links, "beta")) || !isLink(filepath.Join(h.claudeLinks, "beta")) {
		t.Fatal("beta not linked")
	}
	if h.overrides() != nil {
		t.Fatalf("overrides = %v", h.overrides())
	}
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "off", "beta": "on"}) {
		t.Fatalf("states = %v", got)
	}
}

func TestClaudeOnlyLinkDoesNotControlStateAndIsRemoved(t *testing.T) {
	h := newHarness(t)
	link := filepath.Join(h.claudeLinks, "alpha")
	symlink(t, filepath.Join(h.repo, "skills", "alpha"), link)
	if got := h.states(); !reflect.DeepEqual(got, map[string]string{"alpha": "off", "beta": "off"}) {
		t.Fatalf("states = %v", got)
	}
	h.ok("remove", "alpha")
	if isLink(link) {
		t.Fatal("claude link not removed")
	}
}

func TestDefaultClaudeDirectory(t *testing.T) {
	h := newHarness(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	link := filepath.Join(h.base, "home", ".claude", "skills", "alpha")
	h.ok("add", "alpha")
	if !isLink(link) {
		t.Fatal("not linked into ~/.claude/skills")
	}
	resolves(t, link, filepath.Join(h.repo, "skills", "alpha"))
	h.ok("remove", "alpha")
	if isLink(link) {
		t.Fatal("still linked")
	}
}

func TestRealClaudeDirectoryIsPreserved(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(h.claudeLinks, "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(dir, "local.txt")
	if err := os.WriteFile(sentinel, []byte("local skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "alpha"}, {"sync"}, {"remove", "alpha"}} {
		out := h.ok(args...)
		if isLink(dir) {
			t.Fatalf("%v replaced the real directory", args)
		}
		if b, _ := os.ReadFile(sentinel); string(b) != "local skill" {
			t.Fatalf("%v changed the sentinel", args)
		}
		if !strings.Contains(out, "  skip "+dir+" - a real directory or file is ") {
			t.Fatalf("%v did not warn:\n%s", args, out)
		}
	}
	if isLink(filepath.Join(h.links, "alpha")) {
		t.Fatal("alpha still linked in ~/.agents/skills")
	}
}

func TestSyncAddsMissingClaudeLinkForEnabledSkillOnly(t *testing.T) {
	h := newHarness(t)
	symlink(t, filepath.Join(h.repo, "skills", "alpha"), filepath.Join(h.links, "alpha"))
	out := h.ok("sync")
	if !strings.Contains(out, "Syncing links in "+h.links+" and "+h.claudeLinks+"\n  synced alpha\n") {
		t.Fatalf("sync:\n%s", out)
	}
	if !isLink(filepath.Join(h.claudeLinks, "alpha")) {
		t.Fatal("claude link not added")
	}
	resolves(t, filepath.Join(h.claudeLinks, "alpha"), filepath.Join(h.repo, "skills", "alpha"))
	if _, err := os.Lstat(filepath.Join(h.claudeLinks, "beta")); err == nil {
		t.Fatal("sync enabled beta")
	}
}

func TestRealDirectoryIsNotDeleted(t *testing.T) {
	h := newHarness(t)
	if err := os.MkdirAll(filepath.Join(h.links, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := h.ok("remove", "alpha")
	if fi, err := os.Lstat(filepath.Join(h.links, "alpha")); err != nil || !fi.IsDir() {
		t.Fatal("real directory deleted")
	}
	if !strings.Contains(out, "not deleting") {
		t.Fatalf("remove:\n%s", out)
	}
}

func TestSyncRepairsStaleLinksAndPrunesDanglingOnes(t *testing.T) {
	h := newHarness(t)
	old := filepath.Join(h.base, "old-checkout", "skills", "alpha")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{h.links, h.claudeLinks} {
		symlink(t, old, filepath.Join(dir, "alpha"))
		symlink(t, filepath.Join(h.repo, "skills", "gone"), filepath.Join(dir, "gone"))
		symlink(t, filepath.Join(h.base, "elsewhere"), filepath.Join(dir, "other"))
	}
	out := h.ok("sync")
	if !strings.Contains(out, "  removed "+filepath.Join(h.links, "gone")+" (no longer in the repo)\n") {
		t.Fatalf("sync:\n%s", out)
	}
	for _, dir := range []string{h.links, h.claudeLinks} {
		resolves(t, filepath.Join(dir, "alpha"), filepath.Join(h.repo, "skills", "alpha"))
		if isLink(filepath.Join(dir, "gone")) {
			t.Fatalf("%s/gone not pruned", dir)
		}
		if !isLink(filepath.Join(dir, "other")) {
			t.Fatalf("%s/other was removed", dir)
		}
		if isLink(filepath.Join(dir, "beta")) {
			t.Fatalf("%s/beta was enabled", dir)
		}
	}
}

func TestBadInputIsRejected(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{{"add", "nope"}, {"set", "alpha=maybe"}, {"set", "alpha"}, {"set", "=on"}, {}}
	wantErr := []string{"no such skill: nope", "bad state: maybe (valid: on off name-only user-invocable-only)", "expected NAME=STATE, got: alpha", "expected NAME=STATE, got: =on", "not a terminal - use add, remove, set or sync instead"}
	for i, args := range cases {
		_, errOut, code := h.run(args...)
		if code != 1 || !strings.Contains(errOut, "error: "+wantErr[i]) {
			t.Fatalf("%v: exit %d, stderr %q", args, code, errOut)
		}
	}
	if _, err := os.Lstat(h.links); err == nil {
		t.Fatal("a rejected command created the link directory")
	}
	if _, err := os.Lstat(h.settings); err == nil {
		t.Fatal("a rejected command wrote settings")
	}
}

func TestMissingRepoIsHelpful(t *testing.T) {
	h := newHarness(t)
	h.repo = filepath.Join(h.base, "nowhere")
	_, errOut, code := h.run("list")
	if code != 1 || !strings.Contains(errOut, "no skills/ directory in "+h.repo+" (set SKILLS_REPO or pass --repo)") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}

func TestRedirectedOutputHasNoANSI(t *testing.T) {
	h := newHarness(t)
	t.Setenv("NO_COLOR", "")
	os.Unsetenv("NO_COLOR")
	h.ok("set", "alpha=name-only")
	if out := h.ok("list"); strings.Contains(out, "\x1b") {
		t.Fatalf("escape codes in redirected output:\n%q", out)
	}
}

// ----- refresh -----

type refreshHarness struct {
	*harness
	source, target string
}

func newRefreshHarness(t *testing.T) *refreshHarness {
	t.Helper()
	h := &refreshHarness{harness: newHarness(t)}
	for _, name := range []string{"alpha", "beta"} {
		if err := os.RemoveAll(filepath.Join(h.repo, "skills", name)); err != nil {
			t.Fatal(err)
		}
	}
	h.source = filepath.Join(h.repo, "skills", "example")
	if err := os.MkdirAll(h.source, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(h.source, "SKILL.md"), "old instructions")
	h.git("init")
	h.git("add", ".")
	h.git("-c", "user.name=Test", "-c", "user.email=test@example.test", "-c", "core.hooksPath=/dev/null", "commit", "-m", "initial")
	write(filepath.Join(h.source, "SKILL.md"), "current instructions")
	write(filepath.Join(h.source, "asset.txt"), "current asset")
	h.target = filepath.Join(h.links, "example")
	if err := os.MkdirAll(h.target, 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(h.target, "SKILL.md"), "old instructions")
	write(filepath.Join(h.target, "local.txt"), "preserve this")
	return h
}

func (h *refreshHarness) git(args ...string) {
	h.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", h.repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		h.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (h *refreshHarness) backups() []string {
	h.t.Helper()
	matches, _ := filepath.Glob(filepath.Join(h.base, "home", ".agents", "skill-backups", "*", "example"))
	return matches
}

func TestRefreshHistoricalCopyBackupAndRepeat(t *testing.T) {
	h := newRefreshHarness(t)
	out, errOut, code := h.run("refresh")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	resolves(t, h.target, h.source)
	if b, _ := os.ReadFile(filepath.Join(h.target, "asset.txt")); string(b) != "current asset" {
		t.Fatal("link does not reach the current source")
	}
	backups := h.backups()
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	if b, _ := os.ReadFile(filepath.Join(backups[0], "local.txt")); string(b) != "preserve this" {
		t.Fatal("backup lost local file")
	}
	if !strings.Contains(out, "linked   "+h.target+" -> "+h.source+"\nbackup   "+backups[0]+"\n") {
		t.Fatalf("refresh:\n%s", out)
	}
	if _, _, code := h.run("refresh", "--check"); code != 0 {
		t.Fatalf("--check after refresh exited %d", code)
	}
	out, _, code = h.run("refresh")
	if code != 0 || !strings.Contains(out, "current  "+h.target+"\n") {
		t.Fatalf("repeat: exit %d\n%s", code, out)
	}
	if got := h.backups(); !reflect.DeepEqual(got, backups) {
		t.Fatalf("repeat made backups: %v", got)
	}
	if _, err := os.Lstat(filepath.Join(h.base, "codex", "skills", "example")); err == nil {
		t.Fatal("refresh created an installation")
	}
}

func TestRefreshCheckIsReadOnly(t *testing.T) {
	h := newRefreshHarness(t)
	out, _, code := h.run("refresh", "--check")
	if code != 1 || !strings.Contains(out, "refresh  "+h.target+"\n") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if isLink(h.target) {
		t.Fatal("--check changed files")
	}
	if _, err := os.Lstat(filepath.Join(h.base, "home", ".agents", "skill-backups")); err == nil {
		t.Fatal("--check made a backup")
	}
}

func TestRefreshUnrecognisedCopyAndBrokenLinkArePreserved(t *testing.T) {
	h := newRefreshHarness(t)
	if err := os.WriteFile(filepath.Join(h.target, "SKILL.md"), []byte("someone else owns this"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(h.base, "codex", "skills", "example")
	symlink(t, filepath.Join(h.base, "missing"), broken)
	_, errOut, code := h.run("refresh")
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	for _, target := range []string{h.target, broken} {
		if !strings.Contains(errOut, "review   "+target+": not a recognised copy; left untouched\n") {
			t.Fatalf("stderr:\n%s", errOut)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(h.target, "SKILL.md")); string(b) != "someone else owns this" {
		t.Fatal("unrecognised copy changed")
	}
	if dest, err := os.Readlink(broken); err != nil || dest != filepath.Join(h.base, "missing") {
		t.Fatalf("broken link changed: %s %v", dest, err)
	}
}

func TestRefreshMatchingCopyAndOutdatedLinkInCustomRoots(t *testing.T) {
	h := newRefreshHarness(t)
	codex := filepath.Join(h.base, "codex", "skills", "example")
	if err := os.MkdirAll(codex, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"SKILL.md", "asset.txt"} {
		b, _ := os.ReadFile(filepath.Join(h.source, f))
		if err := os.WriteFile(filepath.Join(codex, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := filepath.Join(h.base, "old-checkout", "example")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old instructions"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlink(t, old, filepath.Join(h.claudeLinks, "example"))
	out, errOut, code := h.run("refresh", "example")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	resolves(t, codex, h.source)
	resolves(t, filepath.Join(h.claudeLinks, "example"), h.source)
	resolves(t, h.target, h.source)
	_, errOut, code = h.run("refresh", "unknown", "example", "zzz")
	if code != 2 || errOut != "error: unknown skills: unknown, zzz\n" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}
