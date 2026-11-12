package pagination

import (
	"errors"
	"testing"
)

// ParseRequest is the shared boundary for hand-written list links, so it must
// survive arbitrary input: either a safe Query or ErrInvalidRequest.
func FuzzParseRequest(f *testing.F) {
	f.Add("1", "10")
	f.Add("", "")
	f.Add("0", "-1")
	f.Add("not-a-number", "100")
	f.Add("9223372036854775807", "100")
	f.Fuzz(func(t *testing.T, rawPage, rawPerPage string) {
		query, err := ParseRequest(rawPage, rawPerPage, DefaultPerPage)
		if err != nil {
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		if query.Page < 1 || query.Page > PageHintMax {
			t.Fatalf("page %d out of bounds", query.Page)
		}
		if query.PerPage < 1 || query.PerPage > PerPageMax {
			t.Fatalf("per page %d out of bounds", query.PerPage)
		}
		if query.Offset() < 0 {
			t.Fatalf("offset %d must not be negative", query.Offset())
		}
	})
}
