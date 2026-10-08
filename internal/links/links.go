// Package links creates and removes the symlinks that install a skill.
// A real directory or file at a link's path is never replaced or deleted.
package links

import (
	"os"
	"path/filepath"
	"sort"
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
// A skill that could not be unlinked stays on.
func Apply(r skills.Repo, p skills.Paths, entries []skills.Entry, warn Warn) ([]skills.Entry, error) {
	out := make([]skills.Entry, len(entries))
	for i, e := range entries {
		out[i] = e
		if e.State == skills.Off {
			blocked, err := Unlink(e.Name, p.LinkDirs(), warn)
			if err != nil {
				return out, err
			}
			if blocked {
				out[i].State = skills.On
			}
			continue
		}
		if _, err := Link(r.Source(e.Name), e.Name, p.LinkDirs(), warn); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Prune removes dangling symlinks in dirs that point under srcDir and
// returns the paths it removed. Links elsewhere are left alone.
func Prune(srcDir string, dirs []string) ([]string, error) {
	var removed []string
	prefix := srcDir + string(filepath.Separator)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, n := range names {
			target := filepath.Join(dir, n)
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
			if err := os.Remove(target); err != nil {
				return removed, err
			}
			removed = append(removed, target)
		}
	}
	return removed, nil
}
