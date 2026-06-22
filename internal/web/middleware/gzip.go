package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

// gzipableTypes are content types that benefit from gzip compression.
var gzipableTypes = map[string]bool{
	"text/html":                    true,
	"text/css":                     true,
	"text/plain":                   true,
	"application/javascript":       true,
	"text/javascript":              true,
	"application/json":             true,
	"application/xml":              true,
	"application/atom+xml":         true,
	"application/rss+xml":          true,
	"application/manifest+json":    true,
	"image/svg+xml":                true,
}

// gzipWriter buffers the response, then gzips it if the content type is
// compressible and the response is large enough.
// ponytail: buffers whole response in memory — fine for small HTML pages,
// would need streaming for large responses.
type gzipWriter struct {
	http.ResponseWriter
	buf       []byte
	status    int
	threshold int
}

func (g *gzipWriter) WriteHeader(code int) {
	g.status = code
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	g.buf = append(g.buf, b...)
	return len(b), nil
}

// flush decides whether to gzip and writes the final response.
func (g *gzipWriter) flush(acceptsGzip bool) {
	ct := g.ResponseWriter.Header().Get("Content-Type")
	ctBase := strings.TrimSpace(strings.ToLower(strings.SplitN(ct, ";", 2)[0]))

	if acceptsGzip && len(g.buf) >= g.threshold && gzipableTypes[ctBase] &&
		g.status != http.StatusNoContent && g.status != http.StatusNotModified {
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Del("Content-Length")
		g.ResponseWriter.WriteHeader(g.status)
		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(g.ResponseWriter)
		defer gz.Close()
		gz.Write(g.buf)
		return
	}
	g.ResponseWriter.WriteHeader(g.status)
	g.ResponseWriter.Write(g.buf)
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// Gzip compresses responses for text-based content types when the client
// sends Accept-Encoding: gzip. Responses smaller than 1KB are not compressed
// (overhead exceeds savings). Must be placed after SecurityHeaders so it can
// inspect the final Content-Type.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			threshold:      1024,
		}

		next.ServeHTTP(gw, r)
		gw.flush(true)
	})
}
