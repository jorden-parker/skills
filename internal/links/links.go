// Package links creates and removes the symlinks that install a skill.
// A real directory or file at a link's path is never replaced or deleted.
package links

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jorden-parker/skills/internal/skills"
)

// Warn receives one line per path the operation refused to touch.
type Warn func(line string)

// Link points dir/name at src in every dir. An existing symlink is replaced;
// anything else is reported through warn and blocks the result.
func Link(src, name string, dirs []string, warn Warn) (blocked bool, err error) {
	for _, dir := range dirs {
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return blocked, err
		}
		fi, lerr := os.Lstat(target)
		switch {
		case lerr == nil && fi.Mode()&os.ModeSymlink != 0:
			if err := os.Remove(target); err != nil {
				return blocked, err
			}
			if err := os.Symlink(src, target); err != nil {
				return blocked, err
			}
		case lerr == nil:
			warn(target + " - a real directory or file is already there, not touching it")
			blocked = true
		default:
			if err := os.Symlink(src, target); err != nil {
				return blocked, err
			}
		}
	}
	return blocked, nil
}

// Unlink removes dir/name from every dir when it is a symlink. Anything
// else is reported through warn and blocks the result.
func Unlink(name string, dirs []string, warn Warn) (blocked bool, err error) {
	for _, dir := range dirs {
		target := filepath.Join(dir, name)
		fi, lerr := os.Lstat(target)
		switch {
		case lerr == nil && fi.Mode()&os.ModeSymlink != 0:
			if err := os.Remove(target); err != nil {
				return blocked, err
			}
		case lerr == nil:
			warn(target + " - a real directory or file is there, not deleting it")
			blocked = true
		}
	}
	return blocked, nil
}

// Apply links or unlinks every entry and returns the states that resulted.
// Off entries go first, so a sibling's unlink never removes a new link. A
// skill that could not be unlinked stays on.
func Apply(r skills.Repo, p skills.Paths, entries []skills.Entry, warn Warn) ([]skills.Entry, error) {
	out := append([]skills.Entry(nil), entries...)
	for i, e := range entries {
		if e.State != skills.Off {
			continue
		}
		blocked, err := Unlink(skills.LinkName(e.Name), p.LinkDirs(), warn)
		if err != nil {
			return out, err
		}
		if blocked {
			out[i].State = skills.On
		}
	}
	for _, e := range entries {
		if e.State == skills.Off {
			continue
		}
		if _, err := Link(r.Source(e.Name), skills.LinkName(e.Name), p.LinkDirs(), warn); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Migrate relinks dangling links to a flat skills/<name> path, left from
// before skills moved into author folders, to the one skill with that name.
// It returns the paths it relinked; links to an ambiguous name are left alone.
func Migrate(r skills.Repo, dirs []string) ([]string, error) {
	ids, err := r.Names()
	if err != nil {
		return nil, err
	}
	var relinked []string
	for _, target := range danglingLinksInto(r.SrcDir, dirs) {
		id, ok := FlatLinkSkill(r, ids, target)
		if !ok {
			continue
		}
		if err := os.Remove(target); err != nil {
			return relinked, err
		}
		if err := os.Symlink(r.Source(id), target); err != nil {
			return relinked, err
		}
		relinked = append(relinked, target)
	}
	return relinked, nil
}

// FlatLinkSkill reports the skill a link to a flat skills/<name> path now
// belongs to, when exactly one of ids has that name.
func FlatLinkSkill(r skills.Repo, ids []string, link string) (string, bool) {
	dest, err := os.Readlink(link)
	if err != nil {
		return "", false
	}
	flatName, ok := strings.CutPrefix(dest, r.SrcDir+string(filepath.Separator))
	if !ok {
		return "", false
	}
	var matches []string
	for _, id := range ids {
		if skills.LinkName(id) == flatName {
			matches = append(matches, id)
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

// Prune removes dangling symlinks in dirs that point under srcDir and
// returns the paths it removed. Links elsewhere are left alone.
func Prune(srcDir string, dirs []string) ([]string, error) {
	var removed []string
	for _, target := range danglingLinksInto(srcDir, dirs) {
		if err := os.Remove(target); err != nil {
			return removed, err
		}
		removed = append(removed, target)
	}
	return removed, nil
}

// danglingLinksInto lists the symlinks in dirs that point under srcDir and
// resolve to nothing, sorted by name within each dir.
func danglingLinksInto(srcDir string, dirs []string) []string {
	var out []string
	prefix := srcDir + string(filepath.Separator)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			target := filepath.Join(dir, e.Name())
			fi, err := os.Lstat(target)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				continue
			}
			dest, err := os.Readlink(target)
			if err != nil || !strings.HasPrefix(dest, prefix) {
				continue
			}
			if _, err := os.Stat(target); err == nil {
				continue
			}
			out = append(out, target)
		}
	}
	return out
}
