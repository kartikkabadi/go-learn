// Package practice holds expected outputs for hands-on exercises.
// Outputs are embedded so grading works in environments without a filesystem
// (e.g. Cloudflare Workers WASM).
package practice

import (
	"embed"
	"strings"
)

//go:embed expected/*.txt
var expectedFS embed.FS

// ExpectedOutput returns the trimmed expected stdout for a practice module
// path (e.g. "practice/hello") and whether an expected file exists.
// Missing file => ok=false; callers should treat that as "no grading".
func ExpectedOutput(path string) (string, bool) {
	name := strings.TrimPrefix(path, "practice/") + ".txt"
	b, err := expectedFS.ReadFile("expected/" + name)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}
