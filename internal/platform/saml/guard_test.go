package saml

import (
	"errors"
	"net/http"
	"testing"
)

// TestParseResponseSafelyConvertsPanic exercises the guard with a nil provider,
// which is the one panic the standard library makes easy to reach. The guard
// exists for GO-2026-6909, whose exact trigger is inside the upstream parser.
func TestParseResponseSafelyConvertsPanic(t *testing.T) {
	request := &http.Request{}
	_, err := parseResponseSafely(nil, request, nil)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("nil provider: err = %v, want ErrInvalidResponse", err)
	}
}
