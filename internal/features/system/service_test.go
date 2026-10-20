package system

import (
	"testing"

	"kinstack/internal/pkg/log"
)

func TestPageReturnsCanonicalSystemContract(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	props := service.Page("system-help")
	if props.Surface != "system-help" || len(props.Topics) == 0 || len(props.Shortcuts) == 0 {
		t.Fatalf("unexpected system page props: %#v", props)
	}
}

func TestFeatureUsesExplicitMemoryRepositoryAndLogger(t *testing.T) {
	feature := New(Dependency{Repository: NewMemoryRepository(), Logger: log.NewNoopLogger()})
	props := feature.service.Page("system-help")
	if props.Surface != "system-help" || len(props.Shortcuts) != 3 {
		t.Fatalf("unexpected test feature props: %#v", props)
	}
}
