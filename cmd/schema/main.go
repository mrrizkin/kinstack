// Command schema writes the schema sqlc parses from the goose migrations.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"kinstack/db/migrations"
)

const schemaPath = "db/schema.sql"

func main() {
	check := flag.Bool(
		"check",
		false,
		"fail when the committed schema is stale instead of writing it",
	)
	flag.Parse()

	generated, err := migrations.Schema()
	if err != nil {
		fail(err)
	}

	if *check {
		committed, err := os.ReadFile(schemaPath)
		if err != nil {
			fail(fmt.Errorf("read %s: %w", schemaPath, err))
		}
		if string(committed) != string(generated) {
			fail(fmt.Errorf("%s is stale; run `just sqlc-generate`", schemaPath))
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o750); err != nil {
		fail(fmt.Errorf("create %s: %w", filepath.Dir(schemaPath), err))
	}
	if err := os.WriteFile(schemaPath, generated, 0o600); err != nil {
		fail(fmt.Errorf("write %s: %w", schemaPath, err))
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "schema: %v\n", err)
	os.Exit(1)
}
