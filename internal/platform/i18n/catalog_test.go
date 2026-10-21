package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var translationCall = regexp.MustCompile(`i18n\.T\("([a-z0-9_.]+)"`)

// TestCatalogCoversEveryLiteralCall guards against flash messages and errors
// that silently render as their raw key because nobody added the entry.
func TestCatalogCoversEveryLiteralCall(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	catalog := catalogs[Default()]

	missing := make(map[string]string)
	for _, dir := range []string{"internal", "config", "cmd"} {
		base := filepath.Join(root, dir)
		walkErr := filepath.WalkDir(
			base,
			func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || filepath.Ext(path) != ".go" {
					return nil
				}
				source, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				for _, match := range translationCall.FindAllStringSubmatch(string(source), -1) {
					key := match[1]
					if _, ok := catalog[key]; !ok {
						relative, _ := filepath.Rel(root, path)
						missing[key] = relative
					}
				}
				return nil
			},
		)
		if walkErr != nil {
			t.Fatalf("scan %s: %v", dir, walkErr)
		}
	}
	for key, file := range missing {
		t.Errorf("missing catalog entry %q used in %s", key, file)
	}
}
