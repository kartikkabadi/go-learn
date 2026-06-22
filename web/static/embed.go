// Package staticassets provides an embedded filesystem for serving static files
// (CSS, JS, images) in environments without a real filesystem (e.g. Cloudflare Workers WASM).
package staticassets

import "embed"

//go:embed theme.css
var FS embed.FS
