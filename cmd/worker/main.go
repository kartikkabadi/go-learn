//go:build js && wasm

package main

import (
	"log/slog"
	"net/http"
	"os"
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

	prog := &service.Progress{Store: st}
	web := &handlers.Handler{
		Store:    st,
		Progress: prog,
		Views:    renderer,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticassets.FS))))
	mux.HandleFunc("GET /{$}", web.Dashboard)
	mux.HandleFunc("GET /lessons", web.LessonsIndex)
	mux.HandleFunc("GET /lessons/{slug}", web.LessonShow)
	mux.HandleFunc("POST /lessons/{id}/questions/{qid}/answer", web.AnswerQuestion)
	mux.HandleFunc("GET /progress", web.ProgressPage)
	mux.HandleFunc("GET /reference", web.Reference)
	mux.HandleFunc("GET /practice", web.Practice)
	mux.HandleFunc("POST /practice/{id}/submit", web.SubmitExercise)
	mux.HandleFunc("GET /sitemap.xml", web.Sitemap)
	mux.HandleFunc("GET /robots.txt", web.RobotsTXT)

	// Middleware stack
	rateLimiter := middleware.NewRateLimiter(10, 30*time.Second)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rateLimiter.Cleanup()
		}
	}()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := middleware.Recovery(
		rateLimiter.Limit(
			middleware.SecurityHeaders(
				middleware.ValidateOrigin(
					middleware.BodySizeLimit(1<<20)(
						middleware.Logger(logger, mux),
					),
				),
			),
		),
	)
	workers.Serve(handler)
}
