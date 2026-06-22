// Package fileconflict resolves file sync conflicts and shows sync errors.
package fileconflict

import (
	"context"
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const label = "File conflicts"
const hint = "↑/↓ — move · m — keep mine · s — take server · esc — back"

type Repo interface {
	ListConflicts(ctx context.Context) ([]repository.ConflictRow, error)
	ResolveKeepMine(ctx context.Context, id string) error
	ResolveTakeServer(ctx context.Context, id string) error
}

type Syncer interface {
	SyncAll(ctx context.Context) error
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
	Sync  Syncer
}

type item struct {
	id     string
	local  clientmodel.FileMeta
	server clientmodel.FileMeta
}

type syncedMsg struct{ err error }
type loadedMsg struct {
	items []item
	err   error
}
type resolvedMsg struct{ err error }

type model struct {
	vault    *vault.Vault
	repo     Repo
	sync     Syncer
	items    []item
	selected int
	loading  bool
	status   string
}

func New(p Prop) tea.Model {
	return model{vault: p.Vault, repo: p.Repo, sync: p.Sync, loading: true}
}

func (m model) Init() tea.Cmd {
	return m.runSync()
}

func (m model) runSync() tea.Cmd {
	sync := m.sync
	return func() tea.Msg {
		if sync == nil {
			return syncedMsg{}
		}
		return syncedMsg{err: sync.SyncAll(context.Background())}
	}
}

func (m model) load() tea.Cmd {
	repo := m.repo
	vlt := m.vault
	return func() tea.Msg {
		rows, err := repo.ListConflicts(context.Background())
		if err != nil {
			return loadedMsg{err: err}
		}
		items := make([]item, 0, len(rows))
		for _, r := range rows {
			items = append(items, item{
				id:     r.ID,
				local:  decode(vlt, r.Local),
				server: decode(vlt, r.Server),
			})
		}
		return loadedMsg{items: items}
	}
}

func (m model) resolve(id string, keepMine bool) tea.Cmd {
	repo := m.repo
	return func() tea.Msg {
		var err error
		if keepMine {
			err = repo.ResolveKeepMine(context.Background(), id)
		} else {
			err = repo.ResolveTakeServer(context.Background(), id)
		}
		return resolvedMsg{err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case syncedMsg:
		if msg.err != nil {
			m.status = "Sync error: " + msg.err.Error()
		} else {
			m.status = ""
		}
		return m, m.load()
	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "Load error: " + msg.err.Error()
			return m, nil
		}
		m.items = msg.items
		if m.selected >= len(m.items) {
			m.selected = 0
		}
		return m, nil
	case resolvedMsg:
		if msg.err != nil {
			m.status = "Resolve error: " + msg.err.Error()
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(m.runSync(), nav.SyncNow())
	case tea.KeyPressMsg:
		switch msg.String() {
		case keys.Esc:
			return m, nav.Back()
		case keys.Up:
			if m.selected > 0 {
				m.selected--
			}
		case keys.Down:
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case keys.M:
			if id, ok := m.currentID(); ok {
				m.loading = true
				return m, m.resolve(id, true)
			}
		case keys.S:
			if id, ok := m.currentID(); ok {
				m.loading = true
				return m, m.resolve(id, false)
			}
		}
	}
	return m, nil
}

func (m model) currentID() (string, bool) {
	if m.selected < 0 || m.selected >= len(m.items) {
		return "", false
	}
	return m.items[m.selected].id, true
}

func (m model) View() tea.View {
	if m.loading {
		return tea.NewView(layout.Page(label, "Syncing…", hint))
	}
	var b strings.Builder
	if m.status != "" {
		b.WriteString(m.status + "\n\n")
	}
	if len(m.items) == 0 {
		b.WriteString("No conflicts.")
		return tea.NewView(layout.Page(label, b.String(), hint))
	}
	for i, it := range m.items {
		marker := "  "
		if i == m.selected {
			marker = "> "
		}
		b.WriteString(marker + line(it.local) + "\n")
	}
	cur := m.items[m.selected]
	b.WriteString("\n" + theme.Blurred.Render("MINE:   ") + theme.Filled.Render(line(cur.local)))
	b.WriteString("\n" + theme.Blurred.Render("SERVER: ") + theme.Filled.Render(line(cur.server)))
	return tea.NewView(layout.Page(label, b.String(), hint))
}

func line(m clientmodel.FileMeta) string {
	t := strings.ReplaceAll(m.Name, "\n", " ")
	if t == "" {
		return "—"
	}
	return t
}

func decode(v *vault.Vault, blob []byte) clientmodel.FileMeta {
	var m clientmodel.FileMeta
	raw, err := v.Decrypt(blob)
	if err != nil {
		return m
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return clientmodel.FileMeta{}
	}
	return m
}
