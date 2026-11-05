package skills

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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

type Scope string

const (
	GlobalScope  Scope = "global"
	ProjectScope Scope = "project"
)

func ParseScope(s string) (Scope, error) {
	switch Scope(s) {
	case GlobalScope, ProjectScope:
		return Scope(s), nil
	}
	return "", fmt.Errorf("bad scope: %s (valid: global project)", s)
}

// Paths are the directories and files the installer reads and writes.
type Paths struct {
	Scope         Scope
	Home          string
	LinkDir       string // ~/.agents/skills
	ClaudeDir     string // CLAUDE_CONFIG_DIR or ~/.claude
	ClaudeLinkDir string // ClaudeDir/skills
	Settings      string // ClaudeDir/settings.json
	CodexDir      string // CODEX_HOME or ~/.codex
	RegistryDir   string
	RegistryIndex string
}

func PathsFor(scope Scope, dir string) (Paths, error) {
	if scope != ProjectScope {
		return DefaultPaths(), nil
	}
	top, err := gitTopLevel(dir)
	if err != nil {
		return Paths{}, fmt.Errorf("project scope needs a Git repository: %s is not inside a Git repository", dir)
	}
	paths := DefaultPaths()
	paths.Scope = ProjectScope
	paths.LinkDir = filepath.Join(top, ".agents", "skills")
	paths.ClaudeLinkDir = filepath.Join(top, ".claude", "skills")
	paths.Settings = filepath.Join(top, ".claude", "settings.local.json")
	return paths, nil
}

func gitTopLevel(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
		Scope:         GlobalScope,
		Home:          home,
		LinkDir:       filepath.Join(home, ".agents", "skills"),
		ClaudeDir:     claude,
		ClaudeLinkDir: filepath.Join(claude, "skills"),
		Settings:      filepath.Join(claude, "settings.json"),
		CodexDir:      codex,
		RegistryDir:   filepath.Join(xdgDir("XDG_DATA_HOME", filepath.Join(home, ".local", "share")), "skills", "registry"),
		RegistryIndex: filepath.Join(xdgDir("XDG_CACHE_HOME", filepath.Join(home, ".cache")), "skills", "skills-sh-index.txt"),
	}
}

func xdgDir(envName, fallback string) string {
	if dir := os.Getenv(envName); dir != "" {
		return dir
	}
	return fallback
}

// LinkDirs are the two discovery directories a skill is linked into.
func (p Paths) LinkDirs() []string { return []string{p.LinkDir, p.ClaudeLinkDir} }

// Entry is one skill and its state.
type Entry struct {
	Name  string
	State State
}

// LoadStates reads the state of every skill in the repo. A skill is linked
// when ~/.agents/skills/<link name> exists in any form, unless it points at a
// same-name sibling. A linked skill takes its override from settings.json,
// else on. Unlinked skills are off.
func LoadStates(r Repo, p Paths) ([]Entry, error) {
	ids, err := installableIDs(r)
	if err != nil {
		return nil, err
	}
	overrides := settings.ReadOverrides(p.Settings)
	out := make([]Entry, 0, len(ids))
	for _, id := range ids {
		st := Off
		if linkedTo(r, p, ids, id) {
			st = On
			if o, err := ParseState(overrides[LinkName(id)]); err == nil && o != Off {
				st = o
			}
		}
		out = append(out, Entry{Name: id, State: st})
	}
	return out, nil
}

func installableIDs(r Repo) ([]string, error) {
	catalog, err := r.Names()
	if err != nil {
		return nil, err
	}
	downloaded, err := r.Downloaded()
	if err != nil {
		return nil, err
	}
	ids := append(catalog, downloaded...)
	sort.Strings(ids)
	return ids, nil
}

func linkedTo(r Repo, p Paths, ids []string, id string) bool {
	link := filepath.Join(p.LinkDir, LinkName(id))
	if !lexists(link) {
		return false
	}
	for _, sibling := range ids {
		if sibling != id && LinkName(sibling) == LinkName(id) && ResolvesTo(link, r.Source(sibling)) {
			return false
		}
	}
	return true
}

// ResolvesTo reports whether path resolves to the same directory as source.
func ResolvesTo(path, source string) bool {
	got, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	want, err := filepath.EvalSymlinks(source)
	return err == nil && got == want
}

func EnsureEntry(entries []Entry, id string) []Entry {
	for _, entry := range entries {
		if entry.Name == id {
			return entries
		}
	}
	return append(entries, Entry{Name: id, State: Off})
}

// Stage sets the state of the entry with this ID. A linked state turns off
// every other entry with the same link name, so one skill per name is on.
func Stage(entries []Entry, id string, st State) {
	for i := range entries {
		switch {
		case entries[i].Name == id:
			entries[i].State = st
		case st.Linked() && LinkName(entries[i].Name) == LinkName(id):
			entries[i].State = Off
		}
	}
}

// Overrides maps each link name to the state settings.json should hold. When
// siblings share a name, the linked one wins.
func Overrides(entries []Entry) map[string]string {
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		name := LinkName(e.Name)
		if _, seen := out[name]; !seen || e.State.Linked() {
			out[name] = string(e.State)
		}
	}
	return out
}

// lexists reports whether a path exists, counting dangling symlinks.
func lexists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// Lexists is lexists for other packages.
func Lexists(path string) bool { return lexists(path) }
