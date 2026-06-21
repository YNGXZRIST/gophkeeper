// Package upload is the screen that picks a local file and streams it to the
// server in encrypted chunks, showing upload progress.
package fileupload

import (
	"context"
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/views/files/internal/filepick"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	"io"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"golang.org/x/sync/errgroup"
)

const chunkSize = 1 << 20

type phase int

const (
	phasePick phase = iota
	phaseUpload
	phaseDone
)
const label = "Upload file"

const (
	hintCancel = "esc — cancel"
	hintBack   = "esc — back"
)

type selectedMsg struct{ path string }
type progressMsg struct{ sent, total int }
type doneMsg struct {
	id  string
	err error
}

type Repo interface {
	Insert(ctx context.Context, id string, meta []byte, chunkCount int, version int64) error
}

type Prop struct {
	Vault  *vault.Vault
	Client filev1.FileServiceClient
	Repo   Repo
}

type model struct {
	vault  *vault.Vault
	client filev1.FileServiceClient
	repo   Repo
	picker tea.Model
	phase  phase
	bar    progress.Model
	prog   chan progressMsg
	res    chan doneMsg
	cancel context.CancelFunc
	errMsg string
}

func New(p Prop) tea.Model {
	m := model{vault: p.Vault, client: p.Client, repo: p.Repo, bar: progress.New()}
	m.picker = filepick.New(filepick.Prop{
		Title: label,
		OnSelect: func(path string) tea.Cmd {
			return func() tea.Msg { return selectedMsg{path: path} }
		},
	})
	return m
}

func (m model) Init() tea.Cmd {
	return m.picker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case selectedMsg:
		return m.startUpload(msg.path)
	case progressMsg:
		cmd := m.bar.SetPercent(float64(msg.sent) / float64(msg.total))
		return m, tea.Batch(cmd, m.listen())
	case doneMsg:
		m.phase = phaseDone
		if msg.err != nil {
			m.errMsg = "Upload failed. " + msg.err.Error()
			return m, nil
		}
		return m, tea.Sequence(nav.Back(), nav.Reload())
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		if m.phase != phasePick {
			switch msg.String() {
			case keys.CTRL_C:
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			case keys.ESC:
				if m.cancel != nil {
					m.cancel()
				}
				return m, nav.Back()
			}
			return m, nil
		}
	}

	if m.phase == phasePick {
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) startUpload(path string) (tea.Model, tea.Cmd) {
	f, err := os.Open(path)
	if err != nil {
		m.phase = phaseDone
		m.errMsg = "Cannot open file. " + err.Error()
		return m, nil
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		m.phase = phaseDone
		m.errMsg = "Cannot read file. " + err.Error()
		return m, nil
	}
	size := info.Size()
	total := int((size + chunkSize - 1) / chunkSize)
	meta := clientmodel.FileMeta{Name: filepath.Base(path), Size: size}

	m.phase = phaseUpload
	m.prog = make(chan progressMsg)
	m.res = make(chan doneMsg, 1)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go m.run(ctx, f, meta, total)
	return m, m.listen()
}

func (m model) run(ctx context.Context, f *os.File, meta clientmodel.FileMeta, total int) {
	defer func() { _ = f.Close() }()
	id, err := m.doUpload(ctx, f, meta, total)
	close(m.prog)
	m.res <- doneMsg{id: id, err: err}
}

func (m model) doUpload(ctx context.Context, f *os.File, meta clientmodel.FileMeta, total int) (string, error) {
	stream, err := m.client.Upload(ctx)
	if err != nil {
		return "", err
	}

	rawMeta, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	metaCt, err := m.vault.Encrypt(rawMeta)
	if err != nil {
		return "", err
	}
	header := &filev1.FileHeader{}
	header.SetMeta(metaCt)
	header.SetChunkCount(int32(total))
	headerReq := &filev1.UploadRequest{}
	headerReq.SetHeader(header)
	if err := stream.Send(headerReq); err != nil {
		return "", err
	}

	g, ctx := errgroup.WithContext(ctx)
	chunks := make(chan []byte)

	g.Go(func() error {
		defer close(chunks)
		buf := make([]byte, chunkSize)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				ct, encErr := m.vault.Encrypt(buf[:n])
				if encErr != nil {
					return encErr
				}
				select {
				case chunks <- ct:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	})

	g.Go(func() error {
		idx := 0
		for ct := range chunks {
			chunk := &filev1.FileChunk{}
			chunk.SetIdx(int32(idx))
			chunk.SetData(ct)
			req := &filev1.UploadRequest{}
			req.SetChunk(chunk)
			if err := stream.Send(req); err != nil {
				return err
			}
			idx++
			select {
			case m.prog <- progressMsg{sent: idx, total: total}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return "", err
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}
	id := resp.GetId()
	if err := m.repo.Insert(context.Background(), id, metaCt, total, 1); err != nil {
		return "", err
	}
	return id, nil
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
	switch m.phase {
	case phaseUpload:
		body := "Uploading…\n\n" + m.bar.View()
		return tea.NewView(layout.Page("Upload file", body, hintCancel))
	case phaseDone:
		body := "Done."
		if m.errMsg != "" {
			body = m.errMsg
		}
		return tea.NewView(layout.Page("Upload file", body, hintBack))
	default:
		return m.picker.View()
	}
}
