package migrations

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Seal returns checksums.tsv with rows for unsealed migrations appended.
//
// Sealed rows are copied verbatim. A migration that already ran is history, so
// a changed file is an error whose fix is a new migration, never a reseal.
func Seal(fsys fs.FS) ([]byte, error) {
	contents, err := readMigrations(fsys)
	if err != nil {
		return nil, err
	}
	sealed, err := readChecksums(fsys)
	if err != nil {
		return nil, err
	}
	if err := verifySealed(sealed, contents); err != nil {
		return nil, err
	}
	entries, err := appendUnsealed(sealed, contents)
	if err != nil {
		return nil, err
	}
	return renderChecksums(entries), nil
}

func readMigrations(fsys fs.FS) (map[string][]byte, error) {
	paths, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("discover migrations: %w", err)
	}
	contents := make(map[string][]byte, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		body, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", name, err)
		}
		contents[name] = body
	}
	return contents, nil
}

func verifySealed(sealed []checksumEntry, contents map[string][]byte) error {
	for _, entry := range sealed {
		body, ok := contents[entry.name]
		if !ok {
			return fmt.Errorf("checksum entry for missing migration %q", entry.name)
		}
		if entry.sum != checksum(body) {
			return fmt.Errorf(
				"migration %q changed after it was sealed; add a new migration instead",
				entry.name,
			)
		}
	}
	return nil
}

func appendUnsealed(sealed []checksumEntry, contents map[string][]byte) ([]checksumEntry, error) {
	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)
	sums := make(map[string]string, len(sealed))
	for _, entry := range sealed {
		sums[entry.name] = entry.sum
	}
	entries := sealed
	for _, name := range names {
		if _, ok := sums[name]; ok {
			continue
		}
		last := len(entries) - 1
		if last >= 0 && name < entries[last].name {
			return nil, fmt.Errorf(
				"migration %q sorts before the sealed migration %q; migrations run in filename order",
				name,
				entries[last].name,
			)
		}
		entries = append(entries, checksumEntry{sum: checksum(contents[name]), name: name})
	}
	return entries, nil
}

func renderChecksums(entries []checksumEntry) []byte {
	var out strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&out, "%s  %s\n", entry.sum, entry.name)
	}
	return []byte(out.String())
}
