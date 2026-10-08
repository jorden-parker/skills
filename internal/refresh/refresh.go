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
// checkout knows: the current bytes and every version in Git history.
func KnownVersions(repoDir, rel string) map[string]struct{} {
	versions := map[string]struct{}{}
	if b, err := os.ReadFile(filepath.Join(repoDir, rel)); err == nil {
		versions[string(b)] = struct{}{}
	}
	rel = filepath.ToSlash(rel)
	log, err := exec.Command("git", "-C", repoDir, "log", "--all", "--format=%H", "--", rel).Output()
	if err != nil {
		return versions
	}
	for _, commit := range strings.Fields(string(log)) {
		if b, err := exec.Command("git", "-C", repoDir, "show", commit+":"+rel).Output(); err == nil {
			versions[string(b)] = struct{}{}
		}
	}
	return versions
}

// Unknown returns the names that are not skills in the repo, sorted.
func Unknown(r skills.Repo, names []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		if !r.Has(n) && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// Run inspects every root for each named skill (all repo skills when names
// is empty). With check set nothing is changed. failed is true when any
// result needs review, hit an error, or (with check) would be refreshed.
func Run(r skills.Repo, p skills.Paths, names []string, check bool, now func() time.Time) (results []Result, failed bool) {
	if len(names) == 0 {
		names, _ = r.Names()
	} else {
		names = append([]string(nil), names...)
		sort.Strings(names)
	}
	if now == nil {
		now = time.Now
	}
	for _, name := range names {
		source := r.Source(name)
		var versions map[string]struct{}
		for _, root := range Roots(p) {
			target := filepath.Join(root, name)
			if !skills.Lexists(target) {
				continue
			}
			res := Result{Name: name, Target: target}
			if isLinkTo(target, source) {
				res.Status = Current
				results = append(results, res)
				continue
			}
			if versions == nil {
				versions = KnownVersions(r.Dir, filepath.Join("skills", name, "SKILL.md"))
			}
			manifest, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
			if _, known := versions[string(manifest)]; err != nil || !known || !isRegular(filepath.Join(target, "SKILL.md")) {
				res.Status = Review
				results = append(results, res)
				failed = true
				continue
			}
			if check {
				res.Status = Refresh
				results = append(results, res)
				failed = true
				continue
			}
			backup, err := relink(root, name, target, source, now)
			if err != nil {
				res.Status, res.Err = Error, err
				failed = true
			} else {
				res.Status, res.Source, res.Backup = Linked, source, backup
			}
			results = append(results, res)
		}
	}
	return results, failed
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
