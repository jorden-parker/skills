package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jorden-parker/skills/internal/refresh"
)

type refreshState struct {
	running bool
	check   bool     // the shown results are a plan, not what was done
	names   []string // scope; nil means every skill
	results []refresh.Result
	spin    spinner.Model
	offset  int
}

type refreshMsg struct {
	names   []string
	check   bool
	results []refresh.Result
}

// startRefresh opens the refresh screen and plans what a refresh would do.
func (m *model) startRefresh(names []string) tea.Cmd {
	m.mode = modeRefresh
	m.rf.spin = spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(m.st.accent))
	m.rf.offset = 0
	return tea.Batch(m.rf.spin.Tick, m.refreshCmd(names, true))
}

// refreshCmd runs refresh in the background.
func (m *model) refreshCmd(names []string, check bool) tea.Cmd {
	m.rf.running, m.rf.names, m.rf.check = true, names, check
	repo, paths := m.repo, m.paths
	return func() tea.Msg {
		results, _ := refresh.Run(repo, paths, names, check, nil)
		return refreshMsg{names: names, check: check, results: results}
	}
}

func (m *model) refreshDone(msg refreshMsg) tea.Cmd {
	m.rf.running = false
	m.rf.results = msg.results
	m.rf.check = msg.check
	if !msg.check {
		m.reload()
		return m.resetList()
	}
	return nil
}

func (m *model) refreshable() int {
	n := 0
	for _, r := range m.rf.results {
		if r.Status == refresh.Refresh {
			n++
		}
	}
	return n
}

func (m *model) updateRefresh(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(spinner.TickMsg); ok {
		if !m.rf.running {
			return m, nil
		}
		var cmd tea.Cmd
		m.rf.spin, cmd = m.rf.spin.Update(msg)
		return m, cmd
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || m.rf.running {
		return m, nil
	}
	switch k.String() {
	case "esc", "q", "ctrl+c":
		m.mode = modeRefresh
		m.mode = modeGarden
	case "enter":
		if m.rf.check && m.refreshable() > 0 {
			return m, tea.Batch(m.rf.spin.Tick, m.refreshCmd(m.rf.names, false))
		}
	case "f":
		if name := m.selectedName(); name != "" {
			return m, tea.Batch(m.rf.spin.Tick, m.refreshCmd([]string{name}, true))
		}
	case "A":
		return m, tea.Batch(m.rf.spin.Tick, m.refreshCmd(nil, true))
	case "down", "j":
		m.rf.offset++
	case "up", "k":
		m.rf.offset = max(0, m.rf.offset-1)
	}
	return m, nil
}

func (m *model) refreshHints() []hint {
	if m.rf.running {
		return []hint{{"esc", "back"}}
	}
	var h []hint
	if m.rf.check && m.refreshable() > 0 {
		h = append(h, hint{"enter", fmt.Sprintf("refresh %d", m.refreshable())})
	}
	if m.rf.names == nil {
		h = append(h, hint{"f", "only " + m.selectedName()})
	} else {
		h = append(h, hint{"A", "all skills"})
	}
	return append(h, hint{"↑ ↓", "scroll"}, hint{"esc", "back"})
}

func (m *model) refreshView() string {
	s := m.st
	scope := "every skill"
	if m.rf.names != nil {
		scope = strings.Join(m.rf.names, ", ")
	}
	var rows []string
	switch {
	case m.rf.running:
		rows = append(rows, m.rf.spin.View()+" "+s.subtle.Render("Looking through ~/.agents, Codex and Claude for installed copies…"))
	case len(m.rf.results) == 0:
		rows = append(rows, s.subtle.Render("No installed copies of "+scope+" found. Nothing to refresh."))
	}
	width := max(10, m.dialogWidth()-4)
	for _, r := range m.rf.results {
		var chip, note string
		switch r.Status {
		case refresh.Current:
			chip, note = s.ok.Render("current"), "already a link into this checkout"
		case refresh.Refresh:
			chip, note = s.warn.Render("refresh"), "a recognised copy; enter backs it up and links it"
		case refresh.Review:
			chip, note = s.err.Render("review "), "not a recognised copy; left untouched"
		case refresh.Linked:
			chip, note = s.ok.Render("linked "), "backup at "+tildify(r.Backup)
		case refresh.Error:
			chip, note = s.err.Render("error  "), r.Err.Error()
		}
		line := chip + " " + s.value.Render(short(tildify(r.Target), max(4, width-9)))
		rows = append(rows, line, "         "+s.subtle.Render(short(note, max(4, width-9))))
	}
	var tail string
	switch {
	case m.rf.running:
	case m.rf.check && m.refreshable() > 0:
		tail = s.accent.Render(fmt.Sprintf("Press enter to refresh %d installed cop(y/ies). Backups go under skill-backups beside each skills directory.", m.refreshable()))
	case m.rf.check:
		tail = s.subtle.Render("Nothing to refresh. Rows marked review need a look by hand.")
	default:
		tail = s.ok.Render("Done. Start a new agent session if it already loaded the old copy.")
	}
	if tail != "" {
		rows = append(rows, "", lipgloss.NewStyle().Width(width).Render(tail))
	}
	h := max(1, m.height-dialogChrome)
	start := min(m.rf.offset, max(0, len(rows)-h))
	content := strings.Join(rows[start:min(len(rows), start+h)], "\n")
	help := "enter refresh · f only the selected skill · esc back"
	if m.rf.names != nil {
		help = "enter refresh · A all skills · esc back"
	}
	return m.dialog("✿ Refresh installed copies", "Checking "+scope+" in ~/.agents/skills, Codex and Claude", content, help)
}
