package views

import (
	"context"
	"strings"
	"testing"
)

func TestScriptNonce(t *testing.T) {
	html := `<script type="module" src="/build/app.js"></script><script type="application/json">{"a":1}</script>`
	got := ScriptNonce(html, "abc123")
	if strings.Count(got, `nonce="abc123"`) != 2 {
		t.Fatalf("nonce not injected into every script: %q", got)
	}
	if ScriptNonce(html, "") != html {
		t.Fatal("empty nonce changed the HTML")
	}
}

func TestInertiaEntryUsesNonce(t *testing.T) {
	var out strings.Builder
	err := InertiaEntry(
		ScriptNonce(`<script type="module" src="/build/app.js"></script>`, "nonce-value"),
		ScriptNonce(`<script data-page="app" type="application/json">{"a":1}</script>`, "nonce-value"),
		"",
		"nonce-value",
	).Render(context.Background(), &out)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := out.String()
	if strings.Count(html, `nonce="nonce-value"`) != 3 {
		t.Fatalf("expected nonce on inline, vite, and inertia scripts: %s", html)
	}
}
