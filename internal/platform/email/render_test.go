package email

import (
	"strings"
	"testing"
)

func TestRenderCompiledTemplates(t *testing.T) {
	sender := NewMemorySender()
	for _, name := range []string{"magic-link.html", "password-reset.html", "verify-email.html"} {
		html, err := sender.Render(name, map[string]string{
			"Link":      "https://example.test/token",
			"ExpiresAt": "1 Jan 2030",
		})
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		if !strings.Contains(html, "https://example.test/token") {
			t.Fatalf("%s did not interpolate the link", name)
		}
		if !strings.Contains(html, "<!DOCTYPE") {
			t.Fatalf("%s does not look like a compiled Maizzle document", name)
		}
	}
}
