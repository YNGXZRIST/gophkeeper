package noteform

import (
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/nav"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func rune1(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func collectMsgs(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return flatten(cmd())
}

func flatten(msg tea.Msg) []tea.Msg {
	if msg == nil {
		return nil
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			if c == nil {
				continue
			}
			out = append(out, flatten(c())...)
		}
		return out
	case tea.Cmd:
		if m == nil {
			return nil
		}
		return flatten(m())
	}

	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		var out []tea.Msg
		for i := 0; i < rv.Len(); i++ {
			c, ok := rv.Index(i).Interface().(tea.Cmd)
			if !ok || c == nil {
				continue
			}
			out = append(out, flatten(c())...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestNewAndGetters(t *testing.T) {
	v := testVault(t)
	m := New(v, "Note", clientmodel.NoteData{Text: "hello", Meta: "tag"}, func([]byte) error { return nil })
	if m.GetText() != "hello" {
		t.Errorf("GetText = %q, want hello", m.GetText())
	}
	if m.GetMeta() != "tag" {
		t.Errorf("GetMeta = %q, want tag", m.GetMeta())
	}
	d := m.GetNoteData()
	if d.Text != "hello" || d.Meta != "tag" {
		t.Errorf("GetNoteData = %+v", d)
	}
}

func TestInit(t *testing.T) {
	m := New(testVault(t), "Note", clientmodel.NoteData{}, func([]byte) error { return nil })
	if m.Init() == nil {
		t.Fatal("Init returned nil cmd")
	}
}

func TestView(t *testing.T) {
	m := New(testVault(t), "My Title", clientmodel.NoteData{Text: "t"}, func([]byte) error { return nil })
	v := m.View()
	if !strings.Contains(v.Content, "My Title") {
		t.Errorf("View content missing title: %q", v.Content)
	}
}

func TestUpdateEsc(t *testing.T) {
	m := New(testVault(t), "Note", clientmodel.NoteData{}, func([]byte) error { return nil })
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	msgs := collectMsgs(t, cmd)
	found := false
	for _, msg := range msgs {
		if _, ok := msg.(nav.BackMsg); ok {
			found = true
		}
	}
	if !found {
		t.Fatalf("esc did not produce BackMsg, got %#v", msgs)
	}
}

func TestUpdateCtrlC(t *testing.T) {
	m := New(testVault(t), "Note", clientmodel.NoteData{}, func([]byte) error { return nil })
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, Text: ""})
	if cmd == nil {
		t.Fatal("ctrl+c returned nil cmd")
	}
}

func TestSubmitSuccess(t *testing.T) {
	v := testVault(t)
	var saved []byte
	m := New(v, "Note", clientmodel.NoteData{Text: "body", Meta: "m"}, func(ct []byte) error {
		saved = ct
		return nil
	})

	mm := tea.Model(m)
	var cmd tea.Cmd
	mm, _ = mm.Update(rune1('x'))
	mm, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mm, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = mm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submit produced no cmd")
	}
	msgs := collectMsgs(t, cmd)
	var sm savedMsg
	got := false
	for _, msg := range msgs {
		if s, ok := msg.(savedMsg); ok {
			sm = s
			got = true
		}
	}
	if !got {
		t.Fatalf("no savedMsg, got %#v", msgs)
	}
	if sm.err != nil {
		t.Fatalf("savedMsg err = %v", sm.err)
	}
	if len(saved) == 0 {
		t.Fatal("save callback not invoked with ciphertext")
	}

	raw, err := v.Decrypt(saved)
	if err != nil {
		t.Fatalf("decrypt saved: %v", err)
	}
	var d clientmodel.NoteData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.Meta != "m" {
		t.Errorf("saved meta = %q", d.Meta)
	}
}

func TestSubmitSaveError(t *testing.T) {
	v := testVault(t)
	m := New(v, "Note", clientmodel.NoteData{Text: "t", Meta: "m"}, func([]byte) error {
		return errors.New("boom")
	})
	cmd := m.submit()
	msgs := collectMsgs(t, cmd)
	if len(msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(msgs))
	}
	sm, ok := msgs[0].(savedMsg)
	if !ok || sm.err == nil {
		t.Fatalf("expected savedMsg with error, got %#v", msgs[0])
	}
}

func TestUpdateSavedMsgError(t *testing.T) {
	m := New(testVault(t), "Note", clientmodel.NoteData{}, func([]byte) error { return nil })
	mm, cmd := m.Update(savedMsg{err: errors.New("fail")})
	if cmd != nil {
		t.Errorf("expected nil cmd on save error")
	}
	if !strings.Contains(mm.View().Content, "Save failed") {
		t.Errorf("view missing error message: %q", mm.View().Content)
	}
}

func TestUpdateSavedMsgSuccess(t *testing.T) {
	m := New(testVault(t), "Note", clientmodel.NoteData{}, func([]byte) error { return nil })
	_, cmd := m.Update(savedMsg{})
	msgs := collectMsgs(t, cmd)
	var back, reload, sync bool
	for _, msg := range msgs {
		switch msg.(type) {
		case nav.BackMsg:
			back = true
		case nav.ReloadMsg:
			reload = true
		case nav.SyncNowMsg:
			sync = true
		}
	}
	if !back || !reload || !sync {
		t.Fatalf("savedMsg success cmds: back=%v reload=%v sync=%v (%#v)", back, reload, sync, msgs)
	}
}

func TestSubmitEncryptError(t *testing.T) {

	locked := vault.New()
	m := New(locked, "Note", clientmodel.NoteData{Text: "t", Meta: "m"}, func([]byte) error { return nil })
	msgs := collectMsgs(t, m.submit())
	if len(msgs) != 1 {
		t.Fatalf("want 1 msg, got %#v", msgs)
	}
	if sm, ok := msgs[0].(savedMsg); !ok || sm.err == nil {
		t.Fatalf("expected encrypt error, got %#v", msgs[0])
	}
}

var _ = keys.Esc
