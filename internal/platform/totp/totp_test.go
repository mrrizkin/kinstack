package totp

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndValidateRoundTrip(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	at := time.Unix(1_700_000_000, 0).UTC()

	code, err := GenerateCode(secret, at)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if len(code) != Digits {
		t.Fatalf("expected %d digits, got %q", Digits, code)
	}
	if !Validate(secret, code, at) {
		t.Fatalf("expected current code to validate")
	}
	if !Validate(secret, code, at.Add(Period*time.Second)) {
		t.Fatalf("expected one step of clock skew to validate")
	}
	if Validate(secret, code, at.Add(3*Period*time.Second)) {
		t.Fatalf("expected far-future code to be rejected")
	}
	if Validate(secret, "000000", at) && code != "000000" {
		t.Fatalf("expected wrong code to be rejected")
	}
	if Validate(secret, "12345", at) {
		t.Fatalf("expected short code to be rejected")
	}
}

func TestProvisioningURI(t *testing.T) {
	uri := ProvisioningURI("abcd2345", "pemilik", "Contoh Aplikasi")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Fatalf("unexpected uri: %s", uri)
	}
	for _, fragment := range []string{"secret=ABCD2345", "issuer=Contoh+Aplikasi", "digits=6", "period=30"} {
		if !strings.Contains(uri, fragment) {
			t.Fatalf("uri %s missing %q", uri, fragment)
		}
	}
}

func TestValidateStepReportsMatchedStep(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	at := time.Unix(1_700_000_000, 0).UTC()
	code, err := GenerateCode(secret, at)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	step, ok := ValidateStep(secret, code, at)
	if !ok {
		t.Fatal("expected the current code to validate")
	}
	if want := at.Unix() / Period; step != want {
		t.Fatalf("expected step %d, got %d", want, step)
	}

	skewed, ok := ValidateStep(secret, code, at.Add(Period*time.Second))
	if !ok || skewed != step {
		t.Fatalf("expected the same step for one step of skew, step=%d ok=%v", skewed, ok)
	}

	if _, ok := ValidateStep(secret, "000000", at); ok && code != "000000" {
		t.Fatal("expected a wrong code to be rejected")
	}
}
