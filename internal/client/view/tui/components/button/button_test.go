package button

import "testing"

func TestGetLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Save", "[ Save ]"},
		{"", "[  ]"},
		{"Add card", "[ Add card ]"},
	}
	for _, c := range cases {
		if got := GetLabel(c.in); got != c.want {
			t.Errorf("GetLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
