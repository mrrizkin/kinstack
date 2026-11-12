package migrations

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// checksumFile pins every migration to the bytes that were reviewed. A
// migration that already ran must never change; a correction is a new
// migration. `just migrations-seal` writes the file when one is added.
const checksumFile = "checksums.tsv"

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
	contents := make(map[string][]byte, len(paths))
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
		body, err := fs.ReadFile(Files, path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		contents[name] = body
		if err := validateMigrationSQL(name, migrationName, string(body)); err != nil {
			return err
		}
	}
	return validateChecksums(contents)
}

// validateChecksums enforces immutability: an applied migration is history, so
// editing it would silently diverge every deployed database from the repo.
func validateChecksums(contents map[string][]byte) error {
	sealed, err := readSealedChecksums()
	if err != nil {
		return err
	}
	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		expected, ok := sealed[name]
		if !ok {
			return fmt.Errorf("migration %q is not sealed; run just migrations-seal", name)
		}
		if expected != fmt.Sprintf("%x", sha256.Sum256(contents[name])) {
			return fmt.Errorf(
				"migration %q changed after it was sealed; add a new migration instead",
				name,
			)
		}
	}
	for name := range sealed {
		if _, ok := contents[name]; !ok {
			return fmt.Errorf("checksum entry for missing migration %q", name)
		}
	}
	return nil
}

func readSealedChecksums() (map[string]string, error) {
	raw, err := fs.ReadFile(Files, checksumFile)
	if err != nil {
		return nil, fmt.Errorf("read migration checksums: %w", err)
	}
	sealed := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("migration checksum line %q must be <sha256> <name>", line)
		}
		sealed[fields[1]] = fields[0]
	}
	return sealed, nil
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
