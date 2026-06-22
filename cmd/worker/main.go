//go:build js && wasm

// Command worker runs the go-learn HTTP server on Cloudflare Workers
// via the syumai/workers WASM runtime. Build with:
//
//	GOOS=js GOARCH=wasm go build -o build/worker.wasm ./cmd/worker
package main

import (
	"log/slog"
	"net/http"
	"os"

	"example.com/go-learn/internal/store/d1store"
	"example.com/go-learn/internal/service"
	"example.com/go-learn/internal/web/handlers"
	"example.com/go-learn/internal/web/middleware"
	"example.com/go-learn/internal/web/views"
	"github.com/syumai/workers"
)

func main() {
	renderer, err := views.New()
	if err != nil {
		slog.Error("views", "error", err)
		os.Exit(1)
	}

	// D1 database binding name from wrangler.toml
	dbName := os.Getenv("D1_DB_NAME")
	if dbName == "" {
		dbName = "go-learn-db"
	}
	st, err := d1store.Open(dbName)
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
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

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

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := middleware.Recovery(middleware.Logger(logger, mux))
	workers.Serve(handler)
}
