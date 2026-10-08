package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jorden-parker/skills/internal/links"
	"github.com/jorden-parker/skills/internal/settings"
	"github.com/jorden-parker/skills/internal/skills"
)

func (m *model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "enter", "y", "Y":
		if m.pending() == 0 {
			m.mode = modeGarden
			return m, nil
		}
		return m, m.save()
	case "d", "D":
		m.staged = append([]skills.Entry(nil), m.before...)
		m.mode = modeGarden
		m.setStatus("Staged changes discarded. Your garden can wait.", false)
		m.refreshPreview()
	case "esc", "n", "N", "q", "ctrl+c":
		m.mode = modeGarden
	}
	return m, nil
}

// save applies the staged states to the link directories and settings.json,
// then reloads so the list shows what actually happened.
func (m *model) save() tea.Cmd {
	var warnings []string
	after, err := links.Apply(m.repo, m.paths, m.staged, func(s string) { warnings = append(warnings, s) })
	if err != nil {
		m.mode = modeGarden
		m.setStatus("Save failed: "+err.Error(), true)
		return nil
	}
	ov := make(map[string]string, len(after))
	for _, e := range after {
		ov[e.Name] = string(e.State)
	}
	if err := settings.WriteOverrides(m.paths.Settings, ov); err != nil {
		m.mode = modeGarden
		m.reload()
		m.setStatus("Links updated but settings.json failed: "+err.Error(), true)
		return m.resetList()
	}
	m.mode = modeGarden
	m.reload()
	status := fmt.Sprintf("Saved to %s (.bak kept). Start a new Claude Code session for this to take effect.", tildify(m.paths.Settings))
	if len(warnings) > 0 {
		status = fmt.Sprintf("Saved, but skipped %d path(s) with a real directory or file: %s", len(warnings), strings.Join(warnings, "; "))
	}
	m.setStatus(status, len(warnings) > 0)
	return m.resetList()
}

func (m *model) reviewView() string {
	if m.pending() == 0 {
		return m.dialog("✿ Review & save", "Nothing is staged.", m.st.subtle.Render("Your garden is already just so. No changes to save."), "esc back")
	}
	var b strings.Builder
	pad := 0
	for i := range m.staged {
		if m.staged[i].State != m.before[i].State {
			pad = max(pad, len(m.staged[i].Name))
		}
	}
	for i := range m.staged {
		if m.staged[i].State == m.before[i].State {
			continue
		}
		fmt.Fprintf(&b, "  %-*s  %s %s %s\n", pad, m.staged[i].Name, m.st.chip(string(m.before[i].State)), m.st.subtle.Render("->"), m.st.chip(string(m.staged[i].State)))
	}
	b.WriteString("\n" + m.st.subtle.Render(fmt.Sprintf("Writes links in %s and %s,", tildify(m.paths.LinkDir), tildify(m.paths.ClaudeLinkDir))))
	b.WriteString("\n" + m.st.subtle.Render("and skillOverrides in "+tildify(m.paths.Settings)+"."))
	return m.dialog("✿ Ready to plant these changes?", fmt.Sprintf("%d change(s) staged", m.pending()), strings.TrimRight(b.String(), "\n"), "enter save · d discard staged · esc keep tending")
}
