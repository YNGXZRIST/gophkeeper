package paginatedlist

import (
	"errors"
	"strings"
	"testing"

	"gophkeeper/internal/client/view/tui/components/nav"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type item struct {
	id    string
	name  string
	valid bool
}

func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func baseCfg() Config[item] {
	return Config[item]{
		Title:        "Items",
		Noun:         "item",
		Header:       "ID  NAME",
		AddScreen:    nav.Cards,
		ID:           func(it item) string { return it.id },
		Revealable:   func(it item) bool { return it.valid },
		RenderItem:   func(it item) string { return it.name },
		RenderDetail: func(it item) string { return "detail:" + it.name },
	}
}

func mk(cfg Config[item]) model[item] {
	return New(cfg).(model[item])
}

func loaded(m model[item], items []item, hasNext bool) model[item] {
	msg := loadedMsg[item]{items: items, hasNext: hasNext}
	if len(items) > 0 {
		msg.nextCursor = items[len(items)-1].id
	}
	next, _ := m.Update(msg)
	return next.(model[item])
}

func sample() []item {
	return []item{
		{id: "1", name: "alpha", valid: true},
		{id: "2", name: "beta", valid: true},
		{id: "3", name: "broken", valid: false},
	}
}

func TestInitReturnsCmd(t *testing.T) {
	m := mk(baseCfg())
	if m.Init() == nil {
		t.Error("Init should return a batch cmd")
	}
}

func TestFetchNilFetch(t *testing.T) {
	m := mk(baseCfg())
	msg := m.fetch("cur")()
	lm, ok := msg.(loadedMsg[item])
	if !ok {
		t.Fatalf("msg type = %T", msg)
	}
	if lm.cursor != "cur" {
		t.Errorf("cursor = %q, want cur", lm.cursor)
	}
}

func TestFetchSuccessAndError(t *testing.T) {
	cfg := baseCfg()
	cfg.Fetch = func(cursor string, limit int) ([]item, error) {
		return sample(), nil
	}
	m := mk(cfg)
	msg := m.fetch("")().(loadedMsg[item])
	if len(msg.items) != 3 {
		t.Errorf("items = %d, want 3", len(msg.items))
	}
	if msg.nextCursor != "3" {
		t.Errorf("nextCursor = %q, want 3", msg.nextCursor)
	}

	cfg.Fetch = func(cursor string, limit int) ([]item, error) {
		return nil, errors.New("boom")
	}
	m = mk(cfg)
	if msg := m.fetch("")().(loadedMsg[item]); msg.err == nil {
		t.Error("expected fetch error")
	}
}

func TestLoadedSuccess(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), true)
	if m.loading {
		t.Error("loading should be false after loadedMsg")
	}
	if len(m.items) != 3 {
		t.Errorf("items = %d, want 3", len(m.items))
	}
	if !m.hasNext {
		t.Error("hasNext should be true")
	}
}

func TestLoadedError(t *testing.T) {
	m := mk(baseCfg())
	next, _ := m.Update(loadedMsg[item]{err: errors.New("x")})
	mm := next.(model[item])
	if mm.errMsg == "" {
		t.Error("errMsg should be set on load error")
	}
}

func TestViewLoading(t *testing.T) {
	m := mk(baseCfg())
	out := m.View().Content
	if !strings.Contains(out, "Loading") {
		t.Errorf("loading view missing Loading: %q", out)
	}
}

func TestViewEmpty(t *testing.T) {
	m := loaded(mk(baseCfg()), nil, false)
	out := m.View().Content
	if !strings.Contains(out, "No items") {
		t.Errorf("empty view missing message: %q", out)
	}
}

func TestViewItemsAndBrokenAndReveal(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	out := m.View().Content
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "decryption failed") {
		t.Errorf("view missing rendered/broken items: %q", out)
	}

	m2, _ := m.Update(specialKey(tea.KeyEnter))
	out = m2.(model[item]).View().Content
	if !strings.Contains(out, "detail:alpha") {
		t.Errorf("reveal view missing detail: %q", out)
	}
}

func TestNavigateUpDownResetsReveal(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	m2, _ := m.Update(specialKey(tea.KeyEnter))
	mm := m2.(model[item])
	if !mm.revealed {
		t.Fatal("expected revealed")
	}
	m3, _ := mm.Update(specialKey(tea.KeyDown))
	mm = m3.(model[item])
	if mm.revealed {
		t.Error("down should reset revealed")
	}
	if mm.selected != 1 {
		t.Errorf("selected = %d, want 1", mm.selected)
	}

	m4, _ := mm.Update(specialKey(tea.KeyUp))
	if m4.(model[item]).selected != 0 {
		t.Error("up should move to 0")
	}
}

func TestEsc(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("esc should yield cmd")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Error("esc not BackMsg")
	}
}

func TestAddKey(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	_, cmd := m.Update(runeKey('a'))
	if cmd == nil {
		t.Fatal("a should push add screen")
	}
	if _, ok := cmd().(nav.PushMsg); !ok {
		t.Error("a not PushMsg")
	}
}

func TestConflictKey(t *testing.T) {
	cfg := baseCfg()
	cfg.ConflictScreen = nav.CardSync
	m := loaded(mk(cfg), sample(), false)
	_, cmd := m.Update(runeKey('c'))
	if cmd == nil {
		t.Fatal("c should push conflict screen")
	}
	if _, ok := cmd().(nav.PushMsg); !ok {
		t.Error("c not PushMsg")
	}

	m2 := loaded(mk(baseCfg()), sample(), false)
	if _, cmd := m2.Update(runeKey('c')); cmd != nil {
		t.Error("c with zero ConflictScreen should be nil")
	}
}

func TestEditKey(t *testing.T) {
	cfg := baseCfg()
	cfg.NewEdit = func(it item) tea.Model { return mk(baseCfg()) }
	m := loaded(mk(cfg), sample(), false)
	_, cmd := m.Update(runeKey('e'))
	if cmd == nil {
		t.Fatal("e should push edit model")
	}
	if _, ok := cmd().(nav.PushModelMsg); !ok {
		t.Error("e not PushModelMsg")
	}
}

func TestDownloadKeyAndHint(t *testing.T) {
	cfg := baseCfg()
	cfg.NewDownload = func(it item) tea.Model { return mk(baseCfg()) }
	m := loaded(mk(cfg), sample(), false)
	if !strings.Contains(m.View().Content, "download") {
		t.Error("hint should mention download when NewDownload set")
	}
	_, cmd := m.Update(runeKey('s'))
	if cmd == nil {
		t.Fatal("s should push download model")
	}
	if _, ok := cmd().(nav.PushModelMsg); !ok {
		t.Error("s not PushModelMsg")
	}
}

func TestDeleteConfirmYes(t *testing.T) {
	removed := ""
	cfg := baseCfg()
	cfg.Remove = func(id string) error { removed = id; return nil }
	m := loaded(mk(cfg), sample(), false)
	m2, _ := m.Update(runeKey('d'))
	mm := m2.(model[item])
	if !mm.confirming {
		t.Fatal("d should enter confirming")
	}
	if !strings.Contains(mm.View().Content, "delete selected item?") {
		t.Error("confirm hint missing")
	}
	m3, cmd := mm.Update(runeKey('y'))
	if m3.(model[item]).confirming {
		t.Error("y should clear confirming")
	}
	if cmd == nil {
		t.Fatal("y should yield remove cmd")
	}
	msg := cmd()
	if _, ok := msg.(deletedMsg); !ok {
		t.Errorf("remove cmd msg = %T, want deletedMsg", msg)
	}
	if removed != "1" {
		t.Errorf("removed id = %q, want 1", removed)
	}
}

func TestDeleteConfirmNo(t *testing.T) {
	cfg := baseCfg()
	cfg.Remove = func(id string) error { return nil }
	m := loaded(mk(cfg), sample(), false)
	m2, _ := m.Update(runeKey('d'))
	m3, cmd := m2.(model[item]).Update(runeKey('n'))
	if m3.(model[item]).confirming {
		t.Error("n should clear confirming")
	}
	if cmd != nil {
		t.Error("n should yield nil cmd")
	}
}

func TestDeleteConfirmYesNilID(t *testing.T) {
	cfg := baseCfg()
	cfg.ID = nil
	cfg.Remove = func(id string) error { return nil }
	m := loaded(mk(cfg), sample(), false)
	m2, _ := m.Update(runeKey('d'))
	_, cmd := m2.(model[item]).Update(runeKey('y'))
	if cmd != nil {
		t.Error("y with nil ID should yield nil cmd")
	}
}

func TestRemoveCmdError(t *testing.T) {
	cfg := baseCfg()
	cfg.Remove = func(id string) error { return errors.New("nope") }
	m := loaded(mk(cfg), sample(), false)
	msg := m.remove("1")()
	dm, ok := msg.(deletedMsg)
	if !ok || dm.err == nil {
		t.Errorf("remove error not propagated: %#v", msg)
	}
}

func TestRemoveCmdNilRemove(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	msg := m.remove("1")()
	if dm, ok := msg.(deletedMsg); !ok || dm.err != nil {
		t.Errorf("nil Remove should give empty deletedMsg, got %#v", msg)
	}
}

func TestDeletedMsgSuccess(t *testing.T) {
	cfg := baseCfg()
	cfg.Fetch = func(c string, l int) ([]item, error) { return sample(), nil }
	m := loaded(mk(cfg), sample(), false)
	_, cmd := m.Update(deletedMsg{})
	if cmd == nil {
		t.Error("successful delete should trigger refetch+sync")
	}
}

func TestDeletedMsgError(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	next, _ := m.Update(deletedMsg{err: errors.New("x")})
	if next.(model[item]).errMsg == "" {
		t.Error("delete error should set errMsg")
	}
}

func TestPagingRightLeft(t *testing.T) {
	cfg := baseCfg()
	cfg.Fetch = func(c string, l int) ([]item, error) { return sample(), nil }
	m := loaded(mk(cfg), sample(), true)
	m2, cmd := m.Update(specialKey(tea.KeyRight))
	mm := m2.(model[item])
	if !mm.loading {
		t.Error("right should set loading")
	}
	if len(mm.history) != 1 {
		t.Errorf("history len = %d, want 1", len(mm.history))
	}
	if cmd == nil {
		t.Error("right should yield fetch cmd")
	}

	m3, cmd := mm.Update(specialKey(tea.KeyLeft))
	mm = m3.(model[item])
	if len(mm.history) != 0 {
		t.Errorf("history len = %d, want 0 after left", len(mm.history))
	}
	if cmd == nil {
		t.Error("left should yield fetch cmd")
	}
}

func TestPagingRightNoNext(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	m2, cmd := m.Update(specialKey(tea.KeyRight))
	if cmd != nil {
		t.Error("right with no next should be no-op")
	}
	if m2.(model[item]).loading {
		t.Error("right with no next should not set loading")
	}
}

func TestPagingLeftNoHistory(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	_, cmd := m.Update(specialKey(tea.KeyLeft))
	if cmd != nil {
		t.Error("left with no history should be no-op")
	}
}

func TestReloadMsg(t *testing.T) {
	cfg := baseCfg()
	cfg.Fetch = func(c string, l int) ([]item, error) { return sample(), nil }
	m := loaded(mk(cfg), sample(), false)
	m2, cmd := m.Update(nav.ReloadMsg{})
	if !m2.(model[item]).loading {
		t.Error("reload should set loading")
	}
	if cmd == nil {
		t.Error("reload should yield cmd")
	}
}

func TestSpinnerTickWhileLoading(t *testing.T) {
	m := mk(baseCfg())
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("tick while loading should yield spinner cmd")
	}
}

func TestSpinnerTickNotLoading(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd != nil {
		t.Error("tick when not loading should yield nil cmd")
	}
}

func TestRevealNonRevealable(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)

	m2, _ := m.Update(specialKey(tea.KeyDown))
	m3, _ := m2.(model[item]).Update(specialKey(tea.KeyDown))
	m4, _ := m3.(model[item]).Update(specialKey(tea.KeyEnter))
	if m4.(model[item]).revealed {
		t.Error("non-revealable item should not reveal")
	}
}

func TestKeysOnEmptyListNoPanic(t *testing.T) {
	m := loaded(mk(baseCfg()), nil, false)

	for _, k := range []tea.KeyPressMsg{runeKey('e'), runeKey('d'), specialKey(tea.KeyEnter)} {
		if _, cmd := m.Update(k); cmd != nil {
			t.Errorf("key %v on empty list should be nil cmd", k.String())
		}
	}
}

func TestFetcher(t *testing.T) {
	list := func(cursor string, limit int) ([]int, error) {
		return []int{1, 2, 3}, nil
	}
	decode := func(n int) (string, error) {
		if n == 2 {
			return "", errors.New("bad")
		}
		return "ok", nil
	}
	fallback := func(n int) string { return "fb" }
	f := Fetcher(list, decode, fallback)
	items, err := f("", 10)
	if err != nil {
		t.Fatalf("Fetcher err: %v", err)
	}
	if len(items) != 3 || items[1] != "fb" {
		t.Errorf("Fetcher items = %v, want fallback at idx 1", items)
	}

	ferr := Fetcher(
		func(c string, l int) ([]int, error) { return nil, errors.New("x") },
		decode, fallback,
	)
	if _, err := ferr("", 10); err == nil {
		t.Error("Fetcher should propagate list error")
	}
}

func TestNonKeyNonHandledMsg(t *testing.T) {
	m := loaded(mk(baseCfg()), sample(), false)
	if _, cmd := m.Update("random"); cmd != nil {
		t.Error("unhandled msg should yield nil cmd")
	}
}
