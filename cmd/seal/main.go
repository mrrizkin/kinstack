// Command seal appends checksums for new migrations to checksums.tsv.
// Sealed rows are never rewritten, so editing an applied migration still fails
// the gate instead of being silenced by a reseal.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"kinstack/db/migrations"
)

const dir = "db/migrations"

func main() {
	flag.Parse()

	sealed, err := migrations.Seal(os.DirFS(dir))
	if err != nil {
		fail(err)
	}

	path := filepath.Join(dir, migrations.ChecksumFile)
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fail(fmt.Errorf("read %s: %w", path, err))
	}
	if bytes.Equal(current, sealed) {
		fmt.Println("seal: every migration is already sealed")
		return
	}
	if err := os.WriteFile(path, sealed, 0o600); err != nil {
		fail(fmt.Errorf("write %s: %w", path, err))
	}
	fmt.Println("seal: appended new migration checksums")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "seal: %v\n", err)
	os.Exit(1)
}
