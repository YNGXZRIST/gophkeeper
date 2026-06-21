package fileupload

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
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

type fakeUploadStream struct {
	grpc.ClientStream
	sendErr  error
	closeErr error
	id       string
	sent     int
}

func (s *fakeUploadStream) Send(_ *filev1.UploadRequest) error {
	s.sent++
	return s.sendErr
}

func (s *fakeUploadStream) CloseAndRecv() (*filev1.UploadResponse, error) {
	if s.closeErr != nil {
		return nil, s.closeErr
	}
	resp := &filev1.UploadResponse{}
	resp.SetId(s.id)
	return resp, nil
}

type fakeClient struct {
	filev1.FileServiceClient
	stream  grpc.ClientStreamingClient[filev1.UploadRequest, filev1.UploadResponse]
	openErr error
}

func (c fakeClient) Upload(_ context.Context, _ ...grpc.CallOption) (grpc.ClientStreamingClient[filev1.UploadRequest, filev1.UploadResponse], error) {
	if c.openErr != nil {
		return nil, c.openErr
	}
	return c.stream, nil
}

type fakeRepo struct {
	err    error
	gotID  string
	called bool
}

func (r *fakeRepo) Insert(_ context.Context, id string, _ []byte, _ int, _ int64) error {
	r.called = true
	r.gotID = id
	return r.err
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "upload.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func newModel(v *vault.Vault, c filev1.FileServiceClient, repo Repo) model {
	return model{vault: v, client: c, repo: repo, bar: progress.New()}
}

func TestNewInitView(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Client: fakeClient{}, Repo: &fakeRepo{}})
	if m == nil {
		t.Fatal("New nil")
	}
	if m.Init() == nil {
		t.Fatal("Init nil")
	}
	if m.View().Content == "" {
		t.Error("default (pick) view empty")
	}
}

func TestDoUploadSuccess(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{}
	stream := &fakeUploadStream{id: "new-id"}
	m := newModel(v, fakeClient{stream: stream}, repo)
	m.prog = make(chan progressMsg, 16)

	f, err := os.Open(writeTempFile(t, "some content"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	meta := clientmodel.FileMeta{Name: "upload.txt", Size: 12}

	id, err := m.doUpload(context.Background(), f, meta, 1)
	if err != nil {
		t.Fatalf("doUpload: %v", err)
	}
	if id != "new-id" {
		t.Errorf("id = %q", id)
	}
	if !repo.called || repo.gotID != "new-id" {
		t.Errorf("repo not called correctly: %+v", repo)
	}
	if stream.sent < 2 {
		t.Errorf("expected header+chunk sends, got %d", stream.sent)
	}
}

func TestDoUploadOpenStreamError(t *testing.T) {
	v := testVault(t)
	m := newModel(v, fakeClient{openErr: errors.New("no")}, &fakeRepo{})
	f, _ := os.Open(writeTempFile(t, "x"))
	defer func() { _ = f.Close() }()
	if _, err := m.doUpload(context.Background(), f, clientmodel.FileMeta{}, 1); err == nil {
		t.Fatal("expected open stream error")
	}
}

func TestDoUploadSendError(t *testing.T) {
	v := testVault(t)
	stream := &fakeUploadStream{sendErr: errors.New("send fail")}
	m := newModel(v, fakeClient{stream: stream}, &fakeRepo{})
	m.prog = make(chan progressMsg, 16)
	f, _ := os.Open(writeTempFile(t, "data"))
	defer func() { _ = f.Close() }()
	if _, err := m.doUpload(context.Background(), f, clientmodel.FileMeta{Name: "a", Size: 4}, 1); err == nil {
		t.Fatal("expected send error")
	}
}

func TestDoUploadCloseError(t *testing.T) {
	v := testVault(t)
	stream := &fakeUploadStream{closeErr: errors.New("close fail")}
	m := newModel(v, fakeClient{stream: stream}, &fakeRepo{})
	m.prog = make(chan progressMsg, 16)
	f, _ := os.Open(writeTempFile(t, "data"))
	defer func() { _ = f.Close() }()
	if _, err := m.doUpload(context.Background(), f, clientmodel.FileMeta{Name: "a", Size: 4}, 1); err == nil {
		t.Fatal("expected close error")
	}
}

func TestDoUploadRepoError(t *testing.T) {
	v := testVault(t)
	stream := &fakeUploadStream{id: "id"}
	m := newModel(v, fakeClient{stream: stream}, &fakeRepo{err: errors.New("db")})
	m.prog = make(chan progressMsg, 16)
	f, _ := os.Open(writeTempFile(t, "data"))
	defer func() { _ = f.Close() }()
	if _, err := m.doUpload(context.Background(), f, clientmodel.FileMeta{Name: "a", Size: 4}, 1); err == nil {
		t.Fatal("expected repo insert error")
	}
}

func TestStartUploadOpenError(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m2, _ := m.startUpload(filepath.Join(t.TempDir(), "missing.txt"))
	mm := m2.(model)
	if mm.phase != phaseDone || !strings.Contains(mm.errMsg, "Cannot open file") {
		t.Errorf("expected open error state, got %+v", mm)
	}
}

func TestStartUploadSuccess(t *testing.T) {
	v := testVault(t)
	stream := &fakeUploadStream{id: "ok"}
	m := newModel(v, fakeClient{stream: stream}, &fakeRepo{})
	path := writeTempFile(t, "payload content here")

	m2, cmd := m.startUpload(path)
	mm := m2.(model)
	if mm.phase != phaseUpload {
		t.Errorf("phase = %v, want upload", mm.phase)
	}
	if cmd == nil {
		t.Fatal("startUpload should return a listen cmd")
	}
	for i := 0; i < 50; i++ {
		msg := cmd()
		var next tea.Model
		next, cmd = mm.Update(msg)
		mm = next.(model)
		if _, ok := msg.(doneMsg); ok {
			break
		}
		if cmd == nil {
			break
		}
	}
}

func TestUpdateSelectedMsg(t *testing.T) {
	v := testVault(t)
	stream := &fakeUploadStream{id: "ok"}
	m := newModel(v, fakeClient{stream: stream}, &fakeRepo{})
	m.picker = New(Prop{Vault: v, Client: fakeClient{stream: stream}, Repo: &fakeRepo{}}).(model).picker
	path := writeTempFile(t, "abc")
	m2, cmd := m.Update(selectedMsg{path: path})
	if m2.(model).phase != phaseUpload || cmd == nil {
		t.Fatal("selectedMsg should start upload")
	}
}

func TestUpdateProgressMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m.prog = make(chan progressMsg, 1)
	m.res = make(chan doneMsg, 1)
	_, cmd := m.Update(progressMsg{sent: 1, total: 2})
	if cmd == nil {
		t.Fatal("progress should batch listen cmd")
	}
}

func TestUpdateDoneMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m2, cmd := m.Update(doneMsg{id: "x"})
	if m2.(model).phase != phaseDone {
		t.Error("done should set phaseDone")
	}
	if cmd == nil {
		t.Fatal("success done should sequence Back+Reload")
	}
	if !strings.Contains(m2.(model).View().Content, "Done") {
		t.Error("done view should show Done")
	}

	m3, _ := m.Update(doneMsg{err: errors.New("boom")})
	if !strings.Contains(m3.(model).View().Content, "Upload failed") {
		t.Error("error done view should show failure")
	}
}

func TestUpdateFrameMsg(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m.phase = phaseUpload
	_, _ = m.Update(progress.FrameMsg{})
}

func TestUpdateKeysDuringUpload(t *testing.T) {
	cancelled := false
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m.phase = phaseUpload
	m.cancel = func() { cancelled = true }
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil || !cancelled {
		t.Fatal("esc during upload should cancel and go back")
	}

	cancelled = false
	m.cancel = func() { cancelled = true }
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil || !cancelled {
		t.Fatal("ctrl+c during upload should cancel and quit")
	}
}

func TestUpdateKeyDuringPickForwards(t *testing.T) {
	v := testVault(t)
	m := New(Prop{Vault: v, Client: fakeClient{}, Repo: &fakeRepo{}}).(model)
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
}

func TestViewUploadPhase(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m.phase = phaseUpload
	if !strings.Contains(m.View().Content, "Uploading") {
		t.Error("upload view should show Uploading")
	}
}

func TestListen(t *testing.T) {
	m := newModel(testVault(t), fakeClient{}, &fakeRepo{})
	m.prog = make(chan progressMsg, 1)
	m.res = make(chan doneMsg, 1)
	m.prog <- progressMsg{sent: 1, total: 1}
	if _, ok := m.listen()().(progressMsg); !ok {
		t.Error("listen should return queued progressMsg")
	}
	close(m.prog)
	m.res <- doneMsg{id: "z"}
	if _, ok := m.listen()().(doneMsg); !ok {
		t.Error("listen should fall through to doneMsg when prog closed")
	}
}
