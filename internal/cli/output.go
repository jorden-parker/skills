package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"

	"github.com/jorden-parker/skills/internal/skills"
)

// printer writes the command output, in colour only when it goes to a
// terminal that wants it: stdout itself, a TTY, NO_COLOR unset, TERM not
// dumb. Tests that redirect the command's output get plain text.
type printer struct {
	w      io.Writer
	colour bool
}

func newPrinter(w io.Writer) printer {
	p := printer{w: w}
	f, ok := w.(*os.File)
	if !ok || f != os.Stdout || !term.IsTerminal(f.Fd()) {
		return p
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return p
	}
	if t := os.Getenv("TERM"); t == "" || t == "dumb" {
		return p
	}
	p.colour = true
	return p
}

func (p printer) paint(code string, s string) string {
	if !p.colour {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func (p printer) printf(format string, args ...any) { fmt.Fprintf(p.w, format, args...) }
func (p printer) line(s string)                     { fmt.Fprintln(p.w, s) }
func (p printer) blank()                            { fmt.Fprintln(p.w) }
func (p printer) bold(s string)                     { p.line(p.paint("1", s)) }
func (p printer) dim(s string)                      { p.line(p.paint("2", s)) }
func (p printer) warn(s string)                     { p.line("  " + p.paint("33", "skip") + " " + s) }

func (p printer) state(st skills.State) string {
	code := "33"
	switch st {
	case skills.On:
		code = "32"
	case skills.Off:
		code = "90"
	}
	return p.paint(code, fmt.Sprintf("%-19s", st))
}

func (p printer) list(r skills.Repo, paths skills.Paths, entries []skills.Entry) {
	p.blank()
	p.bold("Skills in " + r.SrcDir)
	p.blank()
	for _, e := range entries {
		p.printf("  %s %-14s %s\n", p.state(e.State), e.Name, r.Describe(e.Name))
	}
	p.blank()
	p.dim("  on, name-only and user-invocable-only are linked into " + paths.LinkDir + " and " + paths.ClaudeLinkDir + "; off is not")
	p.blank()
}

func (p printer) changes(before, after []skills.Entry) {
	prev := map[string]skills.State{}
	for _, e := range before {
		prev[e.Name] = e.State
	}
	changed := false
	for _, e := range after {
		if prev[e.Name] == e.State {
			continue
		}
		if !changed {
			p.blank()
			p.bold("Changed")
			changed = true
		}
		p.printf("  %-14s %s -> %s\n", e.Name, p.state(prev[e.Name]), p.state(e.State))
	}
	if !changed {
		p.blank()
		p.dim("No states changed.")
	}
}

const sessionHint = "Start a new Claude Code session for this to take effect."
