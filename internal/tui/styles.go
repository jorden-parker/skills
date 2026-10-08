package tui

import (
	"image/color"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Garden palette: orchid accent, lilac card text, pink cursor. The dark
// values are the ones the Gum garden used; light terminals get deeper
// counterparts that stay readable on white.
type palette struct {
	ink, dim, faint, accent, card, cursor, ok, warn, err color.Color
}

func newPalette(isDark bool) palette {
	ld := lipgloss.LightDark(isDark)
	return palette{
		ink:    lipgloss.NoColor{},
		dim:    ld(lipgloss.Color("#62596E"), lipgloss.Color("#A99DB8")),
		faint:  ld(lipgloss.Color("#D8CEE3"), lipgloss.Color("#493A58")),
		accent: ld(lipgloss.Color("#6E3FB0"), lipgloss.Color("141")),
		card:   ld(lipgloss.Color("#4B3A6E"), lipgloss.Color("183")),
		cursor: ld(lipgloss.Color("#B5267A"), lipgloss.Color("212")),
		ok:     ld(lipgloss.Color("#14765E"), lipgloss.Color("#88DDC0")),
		warn:   ld(lipgloss.Color("#916000"), lipgloss.Color("#F2CE79")),
		err:    ld(lipgloss.Color("#B42318"), lipgloss.Color("#FF7070")),
	}
}

type styles struct {
	p                                              palette
	title, subtle, label, value, ok, warn, err     lipgloss.Style
	accent, card, cursor, faint, keyCap, selection lipgloss.Style
	chipOn, chipOff, chipPartial, badge            lipgloss.Style
	pane, paneFocus                                lipgloss.Style
}

func newStyles(isDark bool) styles {
	p := newPalette(isDark)
	ld := lipgloss.LightDark(isDark)
	background := ld(lipgloss.Color("#F3ECFA"), lipgloss.Color("#2C2238"))
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	pane := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.faint).Padding(0, 1)
	return styles{
		p:           p,
		title:       fg(p.accent).Bold(true),
		subtle:      fg(p.dim),
		label:       fg(p.ink).Bold(true),
		value:       fg(p.ink),
		ok:          fg(p.ok),
		warn:        fg(p.warn),
		err:         fg(p.err),
		accent:      fg(p.accent),
		card:        fg(p.card),
		cursor:      fg(p.cursor),
		faint:       fg(p.faint),
		keyCap:      fg(p.ink).Bold(true),
		selection:   fg(p.accent).Background(background).Bold(true),
		chipOn:      fg(p.ok),
		chipOff:     fg(p.dim),
		chipPartial: fg(p.warn),
		badge:       fg(p.accent).Background(background).Padding(0, 1).Bold(true),
		pane:        pane,
		paneFocus:   pane.BorderForeground(p.accent),
	}
}

// formTheme keeps the tune form in the garden's colours. huh only learns
// the background from a message the garden consumes, so the form is themed
// with the garden's own isDark instead.
func formTheme(isDark bool) huh.Theme {
	return huh.ThemeFunc(func(bool) *huh.Styles {
		p := newPalette(isDark)
		t := huh.ThemeCharm(isDark)
		for _, f := range []*huh.FieldStyles{&t.Focused, &t.Blurred} {
			f.Title = f.Title.Foreground(p.accent)
			f.Description = f.Description.Foreground(p.dim)
			f.SelectSelector = f.SelectSelector.Foreground(p.cursor)
			f.MultiSelectSelector = f.MultiSelectSelector.Foreground(p.cursor)
			f.NextIndicator = f.NextIndicator.Foreground(p.accent)
			f.PrevIndicator = f.PrevIndicator.Foreground(p.accent)
			f.SelectedOption = f.SelectedOption.Foreground(p.card)
			f.SelectedPrefix = f.SelectedPrefix.Foreground(p.card)
			f.Option = f.Option.Foreground(p.ink)
			f.UnselectedOption = f.UnselectedOption.Foreground(p.ink)
			f.ErrorIndicator = f.ErrorIndicator.Foreground(p.err)
			f.ErrorMessage = f.ErrorMessage.Foreground(p.err)
			f.FocusedButton = lipgloss.NewStyle().Padding(0, 2).MarginRight(1).Bold(true).Foreground(p.accent).Reverse(true)
			f.Next = f.FocusedButton
		}
		t.Focused.Base = t.Focused.Base.BorderForeground(p.accent)
		t.Focused.Card = t.Focused.Base
		t.Group.Title = t.Focused.Title
		t.Group.Description = t.Focused.Description
		return t
	})
}

// chip colours a state the way the list command does: on green, off dim,
// the partial states amber.
func (s styles) chip(st string) string {
	switch st {
	case "on":
		return s.chipOn.Render(st)
	case "off":
		return s.chipOff.Render(st)
	}
	return s.chipPartial.Render(st)
}
