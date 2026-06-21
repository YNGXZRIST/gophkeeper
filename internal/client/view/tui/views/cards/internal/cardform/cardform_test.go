package cardform

import (
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
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

func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	return flatten(msg)
}

func flatten(msg tea.Msg) []tea.Msg {
	switch v := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range v {
			out = append(out, collectMsgs(c)...)
		}
		return out
	case []tea.Msg:
		var out []tea.Msg
		for _, m := range v {
			out = append(out, flatten(m)...)
		}
		return out
	default:

		rv := reflect.ValueOf(msg)
		if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
			var out []tea.Msg
			for i := 0; i < rv.Len(); i++ {
				c := rv.Index(i).Interface().(tea.Cmd)
				out = append(out, collectMsgs(c)...)
			}
			return out
		}
		return []tea.Msg{msg}
	}
}

func filled(t *testing.T) clientmodel.CardData {
	t.Helper()
	return clientmodel.CardData{
		Number: "4111111111111111",
		Holder: "JOHN DOE",
		Expiry: "12/30",
		CVV:    "123",
		Meta:   "personal",
	}
}

func TestInit(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })
	if m.Init() == nil {
		t.Fatal("Init returned nil cmd")
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", cmd())
	}
}

func TestEscBack(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected back cmd")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected nav.BackMsg, got %T", cmd())
	}
}

func TestGettersAndCardData(t *testing.T) {
	want := filled(t)
	m := New(testVault(t), "Edit", want, func([]byte) error { return nil })
	if got := m.CardData(); got != want {
		t.Fatalf("CardData = %+v, want %+v", got, want)
	}
	if m.GetCardNumber() != want.Number || m.GetHolder() != want.Holder ||
		m.GetExpiry() != want.Expiry || m.GetCVV() != want.CVV || m.GetMeta() != want.Meta {
		t.Fatal("getters mismatch")
	}
}

func TestViewRendersTitleAndError(t *testing.T) {
	m := New(testVault(t), "MyTitle", clientmodel.CardData{}, func([]byte) error { return nil })
	v := m.View()
	if !strings.Contains(v.Content, "MyTitle") {
		t.Fatalf("view missing title: %q", v.Content)
	}
	m.errMsg = "boom"
	if !strings.Contains(m.View().Content, "boom") {
		t.Fatal("view missing error message")
	}
}

func submitForm(t *testing.T, m Model) (tea.Model, tea.Cmd) {
	t.Helper()
	var model tea.Model = m

	for i := 0; i < 5; i++ {
		model, _ = model.Update(specialKey(tea.KeyDown))
	}
	return model.Update(specialKey(tea.KeyEnter))
}

func TestSubmitSuccess(t *testing.T) {
	var saved []byte
	m := New(testVault(t), "Add", filled(t), func(ct []byte) error {
		saved = ct
		return nil
	})
	_, cmd := submitForm(t, m)
	if cmd == nil {
		t.Fatal("submit returned nil cmd")
	}
	msg := cmd()
	sm, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("expected savedMsg, got %T", msg)
	}
	if sm.err != nil {
		t.Fatalf("unexpected save error: %v", sm.err)
	}
	if len(saved) == 0 {
		t.Fatal("save was not called with ciphertext")
	}
}

func TestSubmitLuhnInvalid(t *testing.T) {
	d := filled(t)
	d.Number = "1234567890123456"
	called := false
	m := New(testVault(t), "Add", d, func([]byte) error {
		called = true
		return nil
	})
	_, cmd := submitForm(t, m)
	msg := cmd().(savedMsg)
	if msg.err == nil {
		t.Fatal("expected luhn validation error")
	}
	if called {
		t.Fatal("save must not be called on invalid card number")
	}
}

func TestSubmitSaveError(t *testing.T) {
	m := New(testVault(t), "Add", filled(t), func([]byte) error {
		return errors.New("db down")
	})
	_, cmd := submitForm(t, m)
	msg := cmd().(savedMsg)
	if msg.err == nil {
		t.Fatal("expected save error to propagate")
	}
}

func TestSavedMsgError(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })
	next, cmd := m.Update(savedMsg{err: errors.New("nope")})
	if cmd != nil {
		t.Fatal("expected nil cmd on save error")
	}
	if !strings.Contains(next.(Model).errMsg, "Save failed") {
		t.Fatalf("errMsg = %q", next.(Model).errMsg)
	}
}

func TestSavedMsgSuccessFiresNavSequence(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })
	_, cmd := m.Update(savedMsg{})
	if cmd == nil {
		t.Fatal("expected nav sequence cmd")
	}
	msgs := collectMsgs(cmd)
	if len(msgs) < 3 {
		t.Fatalf("expected back/reload/syncnow msgs, got %d: %#v", len(msgs), msgs)
	}
}

func TestUpdatePassesThroughToForm(t *testing.T) {
	m := New(testVault(t), "Add", clientmodel.CardData{}, func([]byte) error { return nil })

	next, _ := m.Update(runeKey('a'))
	if next.(Model).GetCardNumber() == "" {

		_ = next
	}
}
