package i18n

import "testing"

func TestTranslateEnglishCatalog(t *testing.T) {
	if got, want := T("auth.invalid_credentials"), "Incorrect username or password."; got != want {
		t.Fatalf("T() = %q, want %q", got, want)
	}
	if got, want := T("auth.account_locked"), "Too many sign-in attempts. Try again later."; got != want {
		t.Fatalf("T() = %q, want %q", got, want)
	}
}

func TestTranslateArguments(t *testing.T) {
	if got, want := T("users.password_too_short", 12), "Password must be at least 12 characters."; got != want {
		t.Fatalf("T() = %q, want %q", got, want)
	}
}

func TestResolveAndFallbackLocale(t *testing.T) {
	if got := Resolve("id-ID,id;q=0.9,en;q=0.8"); got != English {
		t.Fatalf("Resolve() = %q, want %q", got, English)
	}
	if got := Resolve("id-ID"); got != English {
		t.Fatalf("Resolve() fallback = %q, want %q", got, English)
	}
	if got := TLocale(Locale("fr"), "auth.invalid_credentials"); got != "Incorrect username or password." {
		t.Fatalf("TLocale() fallback = %q", got)
	}
	if got := T("missing.key"); got != "missing.key" {
		t.Fatalf("T() missing = %q, want key", got)
	}
}
