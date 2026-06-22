package middleware

import (
	"strings"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "panic", rec, "path", r.URL.Path)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Logger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "ms", time.Since(start).Milliseconds())
	})
}

// SecurityHeaders sets recommended security headers on every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()")

		// Strict CSP: only same-origin scripts and unpkg for HTMX.
		// style-src 'unsafe-inline' is needed because template-rendered pages
		// include <style> blocks via the CSS file linked in <head>.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' https://unpkg.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"base-uri 'self'; "+
				"form-action 'self'",
		)

		next.ServeHTTP(w, r)
	})
}

// RateLimiter provides simple per-IP request rate limiting for POST endpoints.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rateBucket
	burst    int
	window   time.Duration
}

type rateBucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(burst int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rateBucket),
		burst:    burst,
		window:   window,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			ip := clientIP(r)
			rl.mu.Lock()
			b, exists := rl.visitors[ip]
			now := time.Now()
			if !exists || now.After(b.resetAt) {
				rl.visitors[ip] = &rateBucket{count: 1, resetAt: now.Add(rl.window)}
				rl.mu.Unlock()
			} else {
				b.count++
				if b.count > rl.burst {
					rl.mu.Unlock()
					slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
					http.Error(w, "too many requests", http.StatusTooManyRequests)
					return
				}
				rl.mu.Unlock()
			}
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from CF-Connecting-IP or X-Forwarded-For headers.
// Falls back to RemoteAddr when running locally without a proxy.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i > 0 {
			return fwd[:i]
		}
		return fwd
	}
	return r.RemoteAddr
}

func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, b := range rl.visitors {
		if time.Now().After(b.resetAt) {
			delete(rl.visitors, ip)
		}
	}
}

// ValidateOrigin checks the Origin header on POST requests for CSRF protection.
// In production (HTTPS), only same-origin POSTs are accepted.
func ValidateOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin != "" {
				expected := fmt.Sprintf("https://%s", r.Host)
				expectedHTTP := fmt.Sprintf("http://%s", r.Host)
				if origin != expected && origin != expectedHTTP {
					slog.Warn("origin mismatch", "origin", origin, "expected", expected, "path", r.URL.Path)
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// BodySizeLimit limits the request body to maxBytes for POST/PUT requests.
func BodySizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
