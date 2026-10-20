// Package migrations exposes the canonical Goose migration filesystem.
package migrations

import "embed"

// Files is embedded so application and migration commands use the same schema.
//
//go:embed *.sql
var Files embed.FS
