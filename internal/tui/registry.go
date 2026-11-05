package tui

import (
	"fmt"
	"path"

	tea "charm.land/bubbletea/v2"

	"github.com/jorden-parker/skills/internal/skills"
)

type listView int

const (
	catalogView listView = iota
	registryView
)

type indexMsg struct {
	ids []string
	err error
}

type manifestMsg struct {
	id   string
	body string
	err  error
}

const loadingManifest = "Loading SKILL.md from skills.sh…"

func (m *model) toggleView() tea.Cmd {
	if m.view == registryView {
		m.view = catalogView
		return m.resetList()
	}
	m.view = registryView
	if m.registryIDs != nil || m.loadingIndex {
		return m.resetList()
	}
	m.loadingIndex = true
	m.setStatus("Loading the skills.sh index…", false)
	client := m.registry
	load := func() tea.Msg {
		ids, err := client.Index()
		return indexMsg{ids: ids, err: err}
	}
	return tea.Batch(m.resetList(), load)
}

func (m *model) indexLoaded(msg indexMsg) tea.Cmd {
	m.loadingIndex = false
	if msg.err != nil {
		m.setStatus("skills.sh index: "+msg.err.Error(), true)
		return nil
	}
	m.registryIDs = append([]string{}, msg.ids...)
	m.appendRegistryEntries()
	m.setStatus(fmt.Sprintf("%d skills listed from skills.sh. Space stages one; s saves it.", len(m.registryIDs)), false)
	return m.resetList()
}

func (m *model) manifestLoaded(msg manifestMsg) tea.Cmd {
	delete(m.manifestsPending, msg.id)
	if msg.err != nil {
		m.manifests[msg.id] = msg.err.Error()
	} else {
		m.manifests[msg.id] = msg.body
	}
	if m.selectedName() == msg.id {
		return m.refreshPreview()
	}
	return nil
}

func (m *model) appendRegistryEntries() {
	present := make(map[string]bool, len(m.staged))
	for _, e := range m.staged {
		present[e.Name] = true
	}
	for _, id := range m.registryIDs {
		if present[id] {
			continue
		}
		m.before = append(m.before, skills.Entry{Name: id, State: skills.Off})
		m.staged = append(m.staged, skills.Entry{Name: id, State: skills.Off})
	}
}

func (m *model) fetchManifest(id string) tea.Cmd {
	client := m.registry
	return func() tea.Msg {
		body, err := client.Manifest(id)
		return manifestMsg{id: id, body: body, err: err}
	}
}

func (m *model) previewBody(id string) (string, tea.Cmd) {
	if !skills.IsRegistryID(id) || m.repo.Has(id) {
		body, err := m.repo.Manifest(id)
		if err != nil {
			return m.st.err.Render(err.Error()), nil
		}
		return body, nil
	}
	if body, cached := m.manifests[id]; cached {
		return body, nil
	}
	if m.manifestsPending[id] {
		return loadingManifest, nil
	}
	m.manifestsPending[id] = true
	return loadingManifest, m.fetchManifest(id)
}

func (m *model) inCatalog(e skills.Entry) bool {
	return !skills.IsRegistryID(e.Name) || m.repo.Has(e.Name) || e.State.Linked()
}

func (m *model) visible(e skills.Entry) bool {
	if m.view == registryView {
		return skills.IsRegistryID(e.Name)
	}
	return m.inCatalog(e)
}

func (m *model) visibleCount() int {
	n := 0
	for _, e := range m.staged {
		if m.visible(e) {
			n++
		}
	}
	return n
}

func (m *model) catalogCount() int {
	n := 0
	for _, e := range m.staged {
		if m.inCatalog(e) {
			n++
		}
	}
	return n
}

func (m *model) catalogOnly() bool {
	if m.view != registryView {
		return false
	}
	m.setStatus("a and n work in the catalog view only. Press b to go back.", false)
	return true
}

func (m *model) description(name string) string {
	if skills.IsRegistryID(name) && !m.repo.Has(name) {
		return "skills.sh · " + path.Dir(name)
	}
	return m.repo.Describe(name)
}

func (m *model) countLine() string {
	if m.view == registryView {
		return fmt.Sprintf("skills.sh · %d listed · %d pending", len(m.registryIDs), m.pending())
	}
	return fmt.Sprintf("%d of %d growing · %d pending", m.growing(), m.catalogCount(), m.pending())
}

func (m *model) updateTitle() {
	m.list.Title = m.st.subtle.Render(fmt.Sprintf("%d skills · / to filter", m.visibleCount()))
}
