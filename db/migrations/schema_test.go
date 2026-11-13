package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestSchemaFileIsCurrent(t *testing.T) {
	generated, err := Schema()
	if err != nil {
		t.Fatalf("render schema: %v", err)
	}
	committed, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatalf("read db/schema.sql: %v", err)
	}
	if string(committed) != string(generated) {
		t.Fatalf("db/schema.sql does not match the migrations; run just sqlc-generate")
	}
}

func TestSchemaKeepsOnlyUpSections(t *testing.T) {
	schema, err := Schema()
	if err != nil {
		t.Fatalf("render schema: %v", err)
	}
	body := string(schema)
	if !strings.Contains(body, "CREATE TABLE users") {
		t.Fatalf("schema is missing the users table")
	}
	// 202601010020 recreates magic_link_tokens in its Down section only.
	if strings.Contains(body, "CREATE TABLE magic_link_tokens") {
		t.Fatalf("schema includes a Down section")
	}
}

func TestUpSectionRejectsBlockDirectives(t *testing.T) {
	_, err := upSection(
		"-- +goose Up\n-- +goose StatementBegin\nSELECT 1;\n-- +goose StatementEnd\n",
	)
	if err == nil {
		t.Fatalf("expected an unsupported directive error")
	}
}
