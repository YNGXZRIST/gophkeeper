package nav

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPush(t *testing.T) {
	msg := Push(Cards)()
	got, ok := msg.(PushMsg)
	if !ok {
		t.Fatalf("Push msg type = %T, want PushMsg", msg)
	}
	if got.ID != Cards {
		t.Errorf("ID = %v, want %v", got.ID, Cards)
	}
}

func TestPushModel(t *testing.T) {
	var m tea.Model
	msg := PushModel(m)()
	got, ok := msg.(PushModelMsg)
	if !ok {
		t.Fatalf("PushModel msg type = %T, want PushModelMsg", msg)
	}
	if got.Model != nil {
		t.Errorf("Model = %v, want nil", got.Model)
	}
}

func TestReset(t *testing.T) {
	msg := Reset(Home)()
	got, ok := msg.(ResetMsg)
	if !ok {
		t.Fatalf("Reset msg type = %T, want ResetMsg", msg)
	}
	if got.ID != Home {
		t.Errorf("ID = %v, want %v", got.ID, Home)
	}
}

func TestBack(t *testing.T) {
	if _, ok := Back()().(BackMsg); !ok {
		t.Error("Back did not produce BackMsg")
	}
}

func TestLogout(t *testing.T) {
	if _, ok := Logout()().(LogoutMsg); !ok {
		t.Error("Logout did not produce LogoutMsg")
	}
}

func TestReload(t *testing.T) {
	if _, ok := Reload()().(ReloadMsg); !ok {
		t.Error("Reload did not produce ReloadMsg")
	}
}

func TestSyncNow(t *testing.T) {
	if _, ok := SyncNow()().(SyncNowMsg); !ok {
		t.Error("SyncNow did not produce SyncNowMsg")
	}
}
