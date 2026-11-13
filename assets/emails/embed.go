// Package emails exposes the compiled Maizzle templates. dist/ is a build
// artifact; it is a subdirectory because Maizzle clears its output path.
package emails

import "embed"

// Files is embedded so the app renders what Maizzle compiled; run
// `bun x vite build` before `go build`.

//go:embed dist/*.html
var Files embed.FS
