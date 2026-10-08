package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/jorden-parker/skills/internal/skills"
)

// tuneOptions are the four states with the labels the Gum garden used.
var tuneOptions = []huh.Option[skills.State]{
	huh.NewOption("on — Fully visible", skills.On),
	huh.NewOption("off — Put this one to bed", skills.Off),
	huh.NewOption("name-only — A tiny context footprint", skills.NameOnly),
	huh.NewOption("user-invocable-only — Only when you call it", skills.UserInvocableOnly),
}

// startTune opens the state picker for the selected skill.
func (m *model) startTune() tea.Cmd {
	i := m.selectedIdx()
	if i < 0 {
		return nil
	}
	m.tuneIdx, m.tuneChoice = i, m.staged[i].State
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "back"))
	m.tune = huh.NewForm(huh.NewGroup(huh.NewSelect[skills.State]().
		Title(m.staged[i].Name + " · choose how it grows").
		Description("↑/↓ to browse · enter to choose").
		Options(tuneOptions...).Value(&m.tuneChoice))).
		WithTheme(formTheme(m.isDark)).WithKeyMap(km).WithShowHelp(false).
		WithWidth(max(1, m.dialogWidth()-4)).WithHeight(max(1, m.height-dialogChrome))
	m.mode = modeTune
	return m.tune.Init()
}

func (m *model) updateTune(msg tea.Msg) (tea.Model, tea.Cmd) {
	f, cmd := m.tune.Update(msg)
	if form, ok := f.(*huh.Form); ok {
		m.tune = form
	}
	switch m.tune.State {
	case huh.StateAborted:
		m.mode, m.tune = modeGarden, nil
		return m, nil
	case huh.StateCompleted:
		m.staged[m.tuneIdx].State = m.tuneChoice
		m.mode, m.tune = modeGarden, nil
		m.status = ""
		m.refreshPreview()
		return m, nil
	}
	return m, cmd
}
