// Package skills locates the checkout and reads the state of each skill.
package skills

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
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
	Dir         string
	SrcDir      string
	OldSrcDir   string
	RegistryDir string
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
	r := Repo{
		Dir:         dir,
		SrcDir:      filepath.Join(dir, "catalog"),
		OldSrcDir:   filepath.Join(dir, "skills"),
		RegistryDir: DefaultPaths().RegistryDir,
	}
	if fi, err := os.Stat(r.SrcDir); err != nil || !fi.IsDir() {
		return r, fmt.Errorf("no catalog/ directory in %s (set SKILLS_REPO or pass --repo)", dir)
	}
	return r, nil
}

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
	if IsRegistryID(id) {
		return hasManifest(r.Source(id))
	}
	author, name, ok := strings.Cut(id, "/")
	if !ok || !validSegment(author) || !validSegment(name) {
		return false
	}
	return hasManifest(filepath.Join(r.SrcDir, author, name))
}

func hasManifest(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && fi.Mode().IsRegular()
}

func IsRegistryID(id string) bool {
	segments := strings.Split(id, "/")
	return len(segments) == 3 && validSegment(segments[0]) && validSegment(segments[1]) && validSegment(segments[2])
}

func (r Repo) Downloaded() ([]string, error) {
	owners, err := os.ReadDir(r.RegistryDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, owner := range owners {
		if !strings.HasPrefix(owner.Name(), ".") {
			ids = append(ids, r.downloadedUnder(owner.Name())...)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (r Repo) downloadedUnder(owner string) []string {
	var ids []string
	repos, _ := os.ReadDir(filepath.Join(r.RegistryDir, owner))
	for _, repoDir := range repos {
		if strings.HasPrefix(repoDir.Name(), ".") {
			continue
		}
		skillDirs, _ := os.ReadDir(filepath.Join(r.RegistryDir, owner, repoDir.Name()))
		for _, skillDir := range skillDirs {
			if strings.HasPrefix(skillDir.Name(), ".") {
				continue
			}
			if id := path.Join(owner, repoDir.Name(), skillDir.Name()); r.Has(id) {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func validSegment(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, `/\`)
}

// LinkName is the name a skill is linked under: the last segment of its ID.
func LinkName(id string) string { return path.Base(id) }

// Resolve turns a command-line argument into a skill ID: an exact ID, or a
// bare name that only one author uses.
func (r Repo) Resolve(arg string) (string, error) {
	if r.Has(arg) || IsRegistryID(arg) {
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
func (r Repo) Source(id string) string {
	if IsRegistryID(id) {
		return filepath.Join(r.RegistryDir, filepath.FromSlash(id))
	}
	return filepath.Join(r.SrcDir, filepath.FromSlash(id))
}

// Describe returns the first `description:` line of the skill's SKILL.md,
// cut to 90 characters.
func (r Repo) Describe(name string) string {
	f, err := os.Open(filepath.Join(r.Source(name), "SKILL.md"))
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
	b, err := os.ReadFile(filepath.Join(r.Source(name), "SKILL.md"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
