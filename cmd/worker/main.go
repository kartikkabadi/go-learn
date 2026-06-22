//go:build js && wasm

package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/web/handlers"
	"github.com/kartikkabadi/go-learn/internal/web/middleware"
	"github.com/kartikkabadi/go-learn/internal/web/views"
	"github.com/kartikkabadi/go-learn/internal/store/d1store"
	"github.com/kartikkabadi/go-learn/web/static"
	"github.com/syumai/workers"
)

func main() {
	renderer, err := views.New()
	if err != nil {
		slog.Error("views", "error", err)
		os.Exit(1)
	}

	st, err := d1store.Open("DB")
	if err != nil {
		slog.Error("d1store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := d1store.Migrate(st.DB()); err != nil {
		slog.Error("d1 migrate", "error", err)
		os.Exit(1)
	}

	prog := &service.Progress{Store: st}
	web := &handlers.Handler{
		Store:    st,
		Progress: prog,
		Views:    renderer,
		BaseURL:  os.Getenv("CANONICAL_BASE"),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", cacheStatic(http.StripPrefix("/static/", http.FileServer(http.FS(staticassets.FS)))))
	mux.HandleFunc("GET /{$}", web.Dashboard)
	mux.HandleFunc("GET /lessons", web.LessonsIndex)
	mux.HandleFunc("GET /lessons/{slug}", web.LessonShow)
	mux.HandleFunc("POST /lessons/{id}/questions/{qid}/answer", web.AnswerQuestion)
	mux.Handle("GET /progress", middleware.RequireUser(http.HandlerFunc(web.ProgressPage)))
	mux.HandleFunc("GET /reference", web.Reference)
	mux.HandleFunc("GET /practice", web.Practice)
	mux.Handle("POST /practice/{id}/submit", middleware.RequireUser(http.HandlerFunc(web.SubmitExercise)))
	mux.HandleFunc("GET /", web.NotFound)
	mux.HandleFunc("GET /sitemap.xml", web.Sitemap)
	mux.HandleFunc("GET /robots.txt", web.RobotsTXT)
	mux.HandleFunc("GET /favicon.ico", web.Favicon)
	mux.HandleFunc("GET /health", web.Health)
	mux.HandleFunc("GET /signup", web.Signup)
	mux.HandleFunc("POST /signup", web.Signup)
	mux.HandleFunc("GET /login", web.Login)
	mux.HandleFunc("POST /login", web.Login)
	mux.HandleFunc("POST /logout", web.Logout)

	// Middleware stack
	rateLimiter := middleware.NewRateLimiter(10, 30*time.Second)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// ponytail: no gzip middleware in Workers — Cloudflare auto-gzips text responses.
	// The buffering gzip middleware breaks the WASM runtime (deferred writes don't work).
	handler := middleware.RequestID(
		middleware.Recovery(
			rateLimiter.Limit(
				middleware.SecurityHeaders(
					middleware.LoadUser(st)(
						middleware.ValidateOrigin(
							middleware.BodySizeLimit(1<<20)(
								middleware.Logger(logger, mux),
							),
						),
					),
				),
			),
		),
	)
	workers.Serve(handler)
}

// cacheStatic wraps a static file server with cache headers.
// Immutable files (htmx.min.js) get far-future caching; CSS gets 24h must-revalidate.
func cacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
		}
		h.ServeHTTP(w, r)
	})
}
