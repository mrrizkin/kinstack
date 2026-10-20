package email

import (
	"bytes"
	"html/template"

	"kinstack/assets/emails"
)

// [[ ]] instead of {{ }} because Vue already owns {{ }} at build time.
const (
	runtimeDelimLeft  = "[["
	runtimeDelimRight = "]]"
)

// templates holds every Maizzle-compiled HTML template, parsed once at init.
var templates = template.Must(
	template.New("emails").
		Delims(runtimeDelimLeft, runtimeDelimRight).
		ParseFS(emails.Files, "dist/*.html"),
)

// renderer gives Sender implementations Render for free by executing an
// embedded, pre-parsed Maizzle template (e.g. "magic-link.html").
type renderer struct{}

func (renderer) Render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
