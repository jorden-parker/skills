// Package refresh replaces installed copies of a skill with links to this
// checkout, keeping a backup of each copy it moves aside.
package refresh

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jorden-parker/skills/internal/links"
	"github.com/jorden-parker/skills/internal/skills"
)

// Status is what refresh found or did at one installed path.
type Status string

const (
	Current Status = "current" // already a link into this checkout
	Refresh Status = "refresh" // a recognised copy that --check would replace
	Review  Status = "review"  // not a recognised copy; left untouched
	Linked  Status = "linked"  // replaced with a link
	Error   Status = "error"   // something went wrong at this path
)

// Result is one line of refresh output.
type Result struct {
	Name   string
	Target string
	Source string // set when Linked
	Backup string // set when Linked
	Status Status
	Err    error // set when Error
}

// Failed reports whether the result counts towards a non-zero exit.
func (r Result) Failed() bool { return r.Status == Review || r.Status == Error || r.Status == Refresh }

// Roots are the installation directories refresh inspects, deduplicated.
func Roots(p skills.Paths) []string {
	var out []string
	seen := map[string]bool{}
	for _, dir := range []string{p.LinkDir, filepath.Join(p.CodexDir, "skills"), p.ClaudeLinkDir} {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
		if !seen[dir] {
			seen[dir] = true
			out = append(out, dir)
		}
	}
	return out
}

// KnownVersions returns every version of rel (a path inside repoDir) this
// checkout knows: the current bytes and every version in Git history,
// following the file back through renames.
func KnownVersions(repoDir, rel string) map[string]struct{} {
	versions := map[string]struct{}{}
	if b, err := os.ReadFile(filepath.Join(repoDir, rel)); err == nil {
		versions[string(b)] = struct{}{}
	}
	log, err := exec.Command("git", "-C", repoDir, "log", "--all", "--follow", "--name-only", "--format=commit %H", "--", filepath.ToSlash(rel)).Output()
	if err != nil {
		return versions
	}
	var commit string
	for _, line := range strings.Split(string(log), "\n") {
		if hash, ok := strings.CutPrefix(line, "commit "); ok {
			commit = hash
			continue
		}
		if line == "" || commit == "" {
			continue
		}
		if b, err := exec.Command("git", "-C", repoDir, "show", commit+":"+line).Output(); err == nil {
			versions[string(b)] = struct{}{}
		}
	}
	return versions
}

// Run inspects every root for the link name of each given skill ID (all
// repo skills when ids is empty). Every skill with that link name is a
// candidate source. With check set nothing is changed. failed is true when
// any result needs review, hit an error, or (with check) would be refreshed.
func Run(r skills.Repo, p skills.Paths, ids []string, check bool, now func() time.Time) (results []Result, failed bool) {
	all, _ := r.Names()
	if len(ids) == 0 {
		ids = all
	}
	if now == nil {
		now = time.Now
	}
	for _, name := range linkNames(ids) {
		inspector := &inspector{repo: r, ids: all, candidates: candidates(all, name)}
		for _, root := range Roots(p) {
			target := filepath.Join(root, name)
			if !skills.Lexists(target) {
				continue
			}
			res := inspector.inspect(root, name, target, check, now)
			failed = failed || res.Failed()
			results = append(results, res)
		}
	}
	return results, failed
}

func linkNames(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if name := skills.LinkName(id); !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func candidates(ids []string, name string) []string {
	var out []string
	for _, id := range ids {
		if skills.LinkName(id) == name {
			out = append(out, id)
		}
	}
	return out
}

// inspector checks the installed copies of one link name, loading each
// candidate's known versions at most once.
type inspector struct {
	repo       skills.Repo
	ids        []string
	candidates []string
	versions   map[string]map[string]struct{}
}

func (in *inspector) inspect(root, name, target string, check bool, now func() time.Time) Result {
	res := Result{Name: name, Target: target}
	for _, id := range in.candidates {
		if isLinkTo(target, in.repo.Source(id)) {
			res.Status = Current
			return res
		}
	}
	id, ok := in.recognise(target)
	if !ok {
		res.Status = Review
		return res
	}
	if check {
		res.Status = Refresh
		return res
	}
	source := in.repo.Source(id)
	backup, err := relink(root, name, target, source, now)
	if err != nil {
		res.Status, res.Err = Error, err
		return res
	}
	res.Status, res.Source, res.Backup = Linked, source, backup
	return res
}

// recognise returns the candidate an installed path belongs to: the skill a
// dangling flat link now lives at, or the first whose known versions include
// the copy's SKILL.md.
func (in *inspector) recognise(target string) (string, bool) {
	if id, ok := links.FlatLinkSkill(in.repo, in.ids, target); ok && !isRegular(filepath.Join(target, "SKILL.md")) {
		return id, true
	}
	manifestPath := filepath.Join(target, "SKILL.md")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil || !isRegular(manifestPath) {
		return "", false
	}
	for _, id := range in.candidates {
		if _, known := in.knownVersions(id)[string(manifest)]; known {
			return id, true
		}
	}
	return "", false
}

func (in *inspector) knownVersions(id string) map[string]struct{} {
	if in.versions == nil {
		in.versions = map[string]map[string]struct{}{}
	}
	if v, ok := in.versions[id]; ok {
		return v
	}
	v := KnownVersions(in.repo.Dir, filepath.Join("skills", filepath.FromSlash(id), "SKILL.md"))
	in.versions[id] = v
	return v
}

// relink moves target to a fresh backup directory, links it to source and
// verifies the link. On failure the backup is moved back.
func relink(root, name, target, source string, now func() time.Time) (string, error) {
	var random [4]byte
	_, _ = rand.Read(random[:])
	t := now()
	stamp := t.Format("20060102-150405") + fmt.Sprintf("-%06d-", t.Nanosecond()/1000) + hex.EncodeToString(random[:])
	backup := filepath.Join(filepath.Dir(root), "skill-backups", stamp, name)
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(target, backup); err != nil {
		return "", err
	}
	if err := os.Symlink(source, target); err != nil {
		_ = os.Rename(backup, target)
		return "", err
	}
	if !isLinkTo(target, source) || !isRegular(filepath.Join(target, "SKILL.md")) {
		return "", errors.New("link verification failed")
	}
	return backup, nil
}

func isLinkTo(target, source string) bool {
	fi, err := os.Lstat(target)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	got, err := filepath.EvalSymlinks(target)
	if err != nil {
		return false
	}
	want, err := filepath.EvalSymlinks(source)
	return err == nil && got == want
}

func isRegular(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
