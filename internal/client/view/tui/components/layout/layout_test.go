package layout

import (
	"strings"
	"testing"
)

func TestPageContainsParts(t *testing.T) {
	out := Page("My Title", "the body", "the hint")
	for _, part := range []string{"My Title", "the body", "the hint"} {
		if !strings.Contains(out, part) {
			t.Errorf("Page output missing %q: %q", part, out)
		}
	}
}

func TestPageBodyWithTrailingNewline(t *testing.T) {

	withNL := Page("T", "body\n", "h")
	withoutNL := Page("T", "body", "h")
	if withNL == "" || withoutNL == "" {
		t.Fatal("Page returned empty string")
	}
	if !strings.Contains(withNL, "body") || !strings.Contains(withoutNL, "body") {
		t.Error("Page dropped body text")
	}
}
