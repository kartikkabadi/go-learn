package views

import (
	"embed"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// PageMeta carries SEO metadata and auth state for every full-page render.
type PageMeta struct {
	Title       string
	Description string
	Canonical   string
	JSONLD      template.JS
	OgType      string // og:type — default "website" in template
	NoIndex     bool
	HTMX        bool // load htmx.min.js
	Authed      bool
	UserEmail   string
}

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

// Renderer wraps html/template with common template functions and embedded template files.
type Renderer struct {
	templates *template.Template
}

// New creates a Renderer by parsing all embedded HTML templates.
func New() (*Renderer, error) {
	funcs := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"percent": func(done, total int) int {
			if total == 0 {
				return 0
			}
			return (done * 100) / total
		},
		"accuracy": func(correct, answered int) int {
			if answered == 0 {
				return 0
			}
			return (correct * 100) / answered
		},
		"lessonTotal": func(q, e int) int {
			return q + e
		},
		"lessonDone": func(qa, ed int) int {
			return qa + ed
		},
		"origin": func(rawurl string) string {
			u, err := url.Parse(rawurl)
			if err != nil || u.Host == "" {
				return ""
			}
			return u.Scheme + "://" + u.Host
		},
	}
	t, err := template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: t}, nil
}

// Render writes a full-page template to the response writer.
func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render template", "name", name, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// RenderPartial executes a named template and writes to an arbitrary writer (used for HTMX partials).
func (r *Renderer) RenderPartial(w io.Writer, name string, data any) error {
	return r.templates.ExecuteTemplate(w, name, data)
}
