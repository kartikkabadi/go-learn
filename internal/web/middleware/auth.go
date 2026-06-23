package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/kartikkabadi/go-learn/internal/store"
	"github.com/kartikkabadi/go-learn/internal/web/cookies"
)

type contextKey string

const userKey contextKey = "user"

// UserFromContext returns the authenticated user for the request, or nil.
func UserFromContext(r *http.Request) *store.User {
	if u, ok := r.Context().Value(userKey).(*store.User); ok {
		return u
	}
	return nil
}

// LoadUser looks up the signed session cookie and attaches the user to the request context.
// It runs on every request; unmatched/missing/invalid sessions are silently treated as anonymous.
func LoadUser(s store.Store, key []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("session")
			if err == nil && c.Value != "" {
				token, err := cookies.Verify(c.Value, key)
				if err != nil {
					slog.Debug("invalid session cookie", "error", err)
				} else {
					sess, err := s.GetSession(token)
					if err != nil {
						slog.Error("get session", "error", err)
					} else if sess != nil {
						user, err := s.GetUserByID(sess.UserID)
						if err != nil {
							slog.Error("get user by id", "error", err)
						} else if user != nil {
							r = r.WithContext(context.WithValue(r.Context(), userKey, user))
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireUser rejects requests with no authenticated user, redirecting to /login.
// HTMX requests get HX-Redirect instead of a full-page redirect body swapped into the target.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r) == nil {
			login := "/login?next=" + url.QueryEscape(r.URL.RequestURI())
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", login)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, login, http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithUser attaches a user to the request context. Intended for tests.
func WithUser(r *http.Request, u *store.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, u))
}
