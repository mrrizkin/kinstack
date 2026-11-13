package migrations

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ChecksumFile pins every migration to the bytes that were reviewed. A
// migration that already ran must never change; a correction is a new
// migration. `just migrations-seal` writes the file when one is added.
const ChecksumFile = "checksums.tsv"

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
		if expected != checksum(contents[name]) {
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

type checksumEntry struct {
	sum  string
	name string
}

// parseChecksums requires sorted, unique rows so a hand-edited file is visible
// rather than silently collapsed into a map.
func parseChecksums(raw []byte) ([]checksumEntry, error) {
	entries := make([]checksumEntry, 0, 32)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || len(fields[0]) != sha256.Size*2 {
			return nil, fmt.Errorf("migration checksum line %q must be <sha256> <name>", line)
		}
		entry := checksumEntry{sum: fields[0], name: fields[1]}
		if previous := len(entries) - 1; previous >= 0 && entries[previous].name >= entry.name {
			return nil, fmt.Errorf(
				"migration checksum rows must be unique and sorted by name: %q after %q",
				entry.name,
				entries[previous].name,
			)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func readChecksums(fsys fs.FS) ([]checksumEntry, error) {
	raw, err := fs.ReadFile(fsys, ChecksumFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read migration checksums: %w", err)
	}
	return parseChecksums(raw)
}

func readSealedChecksums() (map[string]string, error) {
	entries, err := readChecksums(Files)
	if err != nil {
		return nil, err
	}
	sealed := make(map[string]string, len(entries))
	for _, entry := range entries {
		sealed[entry.name] = entry.sum
	}
	return sealed, nil
}

func checksum(body []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(body))
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
