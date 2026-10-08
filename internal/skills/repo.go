// Package skills locates the checkout and reads the state of each skill.
package skills

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultDir is the checkout path baked in at build time with
// -ldflags "-X github.com/jorden-parker/skills/internal/skills.DefaultDir=…".
var DefaultDir string

// Repo is a checkout of this repository.
type Repo struct {
	Dir    string // the checkout
	SrcDir string // Dir/skills, the source of every skill
}

// Find resolves the checkout: the --repo flag, then SKILLS_REPO, then the
// baked-in default, then ~/src/skills.
func Find(flag string) (Repo, error) {
	dir := flag
	if dir == "" {
		dir = os.Getenv("SKILLS_REPO")
	}
	if dir == "" {
		dir = DefaultDir
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "src", "skills")
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	r := Repo{Dir: dir, SrcDir: filepath.Join(dir, "skills")}
	if fi, err := os.Stat(r.SrcDir); err != nil || !fi.IsDir() {
		return r, fmt.Errorf("no skills/ directory in %s (set SKILLS_REPO or pass --repo)", dir)
	}
	return r, nil
}

// Names lists every directory under skills/ that holds a SKILL.md, sorted.
func (r Repo) Names() ([]string, error) {
	entries, err := os.ReadDir(r.SrcDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if fi, err := os.Stat(filepath.Join(r.SrcDir, e.Name(), "SKILL.md")); err == nil && fi.Mode().IsRegular() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Has reports whether name is a skill in the repo.
func (r Repo) Has(name string) bool {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return false
	}
	fi, err := os.Stat(filepath.Join(r.SrcDir, name, "SKILL.md"))
	return err == nil && fi.Mode().IsRegular()
}

// Source is the directory of the named skill.
func (r Repo) Source(name string) string { return filepath.Join(r.SrcDir, name) }

// Describe returns the first `description:` line of the skill's SKILL.md,
// cut to 90 characters.
func (r Repo) Describe(name string) string {
	f, err := os.Open(filepath.Join(r.SrcDir, name, "SKILL.md"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(sc.Text(), "description:"); ok {
			rest = strings.TrimLeft(rest, " ")
			if rs := []rune(rest); len(rs) > 90 {
				rest = string(rs[:90])
			}
			return rest
		}
	}
	return ""
}

// Manifest returns the SKILL.md contents for the preview pane.
func (r Repo) Manifest(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(r.SrcDir, name, "SKILL.md"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
