package inertia

import "testing"

func TestBuildVersionPrefersInjectedValue(t *testing.T) {
	previous := buildVersion
	buildVersion = "release-abc123"
	defer func() { buildVersion = previous }()

	if got := BuildVersion(); got != "release-abc123" {
		t.Fatalf("injected version must win, got %q", got)
	}
}

func TestBuildVersionIsStableAndNonEmpty(t *testing.T) {
	first := BuildVersion()
	if first == "" {
		t.Fatal("build version must not be empty")
	}
	if second := BuildVersion(); second != first {
		t.Fatalf("build version must be stable within a process: %q != %q", first, second)
	}
	if len(first) > 20 {
		t.Fatalf("build version must stay short, got %q", first)
	}
}
