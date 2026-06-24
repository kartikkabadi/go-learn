package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kartikkabadi/go-learn/internal/app"
	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
	"github.com/kartikkabadi/go-learn/internal/web/handlers"
	"github.com/kartikkabadi/go-learn/internal/web/middleware"
	"github.com/kartikkabadi/go-learn/internal/web/views"
)

func main() {
	cfg := app.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	renderer, err := views.New()
	if err != nil {
		log.Fatal(err)
	}

	prog := &service.Progress{Store: st}
	web := &handlers.Handler{
		Store:     st,
		Progress:  prog,
		Views:     renderer,
		BaseURL:   cfg.BaseURL,
		CookieKey: cfg.CookieKey,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", cacheStatic(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Root+"/web/static")))))

	mux.HandleFunc("GET /{$}", web.Dashboard)
	mux.HandleFunc("GET /lessons", web.LessonsIndex)
	mux.HandleFunc("GET /lessons/{slug}", web.LessonShow)
	mux.HandleFunc("GET /sitemap.xml", web.Sitemap)
	mux.HandleFunc("GET /robots.txt", web.RobotsTXT)
	mux.HandleFunc("GET /favicon.ico", web.Favicon)
	mux.HandleFunc("GET /health", web.Health)
	mux.HandleFunc("POST /lessons/{id}/questions/{qid}/answer", web.AnswerQuestion)
	mux.HandleFunc("GET /reference", web.Reference)
	mux.HandleFunc("GET /practice", web.Practice)
	mux.Handle("POST /practice/{id}/submit", middleware.RequireUser(http.HandlerFunc(web.SubmitExercise)))

	// Auth routes
	mux.HandleFunc("GET /signup", web.Signup)
	mux.HandleFunc("POST /signup", web.Signup)
	mux.HandleFunc("GET /login", web.Login)
	mux.HandleFunc("POST /login", web.Login)
	mux.HandleFunc("POST /logout", web.Logout)

	// Require auth for progress page
	mux.Handle("GET /progress", middleware.RequireUser(http.HandlerFunc(web.ProgressPage)))

	mux.HandleFunc("GET /", web.NotFound)

	// Middleware stack: outermost first
	// 1. Panic recovery
	// 2. Rate limiter for POST endpoints (10 requests / 30s window)
	rateLimiter := middleware.NewRateLimiter(10, 30*time.Second)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()
	// 3. Security headers
	// 4. CSRF origin check
	// 5. Body size limit (1MB for all POST/PUT)
	// 6. Request logging
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := middleware.RequestID(
		middleware.Recovery(
			rateLimiter.Limit(
				middleware.SecurityHeaders(
					middleware.Gzip(
						middleware.LoadUser(st, cfg.CookieKey)(
							middleware.ValidateOrigin(
								middleware.BodySizeLimit(1 << 20)(
									middleware.Logger(slogLogger, mux),
								),
							),
						),
					),
				),
			),
		),
	)

	fmt.Printf("go-learn: http://%s/\n", cfg.Addr)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("server shutdown", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
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
