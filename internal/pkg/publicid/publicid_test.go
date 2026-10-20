package publicid

import (
	"regexp"
	"testing"
)

func TestIsValidRecognizesStandardNanoIDs(t *testing.T) {
	valid := "A1_b2C3-d4E5f6G7h8I9j"
	if !IsValid(valid) {
		t.Fatalf("expected valid Nano ID: %q", valid)
	}
	for _, invalid := range []string{"usr_" + valid, valid[:20], valid + "x", "contains space 123456789"} {
		if IsValid(invalid) {
			t.Fatalf("expected invalid Nano ID: %q", invalid)
		}
	}
}

func TestNewUsesPureNanoIDURLSafePayload(t *testing.T) {
	pattern := regexp.MustCompile(`^[-_A-Za-z0-9]{21}$`)
	seen := make(map[string]struct{}, 100)
	for range 100 {
		id, err := New()
		if err != nil {
			t.Fatalf("generate public ID: %v", err)
		}
		if !pattern.MatchString(id) {
			t.Fatalf("unexpected Nano ID format: %q", id)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate public ID: %q", id)
		}
		seen[id] = struct{}{}
	}
}
