// Package skills locates the checkout and reads the state of each skill.
package skills

import (
	"bufio"
	"fmt"
	"os"
	"path"
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

// Names lists the ID (author/name) of every skills/<author>/<name>/ that
// holds a SKILL.md, sorted.
func (r Repo) Names() ([]string, error) {
	authors, err := os.ReadDir(r.SrcDir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, author := range authors {
		skillDirs, err := os.ReadDir(filepath.Join(r.SrcDir, author.Name()))
		if err != nil {
			continue
		}
		for _, skillDir := range skillDirs {
			if id := author.Name() + "/" + skillDir.Name(); r.Has(id) {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// Has reports whether id is exactly author/name and names a skill in the repo.
func (r Repo) Has(id string) bool {
	author, name, ok := strings.Cut(id, "/")
	if !ok || !validSegment(author) || !validSegment(name) {
		return false
	}
	fi, err := os.Stat(filepath.Join(r.SrcDir, author, name, "SKILL.md"))
	return err == nil && fi.Mode().IsRegular()
}

func validSegment(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, `/\`)
}

// LinkName is the name a skill is linked under: the last segment of its ID.
func LinkName(id string) string { return path.Base(id) }

// Resolve turns a command-line argument into a skill ID: an exact ID, or a
// bare name that only one author uses.
func (r Repo) Resolve(arg string) (string, error) {
	if r.Has(arg) {
		return arg, nil
	}
	ids, err := r.Names()
	if err != nil {
		return "", err
	}
	var matches []string
	for _, id := range ids {
		if !strings.Contains(arg, "/") && LinkName(id) == arg {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no such skill: %s", arg)
	case 1:
		return matches[0], nil
	}
	return "", fmt.Errorf("ambiguous skill: %s (use one of: %s)", arg, strings.Join(matches, ", "))
}

// Source is the directory of the skill with this ID.
func (r Repo) Source(id string) string { return filepath.Join(r.SrcDir, filepath.FromSlash(id)) }

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
