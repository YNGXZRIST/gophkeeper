package layout

import (
	"strings"

	"gophkeeper/internal/client/view/tui/components/theme"
)

func Page(title, body, hint string) string {
	var b strings.Builder
	b.WriteString(theme.Title.Render(title))
	b.WriteString("\n\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteRune('\n')
	}
	b.WriteString("\n")
	b.WriteString(theme.Blurred.Render(hint))
	return b.String()
}
