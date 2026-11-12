package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// envReaderNames are the helpers in config/env.go plus os.Getenv. A key read
// through any of them must be documented in .env.example.
var envReaderNames = map[string]struct{}{
	"env":         {},
	"envInt":      {},
	"envBool":     {},
	"envDuration": {},
	"envList":     {},
	"envFloat":    {},
	"Getenv":      {},
}

// envKeyPattern excludes dynamic keys such as loadSocial's prefix+"_ENABLED";
// those are documented by the caller's literal block in .env.example.
var envKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func documentedEnvKeys(t *testing.T) map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", ".env.example"))
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	documented := map[string]struct{}{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, _, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || key == "" || strings.HasPrefix(key, "#") {
			continue
		}
		documented[key] = struct{}{}
	}
	return documented
}

func configEnvKeys(t *testing.T) map[string]struct{} {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob config sources: %v", err)
	}
	keys := map[string]struct{}{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if key, ok := literalEnvKey(call); ok {
				keys[key] = struct{}{}
			}
			return true
		})
	}
	return keys
}

func literalEnvKey(call *ast.CallExpr) (string, bool) {
	var name string
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		name = fun.Name
	case *ast.SelectorExpr:
		if pkg, ok := fun.X.(*ast.Ident); ok && pkg.Name == "os" {
			name = fun.Sel.Name
		}
	}
	if _, ok := envReaderNames[name]; !ok || len(call.Args) == 0 {
		return "", false
	}
	argument, ok := call.Args[0].(*ast.BasicLit)
	if !ok || argument.Kind != token.STRING {
		return "", false
	}
	key, err := strconv.Unquote(argument.Value)
	if err != nil || !envKeyPattern.MatchString(key) {
		return "", false
	}
	return key, true
}

// TestEnvExampleDocumentsEveryConfigKey keeps the template honest: a key the
// configuration package reads but .env.example never mentions is a setting that
// silently falls back to its default in every deployment built from the file.
func TestEnvExampleDocumentsEveryConfigKey(t *testing.T) {
	documented := documentedEnvKeys(t)
	var missing []string
	for key := range configEnvKeys(t) {
		if _, ok := documented[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf(".env.example does not document: %s", strings.Join(missing, ", "))
	}
}
