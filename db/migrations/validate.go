package migrations

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var migrationNamePattern = regexp.MustCompile(`^(\d{8})(\d{4})_([a-z0-9]+(?:_[a-z0-9]+)*)\.sql$`)

// Validate checks the compatibility metadata required by the migration
// runner. It deliberately does not execute SQL; the opt-in database contract
// test performs the up/down round trip against a dedicated PostgreSQL URL.
func Validate() error {
	paths, err := fs.Glob(Files, "*.sql")
	if err != nil {
		return fmt.Errorf("discover migrations: %w", err)
	}
	sort.Strings(paths)
	seen := make(map[int64]string, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		version, migrationName, err := parseMigrationName(name)
		if err != nil {
			return err
		}
		if previous, ok := seen[version]; ok {
			return fmt.Errorf(
				"migration version %d is duplicated by %q and %q",
				version,
				previous,
				name,
			)
		}
		seen[version] = name
		contents, err := fs.ReadFile(Files, path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		if err := validateMigrationSQL(name, migrationName, string(contents)); err != nil {
			return err
		}
	}
	return nil
}

func parseMigrationName(name string) (int64, string, error) {
	match := migrationNamePattern.FindStringSubmatch(name)
	if match == nil {
		return 0, "", fmt.Errorf(
			"migration %q must use YYYYMMDDOOOO_<lowercase_snake_case>.sql",
			name,
		)
	}
	if _, err := time.Parse("20060102", match[1]); err != nil {
		return 0, "", fmt.Errorf("migration %q has an invalid YYYYMMDD date", name)
	}
	order, err := strconv.Atoi(match[2])
	if err != nil || order < 1 {
		return 0, "", fmt.Errorf("migration %q has an invalid four-digit order", name)
	}
	version, err := strconv.ParseInt(match[1]+match[2], 10, 64)
	if err != nil || version <= 0 {
		return 0, "", fmt.Errorf("migration %q has an invalid version", name)
	}
	return version, match[3], nil
}

func validateMigrationSQL(filename, _ string, contents string) error {
	up := strings.Index(contents, "-- +goose Up")
	down := strings.Index(contents, "-- +goose Down")
	if up < 0 || down < 0 || down < up {
		return fmt.Errorf("migration %q must contain ordered Goose Up and Down sections", filename)
	}
	return nil
}
