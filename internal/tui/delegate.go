package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// item is one skill row. The states live on the model so staging a change
// never rebuilds the list.
type item struct {
	idx  int
	name string
	desc string
}

func (i item) FilterValue() string { return i.name + " " + i.desc }

// rowDelegate draws the marker, name, state chip and pending flag above the
// description.
type rowDelegate struct{ m *model }

func (rowDelegate) Height() int                         { return 2 }
func (rowDelegate) Spacing() int                        { return 0 }
func (rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, lm list.Model, index int, li list.Item) {
	it, ok := li.(item)
	if !ok {
		return
	}
	s := d.m.st
	width := lm.Width()
	state := d.m.staged[it.idx].State
	before := d.m.before[it.idx].State
	selected := index == lm.Index()

	gutter := "  "
	nameStyle := s.value
	if selected {
		gutter = s.cursor.Render("❯ ")
		nameStyle = s.selection
	}
	marker := s.subtle.Render("· ")
	if state.Linked() {
		marker = s.accent.Render("✿ ")
	}
	right := s.chip(string(state))
	if state != before {
		right += " " + s.cursor.Render("*")
	} else {
		right += "  "
	}
	room := max(4, width-4-lipgloss.Width(right)-1)
	name := ansi.Truncate(it.name, room, "…")
	nameView := nameStyle.Render(name)
	var idx []int
	for _, i := range lm.MatchesForItem(index) {
		if i >= 0 && i < len([]rune(name)) {
			idx = append(idx, i)
		}
	}
	if len(idx) > 0 {
		nameView = lipgloss.StyleRunes(name, idx, nameStyle.Underline(true).Foreground(s.p.accent), nameStyle)
	}
	gap := max(1, width-4-lipgloss.Width(nameView)-lipgloss.Width(right))
	fmt.Fprint(w, gutter+marker+nameView+fmt.Sprintf("%*s", gap, "")+right)
	desc := strings.Join(strings.Fields(it.desc), " ")
	fmt.Fprint(w, "\n    "+s.subtle.Render(ansi.Truncate(desc, max(1, width-4), "…")))
}
