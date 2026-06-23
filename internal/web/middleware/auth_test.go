package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/web/middleware"
)

func TestRequireUser_HTMXRedirect(t *testing.T) {
	var called bool
	handler := middleware.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/practice/ex1/submit", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Fatal("handler should not run for anonymous HTMX request")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
	if got := w.Header().Get("HX-Redirect"); got != "/login?next=%2Fpractice%2Fex1%2Fsubmit" {
		t.Fatalf("unexpected HX-Redirect: %q", got)
	}
}

func TestRequireUser_FullPageRedirect(t *testing.T) {
	handler := middleware.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/progress", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/login?next=%2Fprogress" {
		t.Fatalf("unexpected Location: %q", got)
	}
}
