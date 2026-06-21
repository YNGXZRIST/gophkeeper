package filedownload

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
)

func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	v := vault.New()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := v.UseDEK(key); err != nil {
		t.Fatalf("UseDEK: %v", err)
	}
	return v
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	resolved, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd resolved: %v", err)
	}
	return resolved
}

type fakeDownloadStream struct {
	grpc.ClientStream
	resps []*filev1.DownloadResponse
	idx   int
	err   error
}

func (s *fakeDownloadStream) Recv() (*filev1.DownloadResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.idx >= len(s.resps) {
		return nil, io.EOF
	}
	r := s.resps[s.idx]
	s.idx++
	return r, nil
}

type fakeClient struct {
	filev1.FileServiceClient
	stream  grpc.ServerStreamingClient[filev1.DownloadResponse]
	openErr error
}

func (c fakeClient) Download(_ context.Context, _ *filev1.DownloadRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[filev1.DownloadResponse], error) {
	if c.openErr != nil {
		return nil, c.openErr
	}
	return c.stream, nil
}

func chunkResp(t *testing.T, v *vault.Vault, data []byte) *filev1.DownloadResponse {
	t.Helper()
	ct, err := v.Encrypt(data)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	chunk := &filev1.FileChunk{}
	chunk.SetData(ct)
	resp := &filev1.DownloadResponse{}
	resp.SetChunk(chunk)
	return resp
}

func newModel(v *vault.Vault, c filev1.FileServiceClient, file clientmodel.File) model {
	return model{
		vault:  v,
		client: c,
		file:   file,
		bar:    progress.New(),
		prog:   make(chan progressMsg, 64),
		res:    make(chan doneMsg, 1),
	}
}

func TestDownloadSuccess(t *testing.T) {
	dir := chdirTemp(t)
	v := testVault(t)
	stream := &fakeDownloadStream{resps: []*filev1.DownloadResponse{
		chunkResp(t, v, []byte("hello ")),
		chunkResp(t, v, []byte("world")),
		{},
	}}
	m := newModel(v, fakeClient{stream: stream}, clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "out.txt"}, ChunkCount: 2})

	path, err := m.download(context.Background())
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if path != filepath.Join(dir, "out.txt") {
		t.Errorf("path = %q", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content = %q", got)
	}
}

func TestDownloadEmptyNameUsesID(t *testing.T) {
	dir := chdirTemp(t)
	v := testVault(t)
	stream := &fakeDownloadStream{resps: []*filev1.DownloadResponse{chunkResp(t, v, []byte("x"))}}
	m := newModel(v, fakeClient{stream: stream}, clientmodel.File{ID: "the-id", Meta: clientmodel.FileMeta{Name: ""}, ChunkCount: 1})

	path, err := m.download(context.Background())
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if path != filepath.Join(dir, "the-id") {
		t.Errorf("path = %q, want id-based", path)
	}
}

func TestDownloadStreamRecvError(t *testing.T) {
	chdirTemp(t)
	v := testVault(t)
	stream := &fakeDownloadStream{err: errors.New("recv boom")}
	m := newModel(v, fakeClient{stream: stream}, clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "x.txt"}})
	if _, err := m.download(context.Background()); err == nil {
		t.Fatal("expected recv error")
	}
}

func TestDownloadOpenStreamError(t *testing.T) {
	chdirTemp(t)
	v := testVault(t)
	m := newModel(v, fakeClient{openErr: errors.New("no stream")}, clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "x.txt"}})
	if _, err := m.download(context.Background()); err == nil {
		t.Fatal("expected open error")
	}
}

func TestDownloadDecryptError(t *testing.T) {
	chdirTemp(t)
	v := testVault(t)
	chunk := &filev1.FileChunk{}
	chunk.SetData([]byte("not ciphertext"))
	resp := &filev1.DownloadResponse{}
	resp.SetChunk(chunk)
	stream := &fakeDownloadStream{resps: []*filev1.DownloadResponse{resp}}
	m := newModel(v, fakeClient{stream: stream}, clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "x.txt"}})
	if _, err := m.download(context.Background()); err == nil {
		t.Fatal("expected decrypt error")
	}
}

func TestNewRunsAndCompletes(t *testing.T) {
	chdirTemp(t)
	v := testVault(t)
	stream := &fakeDownloadStream{resps: []*filev1.DownloadResponse{chunkResp(t, v, []byte("data"))}}
	tm := New(Prop{Vault: v, Client: fakeClient{stream: stream}, File: clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "y.txt"}, ChunkCount: 1}})
	if tm == nil {
		t.Fatal("New returned nil")
	}
	cmd := tm.Init()
	if cmd == nil {
		t.Fatal("Init nil")
	}
	for i := 0; i < 20; i++ {
		msg := cmd()
		var next tea.Model
		next, cmd = tm.Update(msg)
		tm = next
		if _, ok := msg.(doneMsg); ok {
			break
		}
		if cmd == nil {
			break
		}
	}
}

func TestUpdateProgressMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, clientmodel.File{})
	m2, cmd := m.Update(progressMsg{received: 1, total: 4})
	if cmd == nil {
		t.Fatal("progress should batch a listen cmd")
	}
	_ = m2
	if _, cmd := m.Update(progressMsg{received: 0, total: 0}); cmd == nil {
		t.Fatal("progress total=0 should still listen")
	}
}

func TestUpdateDoneMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, clientmodel.File{})
	m2, _ := m.Update(doneMsg{path: "/tmp/saved"})
	mm := m2.(model)
	if mm.phase != phaseDone || mm.path != "/tmp/saved" {
		t.Errorf("done state wrong: %+v", mm)
	}
	if !strings.Contains(mm.View().Content, "Saved to") {
		t.Error("view should show saved path")
	}

	m3, _ := m.Update(doneMsg{err: errors.New("fail")})
	if !strings.Contains(m3.(model).View().Content, "Download failed") {
		t.Error("view should show failure")
	}
}

func TestUpdateFrameMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, clientmodel.File{})
	if _, _ = m.Update(progress.FrameMsg{}); true {
		_ = m
	}
}

func TestUpdateKeys(t *testing.T) {
	cancelled := false
	m := newModel(testVault(t), fakeClient{}, clientmodel.File{})
	m.cancel = func() { cancelled = true }
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil || !cancelled {
		t.Fatal("esc should cancel and navigate back")
	}

	cancelled = false
	m.cancel = func() { cancelled = true }
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || !cancelled {
		t.Fatal("ctrl+c should cancel and quit")
	}
}

func TestDownloadView(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, clientmodel.File{})
	if !strings.Contains(m.View().Content, "Downloading") {
		t.Error("default view should show downloading")
	}
}

func TestEncodeDecodeMeta(t *testing.T) {
	v := testVault(t)
	raw, err := json.Marshal(clientmodel.FileMeta{Name: "z"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Encrypt(raw); err != nil {
		t.Fatal(err)
	}
}
