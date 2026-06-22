//go:build js && wasm

package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/syumai/workers/cloudflare/cache"
)

// sessionCookieName must match the cookie used by auth.go.
const sessionCookieName = "session"

// cacheablePaths are paths whose anonymous response is identical for all users.
var cacheablePaths = map[string]bool{
	"/":              true,
	"/lessons":       true,
	"/reference":     true,
	"/practice":      true,
	"/login":         true,
	"/signup":        true,
	"/sitemap.xml":   true,
	"/robots.txt":    true,
	"/favicon.ico":   true,
	"/health":        true,
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
		// Lesson pages are cacheable for anonymous users (no quiz answers shown).
		if strings.HasPrefix(r.URL.Path, "/lessons/") {
			return true
		}
		return false
	}
	return true
}

// responseRecorder captures the response body and headers for caching.
type responseRecorder struct {
	http.ResponseWriter
	buf     bytes.Buffer
	status  int
	header  http.Header
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		header:         http.Header{},
		status:         http.StatusOK,
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	// Copy headers to the recorder.
	for k, v := range r.ResponseWriter.Header() {
		r.header[k] = v
	}
	r.ResponseWriter.WriteHeader(code)
}

// EdgeCache wraps a handler with Cloudflare Cache API for anonymous GET requests.
// Cached responses are served from the edge (sub-millisecond). Authenticated
// requests and non-cacheable paths bypass the cache entirely.
func EdgeCache(next http.Handler) http.Handler {
	c := cache.New()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isCacheable(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Check edge cache.
		cached, err := c.Match(r, &cache.MatchOptions{IgnoreMethod: true})
		if err == nil && cached != nil {
			// Copy cached headers.
			for k, v := range cached.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(cached.StatusCode)
			io.Copy(w, cached.Body)
			cached.Body.Close()
			return
		}

		// Cache miss: render and cache.
		rec := newResponseRecorder(w)
		next.ServeHTTP(rec, r)

		// Only cache successful responses.
		if rec.status != http.StatusOK || rec.buf.Len() == 0 {
			return
		}

		// Build a response for the cache.
		cachedResp := &http.Response{
			StatusCode: rec.status,
			Header:     rec.header,
			Body:       io.NopCloser(bytes.NewReader(rec.buf.Bytes())),
		}
		if cachedResp.Header == nil {
			cachedResp.Header = w.Header()
		}
		cachedResp.Header.Set("Cache-Control", "public, max-age=300, stale-while-revalidate=600")
		_ = c.Put(r, cachedResp)
	})
}
