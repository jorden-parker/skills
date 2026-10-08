// Package tui is the skill garden: an interactive picker built on Bubble
// Tea, Bubbles, Lip Gloss and Huh. Changes are staged until you save.
package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jorden-parker/skills/internal/skills"
)

type mode int

const (
	modeGarden mode = iota
	modeTune
	modeReview
	modeRefresh
	modeConfirmQuit
	modeHelp
)

type keymap struct {
	toggle, tune, allOn, allOff, review, refresh, reload, help, quit, details, esc key.Binding
}

var keys = keymap{
	toggle:  key.NewBinding(key.WithKeys("space", "tab"), key.WithHelp("space", "toggle")),
	tune:    key.NewBinding(key.WithKeys("enter", "t"), key.WithHelp("enter", "tune")),
	allOn:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all on")),
	allOff:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "all off")),
	review:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "review & save")),
	refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	reload:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reload")),
	help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
	quit:    key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	details: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "preview")),
	esc:     key.NewBinding(key.WithKeys("esc")),
}

// Rows outside the panes: three header lines, the status line, the footer.
const chromeRows = 5

// dialogChrome is the rows a dialog uses around an embedded form.
const dialogChrome = 10

type model struct {
	repo   skills.Repo
	paths  skills.Paths
	before []skills.Entry // on disk at the last load
	staged []skills.Entry // what save will apply; same order as before
	mode   mode
	isDark bool
	st     styles

	width, height int
	list          list.Model
	preview       viewport.Model
	details       bool // the preview has the whole width (narrow terminals)
	status        string
	statusErr     bool
	loadErr       error

	tune       *huh.Form
	tuneIdx    int
	tuneChoice skills.State
	rf         refreshState
	helpOffset int
}

// Run opens the garden.
func Run(r skills.Repo, p skills.Paths) error {
	m := newModel(r, p)
	_, err := tea.NewProgram(m).Run()
	return err
}

func newModel(r skills.Repo, p skills.Paths) *model {
	m := &model{repo: r, paths: p, isDark: true}
	m.st = newStyles(true)
	m.reload()
	m.list = list.New(m.items(), rowDelegate{m}, 40, 20)
	m.list.SetShowHelp(false)
	m.list.SetShowStatusBar(false)
	m.list.SetShowPagination(false)
	m.list.SetStatusBarItemName("skill", "skills")
	m.list.KeyMap.Quit.SetEnabled(false)
	m.list.KeyMap.ShowFullHelp.SetEnabled(false)
	m.list.KeyMap.CloseFullHelp.SetEnabled(false)
	m.list.FilterInput.Prompt = "/ "
	m.applyStyles()
	m.preview = viewport.New()
	m.refreshPreview()
	return m
}

func (m *model) applyStyles() {
	m.list.SetDelegate(rowDelegate{m})
	ls := list.DefaultStyles(m.isDark)
	ls.TitleBar = lipgloss.NewStyle()
	ls.Title = lipgloss.NewStyle()
	ls.NoItems = m.st.subtle.PaddingLeft(2)
	ls.Filter.Focused.Prompt = m.st.accent.Bold(true)
	ls.Filter.Blurred.Prompt = m.st.subtle
	ls.Filter.Cursor.Color = m.st.p.accent
	m.list.Styles = ls
	m.list.FilterInput.SetStyles(ls.Filter)
	m.list.Title = m.st.subtle.Render(fmt.Sprintf("%d skills · / to filter", len(m.staged)))
}

// reload reads the states from disk and drops anything staged.
func (m *model) reload() {
	entries, err := skills.LoadStates(m.repo, m.paths)
	m.loadErr = err
	m.before = entries
	m.staged = append([]skills.Entry(nil), entries...)
}

func (m *model) items() []list.Item {
	out := make([]list.Item, len(m.staged))
	for i, e := range m.staged {
		out[i] = item{idx: i, name: e.Name, desc: m.repo.Describe(e.Name)}
	}
	return out
}

// resetList rebuilds the rows after a reload, keeping the cursor.
func (m *model) resetList() tea.Cmd {
	idx := m.list.Index()
	cmd := m.list.SetItems(m.items())
	m.list.Select(min(idx, max(0, len(m.staged)-1)))
	m.refreshPreview()
	return paneCommand(cmd)
}

func (m *model) selectedIdx() int {
	if it, ok := m.list.SelectedItem().(item); ok {
		return it.idx
	}
	return -1
}

func (m *model) selectedName() string {
	if i := m.selectedIdx(); i >= 0 {
		return m.staged[i].Name
	}
	return ""
}

// pending counts rows whose staged state differs from the one on disk.
func (m *model) pending() int {
	n := 0
	for i := range m.staged {
		if m.staged[i].State != m.before[i].State {
			n++
		}
	}
	return n
}

func (m *model) growing() int {
	n := 0
	for _, e := range m.staged {
		if e.State.Linked() {
			n++
		}
	}
	return n
}

func (m *model) setStatus(s string, isErr bool) { m.status, m.statusErr = s, isErr }

func (m *model) Init() tea.Cmd { return tea.RequestBackgroundColor }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case paneMsg:
		updated, cmd := m.list.Update(msg.msg)
		m.list = updated
		m.refreshPreview()
		return m, paneCommand(cmd)
	case refreshMsg:
		return m, m.refreshDone(msg)
	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		m.st = newStyles(m.isDark)
		m.applyStyles()
		m.refreshPreview()
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		if m.tune != nil {
			m.tune = m.tune.WithWidth(max(1, m.dialogWidth()-4)).WithHeight(max(1, m.height-dialogChrome))
		}
		return m, nil
	}

	switch m.mode {
	case modeTune:
		return m.updateTune(msg)
	case modeReview:
		return m.updateReview(msg)
	case modeRefresh:
		return m.updateRefresh(msg)
	case modeHelp:
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch k.String() {
			case "down", "j", "pgdown":
				m.helpOffset++
			case "up", "k", "pgup":
				m.helpOffset = max(0, m.helpOffset-1)
			default:
				m.mode = modeGarden
			}
		}
		return m, nil
	case modeConfirmQuit:
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch k.String() {
			case "y", "Y", "enter", "q", "ctrl+c":
				return m, tea.Quit
			}
			m.mode = modeGarden
			m.setStatus("Still tending. Press s to review and save.", false)
		}
		return m, nil
	}
	return m.updateGarden(msg)
}

func (m *model) updateGarden(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, isKey := msg.(tea.KeyPressMsg)
	if isKey && !m.list.SettingFilter() {
		switch {
		case key.Matches(k, keys.details):
			m.details = !m.details
			m.layout()
			return m, nil
		case key.Matches(k, keys.esc) && m.details:
			m.details = false
			m.layout()
			return m, nil
		case key.Matches(k, keys.esc) && m.list.IsFiltered():
			// The list clears its own filter.
		case key.Matches(k, keys.quit):
			if m.pending() > 0 {
				m.mode = modeConfirmQuit
				return m, nil
			}
			return m, tea.Quit
		case key.Matches(k, keys.help):
			m.mode, m.helpOffset = modeHelp, 0
			return m, nil
		case key.Matches(k, keys.toggle):
			m.toggle(m.selectedIdx())
			return m, nil
		case key.Matches(k, keys.tune):
			return m, m.startTune()
		case key.Matches(k, keys.allOn):
			for i := range m.staged {
				if m.staged[i].State == skills.Off {
					m.staged[i].State = skills.On
				}
			}
			m.setStatus("Every skill staged on. Partial states kept.", false)
			return m, nil
		case key.Matches(k, keys.allOff):
			for i := range m.staged {
				m.staged[i].State = skills.Off
			}
			m.setStatus("Every skill staged off. Nothing is saved until you press s.", false)
			return m, nil
		case key.Matches(k, keys.review):
			m.mode = modeReview
			return m, nil
		case key.Matches(k, keys.refresh):
			return m, m.startRefresh(nil)
		case key.Matches(k, keys.reload):
			m.reload()
			m.setStatus("Reloaded from disk. Staged changes dropped.", false)
			return m, m.resetList()
		case k.String() == "pgdown":
			m.preview.HalfPageDown()
			return m, nil
		case k.String() == "pgup":
			m.preview.HalfPageUp()
			return m, nil
		}
	}
	before := m.selectedIdx()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.selectedIdx() != before {
		m.refreshPreview()
	}
	return m, paneCommand(cmd)
}

// toggle turns an off skill on and anything else off.
func (m *model) toggle(i int) {
	if i < 0 {
		return
	}
	if m.staged[i].State == skills.Off {
		m.staged[i].State = skills.On
	} else {
		m.staged[i].State = skills.Off
	}
	m.status = ""
}

type paneMsg struct{ msg tea.Msg }

// paneCommand wraps a list command so its message goes back to the list.
func paneCommand(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			commands := make([]tea.Cmd, len(batch))
			for i, c := range batch {
				commands[i] = paneCommand(c)
			}
			return tea.BatchMsg(commands)
		}
		return paneMsg{msg: msg}
	}
}

// ----- layout and views -----

func (m *model) listWidth() int {
	switch {
	case m.width < 60:
		return m.width
	case m.width < 100:
		return m.width / 2
	}
	return min(56, m.width/2)
}

func (m *model) layout() {
	if m.width == 0 {
		return
	}
	lw := m.listWidth()
	inner := max(1, m.height-chromeRows-2)
	m.list.SetSize(lw-4, inner)
	pw := m.width - lw - 4
	if m.width < 60 {
		pw = m.width - 4
	}
	m.preview.SetWidth(max(1, pw))
	m.preview.SetHeight(inner)
	m.refreshPreview()
}

func (m *model) refreshPreview() {
	i := m.selectedIdx()
	w := m.preview.Width()
	if w < 20 {
		w = 60
	}
	if i < 0 {
		m.preview.SetContent(m.st.subtle.Width(w).Render("No skill matches the filter. Press esc to clear it."))
		return
	}
	e := m.staged[i]
	wrap := lipgloss.NewStyle().Width(w)
	head := m.st.title.Render(e.Name) + "  " + m.st.chip(string(e.State))
	if e.State != m.before[i].State {
		head += m.st.subtle.Render("  was "+string(m.before[i].State)) + " " + m.st.cursor.Render("*")
	}
	body, err := m.repo.Manifest(e.Name)
	if err != nil {
		body = m.st.err.Render(err.Error())
	}
	m.preview.SetContent(wrap.Render(head) + "\n" + wrap.Render(m.st.subtle.Render(tildify(m.repo.Source(e.Name)))) + "\n\n" + wrap.Render(body))
	m.preview.GotoTop()
}

func (m *model) dialogWidth() int { return max(1, min(88, m.width-4)) }

func (m *model) dialog(title, subtitle, content, help string) string {
	w := m.dialogWidth()
	inner := max(1, w-4)
	head := m.st.title.Render(title) + "\n" + m.st.subtle.Render(short(subtitle, inner))
	card := m.st.paneFocus.Width(w).Render(head + "\n\n" + content)
	body := card + "\n" + m.st.subtle.Render(short(help, w))
	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(
		lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body))
}

func (m *model) header() string {
	brand := m.st.badge.Render("✿ skills") + " " + m.st.title.Render("The skill garden")
	count := m.st.subtle.Render(fmt.Sprintf("%d of %d growing · %d pending", m.growing(), len(m.staged), m.pending()))
	line1 := brand + strings.Repeat(" ", max(1, m.width-lipgloss.Width(brand)-lipgloss.Width(count)-1)) + count
	line2 := " " + m.st.subtle.Render(tildify(m.repo.SrcDir))
	return short(line1, m.width) + "\n" + short(line2, m.width) + "\n" + m.st.faint.Render(strings.Repeat("─", max(1, m.width)))
}

func (m *model) statusLine() string {
	var status string
	switch {
	case m.mode == modeConfirmQuit:
		status = m.st.warn.Render(fmt.Sprintf("%d pending change(s) will be lost. Quit without saving? y / n", m.pending()))
	case m.loadErr != nil:
		status = m.st.err.Render("✕ " + m.loadErr.Error())
	case m.status != "" && m.statusErr:
		status = m.st.err.Render("✕ " + m.status)
	case m.status != "":
		status = m.st.ok.Render("✓ " + m.status)
	case m.pending() > 0:
		status = m.st.subtle.Render("Press s to review and save your changes.")
	default:
		status = m.st.subtle.Render("Space toggles a skill, enter tunes it. A little tending. A cleverer assistant.")
	}
	return " " + status
}

type hint struct{ key, desc string }

func (m *model) hints() []hint {
	switch {
	case m.mode == modeHelp:
		return []hint{{"esc", "close"}, {"↑ ↓", "scroll"}}
	case m.mode == modeConfirmQuit:
		return []hint{{"y", "quit"}, {"n", "keep tending"}}
	case m.mode == modeReview:
		return []hint{{"enter", "save"}, {"d", "discard staged"}, {"esc", "back"}}
	case m.mode == modeRefresh:
		return m.refreshHints()
	case m.list.SettingFilter():
		return []hint{{"enter", "apply filter"}, {"esc", "cancel"}}
	case m.details:
		return []hint{{"esc", "back"}, {"pgup pgdn", "scroll"}}
	}
	h := []hint{{"space", "toggle"}, {"enter", "tune"}, {"s", "review & save"}, {"r", "refresh"}, {"/", "filter"}}
	if m.width < 60 {
		h = append(h, hint{"ctrl+d", "preview"})
	}
	return append(h, hint{"a", "all on"}, hint{"n", "all off"}, hint{"R", "reload"})
}

func (m *model) footer() string {
	render := func(h hint) string { return m.st.keyCap.Render(h.key) + " " + m.st.subtle.Render(h.desc) }
	tail := []hint{{"?", "keys"}, {"q", "quit"}}
	if m.mode != modeGarden {
		tail = nil
	}
	var tailParts []string
	for _, h := range tail {
		tailParts = append(tailParts, render(h))
	}
	tailView := strings.Join(tailParts, "   ")
	var parts []string
	used := lipgloss.Width(tailView)
	for _, h := range m.hints() {
		r := render(h)
		if used+lipgloss.Width(r)+3 > m.width-1 {
			break
		}
		parts = append(parts, r)
		used += lipgloss.Width(r) + 3
	}
	left := " " + strings.Join(parts, "   ")
	if tailView == "" {
		return left
	}
	return left + strings.Repeat(" ", max(3, m.width-lipgloss.Width(left)-lipgloss.Width(tailView)-1)) + tailView
}

func (m *model) keysOverlay() string {
	s := m.st
	groups := []struct {
		title string
		rows  []hint
	}{
		{"Move", []hint{{"↑ ↓  j k", "move between skills"}, {"/", "filter by name or description"}, {"pgup pgdn", "scroll the SKILL.md preview"}, {"ctrl+d", "show only the preview (narrow terminals)"}}},
		{"Stage", []hint{{"space tab", "turn a skill on, or anything else off"}, {"enter t", "tune: pick on, off, name-only or user-invocable-only"}, {"a", "stage every skill on; partial states are kept"}, {"n", "stage every skill off"}}},
		{"Save", []hint{{"s", "review the staged changes, then save or discard"}, {"R", "reload from disk and drop staged changes"}, {"r", "refresh: replace installed copies with links"}, {"q esc", "quit; asks first when changes are pending"}}},
	}
	var b strings.Builder
	for i, g := range groups {
		if i > 0 && m.height >= 30 {
			b.WriteString("\n")
		}
		b.WriteString(s.label.Render(g.title) + "\n")
		for _, r := range g.rows {
			b.WriteString("  " + s.accent.Render(fmt.Sprintf("%-10s", r.key)) + "  " + s.value.Render(r.desc) + "\n")
		}
	}
	if m.height >= 28 {
		b.WriteString("\n" + s.subtle.Render("Nothing changes on disk until you save. Adding or removing a skill needs a new Claude Code session."))
	}
	content := lipgloss.NewStyle().Width(max(1, m.width-8)).Render(strings.TrimRight(b.String(), "\n"))
	lines := strings.Split(content, "\n")
	h := max(1, m.height-chromeRows-2)
	start := min(m.helpOffset, max(0, len(lines)-h))
	return s.paneFocus.Width(m.width).Padding(0, 2).Render(strings.Join(lines[start:min(len(lines), start+h)], "\n"))
}

func (m *model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	if m.width == 0 {
		v.SetContent("✿ Opening the skill garden…")
		return v
	}
	if m.width < 32 || m.height < 12 {
		v.SetContent(lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render("✿ A little more room?\nResize to at least 32 × 12.\nq quits"))
		return v
	}
	switch m.mode {
	case modeTune:
		if m.tune != nil {
			v.SetContent(m.dialog("✿ "+m.staged[m.tuneIdx].Name, m.repo.Describe(m.staged[m.tuneIdx].Name), m.tune.View(), "esc back · enter choose"))
			return v
		}
	case modeReview:
		v.SetContent(m.reviewView())
		return v
	case modeRefresh:
		v.SetContent(m.refreshView())
		return v
	}

	bodyHeight := m.height - chromeRows
	var body string
	switch {
	case m.mode == modeHelp:
		body = lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, m.keysOverlay())
	case m.details:
		body = m.st.paneFocus.Width(m.width).Height(bodyHeight).Render(m.preview.View())
	default:
		lw := m.listWidth()
		left := m.st.paneFocus.Width(lw).Height(bodyHeight).Render(m.list.View())
		if m.width < 60 {
			body = left
			break
		}
		right := m.st.pane.Width(m.preview.Width() + 4).Height(bodyHeight).Render(m.preview.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}
	line := lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(1)
	v.SetContent(lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(bodyHeight).Render(body),
		line.Width(m.width).Render(m.statusLine()),
		line.Render(m.footer()),
	))
	return v
}

// short truncates by display width so multi-byte text is never cut mid-rune.
func short(s string, n int) string { return ansi.Truncate(s, n, "…") }

// tildify shows paths under the home directory as ~/…
func tildify(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}
