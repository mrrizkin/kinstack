package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func sealedContents(t *testing.T) map[string][]byte {
	t.Helper()
	paths, err := fs.Glob(Files, "*.sql")
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	contents := make(map[string][]byte, len(paths))
	for _, path := range paths {
		body, err := fs.ReadFile(Files, path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		contents[path] = body
	}
	return contents
}

func TestValidateAcceptsSealedMigrations(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatalf("validate migrations: %v", err)
	}
}

// An applied migration is history. Editing it would silently diverge every
// deployed database from the repository, so the checksum gate must fail.
func TestValidateChecksumsRejectsEditedMigration(t *testing.T) {
	contents := sealedContents(t)
	name := "202611140001_create_users.sql"
	edited := append([]byte("-- edited\n"), contents[name]...)
	contents[name] = edited

	err := validateChecksums(contents)
	if err == nil || !strings.Contains(err.Error(), "changed after it was sealed") {
		t.Fatalf("expected sealed-change error, got %v", err)
	}
}

func TestValidateChecksumsRejectsUnsealedMigration(t *testing.T) {
	contents := sealedContents(t)
	contents["202602010001_add_something.sql"] = []byte("-- +goose Up\n")

	err := validateChecksums(contents)
	if err == nil || !strings.Contains(err.Error(), "is not sealed") {
		t.Fatalf("expected unsealed error, got %v", err)
	}
}
