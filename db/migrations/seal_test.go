package migrations

import (
	"strings"
	"testing"
	"testing/fstest"
)

const sealedMigration = "-- +goose Up\nCREATE TABLE users ();\n-- +goose Down\nDROP TABLE users;\n"

func sealFixture() fstest.MapFS {
	return fstest.MapFS{
		"202601010001_create_users.sql": {Data: []byte(sealedMigration)},
		ChecksumFile: {
			Data: []byte(checksum([]byte(sealedMigration)) + "  202601010001_create_users.sql\n"),
		},
	}
}

func TestSealAppendsNewMigrations(t *testing.T) {
	fsys := sealFixture()
	fsys["202601010002_add_users_email.sql"] = &fstest.MapFile{
		Data: []byte("-- +goose Up\nSELECT 1;\n"),
	}

	sealed, err := Seal(fsys)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	lines := strings.Fields(strings.TrimSpace(string(sealed)))
	if len(lines) != 4 {
		t.Fatalf("expected two rows, got %q", sealed)
	}
	if lines[2] != checksum([]byte("-- +goose Up\nSELECT 1;\n")) ||
		lines[3] != "202601010002_add_users_email.sql" {
		t.Fatalf("new migration was not appended in order: %q", sealed)
	}
}

// The point of the tool: a reseal must not launder an edit to an applied
// migration, otherwise the checksum gate is only a speed bump.
func TestSealRejectsChangedSealedMigration(t *testing.T) {
	fsys := sealFixture()
	fsys["202601010001_create_users.sql"] = &fstest.MapFile{
		Data: []byte("-- +goose Up\nDROP TABLE users;\n"),
	}

	_, err := Seal(fsys)
	if err == nil || !strings.Contains(err.Error(), "changed after it was sealed") {
		t.Fatalf("expected sealed-change error, got %v", err)
	}
}

func TestSealRejectsRemovedMigration(t *testing.T) {
	fsys := sealFixture()
	delete(fsys, "202601010001_create_users.sql")

	_, err := Seal(fsys)
	if err == nil || !strings.Contains(err.Error(), "missing migration") {
		t.Fatalf("expected missing-migration error, got %v", err)
	}
}

func TestSealRejectsMigrationThatSortsBeforeASealedOne(t *testing.T) {
	fsys := sealFixture()
	fsys["202501010001_backdated.sql"] = &fstest.MapFile{Data: []byte("-- +goose Up\nSELECT 1;\n")}

	_, err := Seal(fsys)
	if err == nil || !strings.Contains(err.Error(), "sorts before the sealed migration") {
		t.Fatalf("expected out-of-order error, got %v", err)
	}
}

func TestSealIsIdempotent(t *testing.T) {
	fsys := sealFixture()
	first, err := Seal(fsys)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	fsys[ChecksumFile] = &fstest.MapFile{Data: first}
	second, err := Seal(fsys)
	if err != nil {
		t.Fatalf("resal: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("seal is not idempotent:\n%s\n%s", first, second)
	}
}

func TestParseChecksumsRejectsUnsortedOrDuplicateRows(t *testing.T) {
	sum := checksum([]byte(sealedMigration))
	for name, raw := range map[string]string{
		"unsorted":  sum + "  b.sql\n" + sum + "  a.sql\n",
		"duplicate": sum + "  a.sql\n" + sum + "  a.sql\n",
		"malformed": "not-a-checksum  a.sql\n",
	} {
		if _, err := parseChecksums([]byte(raw)); err == nil {
			t.Fatalf("%s: expected a parse error", name)
		}
	}
}
