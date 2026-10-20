package views

import "strings"

// ScriptNonce injects a CSP nonce attribute into every script tag in an HTML
// fragment produced outside templ (Vite tags and the Inertia page-data script).
func ScriptNonce(html, nonce string) string {
	if html == "" || nonce == "" {
		return html
	}
	return strings.ReplaceAll(html, "<script", `<script nonce="`+nonce+`"`)
}
