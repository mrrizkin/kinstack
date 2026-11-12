package apitoken

import (
	"strings"
	"testing"
)

// FuzzSplitToken checks the parser that stands between a raw bearer header and
// a repository lookup. Anything it accepts must rebuild the input exactly, so a
// token can never be authenticated from a string it did not come from.
func FuzzSplitToken(f *testing.F) {
	f.Add(tokenScheme + "abc123.secret")
	f.Add(tokenScheme + "abc123.")
	f.Add(tokenScheme + ".secret")
	f.Add(tokenScheme + "abc123")
	f.Add("bearer abc123.secret")
	f.Add("")
	f.Add(tokenScheme + "a.b.c")

	f.Fuzz(func(t *testing.T, raw string) {
		publicID, secret, ok := splitToken(raw)
		if !ok {
			return
		}
		if publicID == "" || secret == "" {
			t.Fatalf("splitToken(%q) accepted empty part: id=%q secret=%q", raw, publicID, secret)
		}
		if strings.Contains(publicID, ".") {
			t.Fatalf("splitToken(%q) left a dot in the public id: %q", raw, publicID)
		}
		rebuilt := tokenScheme + publicID + "." + secret
		if rebuilt != strings.TrimSpace(raw) {
			t.Fatalf("splitToken(%q) rebuilt %q", raw, rebuilt)
		}
	})
}
