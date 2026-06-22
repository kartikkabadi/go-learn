//go:build js && wasm

package middleware

import (
	"net/http"
	"strings"
)

// sessionCookieName must match the cookie used by auth.go.
const sessionCookieName = "session"

// cacheablePaths are paths whose anonymous response is identical for all users.
var cacheablePaths = map[string]bool{
	"/":            true,
	"/lessons":     true,
	"/reference":   true,
	"/practice":    true,
	"/login":       true,
	"/signup":      true,
	"/sitemap.xml": true,
	"/robots.txt":  true,
	"/favicon.ico": true,
	"/health":      true,
}

// isCacheable returns true for anonymous GET requests to cacheable paths.
func isCacheable(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if _, err := r.Cookie(sessionCookieName); err == nil {
		return false
	}
	if !cacheablePaths[r.URL.Path] {
		if strings.HasPrefix(r.URL.Path, "/lessons/") {
			return true
		}
		return false
	}
	return true
}

// EdgeCache sets cache-control headers on anonymous GET responses so that
// Cloudflare's built-in edge cache serves repeat requests without invoking
// the Worker. Zero overhead — no Cache API calls, just headers.
// Authenticated requests get no-cache to prevent caching user-specific data.
func EdgeCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isCacheable(r) {
			w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=600")
		} else {
			w.Header().Set("Cache-Control", "private, no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
