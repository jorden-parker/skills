package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jorden-parker/skills/internal/settings"
)

// State is how visible a skill is: on, off, or one of the two partial states
// that keep the link and add a skillOverrides entry (docs/adr/0003).
type State string

const (
	On                State = "on"
	Off               State = "off"
	NameOnly          State = "name-only"
	UserInvocableOnly State = "user-invocable-only"
)

// States lists the valid states in display order.
var States = []State{On, Off, NameOnly, UserInvocableOnly}

// ParseState validates a state from the command line.
func ParseState(s string) (State, error) {
	for _, st := range States {
		if string(st) == s {
			return st, nil
		}
	}
	return "", fmt.Errorf("bad state: %s (valid: on off name-only user-invocable-only)", s)
}

// Linked reports whether the state keeps a link in place.
func (s State) Linked() bool { return s != Off }

// Paths are the directories and files the installer reads and writes.
type Paths struct {
	Home          string
	LinkDir       string // ~/.agents/skills
	ClaudeDir     string // CLAUDE_CONFIG_DIR or ~/.claude
	ClaudeLinkDir string // ClaudeDir/skills
	Settings      string // ClaudeDir/settings.json
	CodexDir      string // CODEX_HOME or ~/.codex
}

// DefaultPaths reads HOME, CLAUDE_CONFIG_DIR and CODEX_HOME.
func DefaultPaths() Paths {
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		claude = filepath.Join(home, ".claude")
	}
	codex := os.Getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(home, ".codex")
	}
	return Paths{
		Home:          home,
		LinkDir:       filepath.Join(home, ".agents", "skills"),
		ClaudeDir:     claude,
		ClaudeLinkDir: filepath.Join(claude, "skills"),
		Settings:      filepath.Join(claude, "settings.json"),
		CodexDir:      codex,
	}
}

// LinkDirs are the two discovery directories a skill is linked into.
func (p Paths) LinkDirs() []string { return []string{p.LinkDir, p.ClaudeLinkDir} }

// Entry is one skill and its state.
type Entry struct {
	Name  string
	State State
}

// LoadStates reads the state of every skill in the repo. A skill is linked
// when ~/.agents/skills/<name> exists in any form; a linked skill takes its
// override from settings.json, else on. Unlinked skills are off.
func LoadStates(r Repo, p Paths) ([]Entry, error) {
	names, err := r.Names()
	if err != nil {
		return nil, err
	}
	overrides := settings.ReadOverrides(p.Settings)
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		st := Off
		if lexists(filepath.Join(p.LinkDir, name)) {
			st = On
			if o, err := ParseState(overrides[name]); err == nil && o != Off {
				st = o
			}
		}
		out = append(out, Entry{Name: name, State: st})
	}
	return out, nil
}

// lexists reports whether a path exists, counting dangling symlinks.
func lexists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// Lexists is lexists for other packages.
func Lexists(path string) bool { return lexists(path) }
