package bootstrap

import (
	"testing"

	"kinstack/config"
)

// Without object storage the slice must stay unmounted so no surface links to it.
func TestMediaFeatureDisabledWithoutStorage(t *testing.T) {
	feature := newMediaFeature(
		AppDependency{Config: config.Config{RustFS: config.RustFSConfig{Enabled: false}}},
		&appDependencies{},
		nil,
	)
	if feature != nil {
		t.Fatal("media feature must be nil when MEDIA_ENABLED is false")
	}
}
