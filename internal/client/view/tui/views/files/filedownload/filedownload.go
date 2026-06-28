// Package filedownload streams a file from the server, decrypts each chunk, and
// writes it to disk, showing download progress.
package filedownload

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	"io"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

const label = "Download file"

const (
	hintCancel = "esc — cancel"
	hintBack   = "esc — back"
)

type phase int

const (
	phaseDownload phase = iota
	phaseDone
)

type progressMsg struct{ received, total int }
type doneMsg struct {
	path string
	err  error
}

type Prop struct {
	Vault  *vault.Vault
	Client filev1.FileServiceClient
	File   clientmodel.File
}

type model struct {
	vault  *vault.Vault
	client filev1.FileServiceClient
	file   clientmodel.File
	phase  phase
	bar    progress.Model
	prog   chan progressMsg
	res    chan doneMsg
	cancel context.CancelFunc
	path   string
	errMsg string
}

func New(p Prop) tea.Model {
	ctx, cancel := context.WithCancel(context.Background())
	m := model{
		vault:  p.Vault,
		client: p.Client,
		file:   p.File,
		bar:    progress.New(),
		prog:   make(chan progressMsg),
		res:    make(chan doneMsg, 1),
		cancel: cancel,
	}
	go m.run(ctx)
	return m
}

func (m model) Init() tea.Cmd {
	return m.listen()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		var cmd tea.Cmd
		if msg.total > 0 {
			cmd = m.bar.SetPercent(float64(msg.received) / float64(msg.total))
		}
		return m, tea.Batch(cmd, m.listen())
	case doneMsg:
		m.phase = phaseDone
		if msg.err != nil {
			m.errMsg = "Download failed. " + msg.err.Error()
			return m, nil
		}
		m.path = msg.path
		return m, nil
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case keys.CtrlC:
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		case keys.Esc:
			if m.cancel != nil {
				m.cancel()
			}
			return m, nav.Back()
		}
	}
	return m, nil
}

func (m model) run(ctx context.Context) {
	path, err := m.download(ctx)
	close(m.prog)
	m.res <- doneMsg{path: path, err: err}
}

func (m model) download(ctx context.Context) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name := filepath.Base(m.file.Meta.Name)
	if name == "." || name == ".." || name == string(os.PathSeparator) {
		name = m.file.ID
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	req := &filev1.DownloadRequest{}
	req.SetId(m.file.ID)
	stream, err := m.client.Download(ctx, req)
	if err != nil {
		return "", err
	}

	total := m.file.ChunkCount
	received := 0
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		chunk := resp.GetChunk()
		if chunk == nil {
			continue
		}
		plain, err := m.vault.Decrypt(chunk.GetData())
		if err != nil {
			return "", err
		}
		if _, err := f.Write(plain); err != nil {
			return "", err
		}
		received++
		select {
		case m.prog <- progressMsg{received: received, total: total}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return path, nil
}

func (m model) listen() tea.Cmd {
	return func() tea.Msg {
		if p, ok := <-m.prog; ok {
			return p
		}
		return <-m.res
	}
}

func (m model) View() tea.View {
	if m.phase == phaseDone {
		body := "Saved to " + m.path
		if m.errMsg != "" {
			body = m.errMsg
		}
		return tea.NewView(layout.Page(label, body, hintBack))
	}
	body := "Downloading…\n\n" + m.bar.View()
	return tea.NewView(layout.Page(label, body, hintCancel))
}
