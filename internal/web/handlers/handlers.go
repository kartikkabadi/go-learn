package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kartikkabadi/go-learn/internal/practice"
	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
	"github.com/kartikkabadi/go-learn/internal/web/middleware"
	"github.com/kartikkabadi/go-learn/internal/web/views"
)

// userIDFrom returns the authenticated user's ID, or "" if anonymous.
func userIDFrom(r *http.Request) string {
	if u := middleware.UserFromContext(r); u != nil {
		return u.ID
	}
	return ""
}

// meta builds a PageMeta with auth state from the request.
func (h *Handler) meta(r *http.Request, title, desc, canonical string, jsonld template.JS) views.PageMeta {
	m := views.PageMeta{Title: title, Description: desc, Canonical: canonical, JSONLD: jsonld}
	if u := middleware.UserFromContext(r); u != nil {
		m.Authed = true
		m.UserEmail = u.Email
	}
	return m
}

// Handler wires together the store, service layer, and template renderer for web routes.
type Handler struct {
	Store     store.Store
	Progress  *service.Progress
	Views     *views.Renderer
	BaseURL   string // canonical site URL; empty = derive per-request
	CookieKey []byte // HMAC key for signing session cookies
}

type DashboardPage struct {
	views.PageMeta
	Data service.Dashboard
}

type LessonsIndexPage struct {
	views.PageMeta
	Lessons []store.LessonProgress
}

type QuestionView struct {
	LessonID string
	Question store.Question
	Review   bool
}

type LessonPage struct {
	views.PageMeta
	Lesson        *store.Lesson
	Sections      []store.LessonSection
	QuestionViews []QuestionView
	Exercises     []store.Exercise
}

type ProgressPage struct {
	views.PageMeta
	Answers []store.AnswerRow
}

type ReferencePage struct {
	views.PageMeta
	Terms      []store.GlossaryTerm
	References []store.Reference
}

type ExerciseView struct {
	store.Exercise
	Output string
}

type PracticePage struct {
	views.PageMeta
	Exercises []ExerciseView
}

// HTTP error helpers

func internalError(w http.ResponseWriter, msg string, err error) {
	slog.Error(msg, "error", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Anonymous visitors see a landing page — no personal data.
	if middleware.UserFromContext(r) == nil {
		h.renderLanding(w, r)
		return
	}

	userID := userIDFrom(r)
	data, err := h.Progress.Dashboard(userID)
	if err != nil {
		internalError(w, "dashboard", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "dashboard.html", DashboardPage{
		PageMeta: h.meta(r, "Home — go-learn", "Go learning workspace with interactive lessons, quizzes, and practice exercises.", base+"/", courseJSONLD(base, h.Store)),
		Data:     data,
	})
}

// LandingPageData holds the data for the anonymous landing page.
type LandingPageData struct {
	views.PageMeta
	Lessons []store.Lesson
	Mission *store.Mission
}

func (h *Handler) renderLanding(w http.ResponseWriter, r *http.Request) {
	lessons, err := h.Store.ListLessons()
	if err != nil {
		internalError(w, "landing lessons", err)
		return
	}
	mission, err := h.Store.GetMission()
	if err != nil {
		internalError(w, "landing mission", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "landing.html", LandingPageData{
		PageMeta: h.meta(r, "Learn Go interactively — go-learn", "A free, interactive Go programming course with lessons, quizzes, and hands-on exercises. Learn at your own pace and track your progress.", base+"/", courseJSONLD(base, h.Store)),
		Lessons:  lessons,
		Mission:  mission,
	})
}

func (h *Handler) LessonsIndex(w http.ResponseWriter, r *http.Request) {
	lessons, err := h.Store.LessonProgress(userIDFrom(r))
	if err != nil {
		internalError(w, "lessons index", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "lessons_index.html", LessonsIndexPage{
		PageMeta: h.meta(r, "Lessons — go-learn", "Browse all Go lessons with progress tracking and interactive quizzes.", base+"/lessons", ""),
		Lessons:  lessons,
	})
}

func (h *Handler) LessonShow(w http.ResponseWriter, r *http.Request) {
	slugOrID := r.PathValue("slug")
	lesson, err := h.Store.GetLessonBySlug(slugOrID)
	if lesson == nil {
		lesson, err = h.Store.GetLesson(slugOrID)
	}
	if err != nil {
		internalError(w, "lesson show", err)
		return
	}
	if lesson == nil {
		http.NotFound(w, r)
		return
	}

	// Redirect numeric ID to slug for canonical URLs
	if lesson.Slug != "" && slugOrID != lesson.Slug {
		base := h.baseURL(r)
		http.Redirect(w, r, base+"/lessons/"+lesson.Slug, http.StatusMovedPermanently)
		return
	}

	sections, err := h.Store.ListLessonSections(lesson.ID)
	if err != nil {
		internalError(w, "lesson sections", err)
		return
	}
	questions, err := h.Store.ListQuestionsByLesson(userIDFrom(r), lesson.ID)
	if err != nil {
		internalError(w, "lesson questions", err)
		return
	}
	exercises, err := h.Store.ListExercisesByLesson(userIDFrom(r), lesson.ID)
	if err != nil {
		internalError(w, "lesson exercises", err)
		return
	}

	var qviews []QuestionView
	for _, q := range questions {
		qviews = append(qviews, QuestionView{
			LessonID: lesson.ID,
			Question: q,
			Review:   q.SectionTag == "review",
		})
	}

	base := h.baseURL(r)
	desc := lesson.Summary
	if desc == "" {
		desc = "Go lesson: " + lesson.Title
	}
	h.Views.Render(w, "lesson_show.html", LessonPage{
		PageMeta:      h.meta(r, "Learn Go: "+lesson.Title+" — go-learn", desc, base+"/lessons/"+lesson.Slug, lessonJSONLD(base, lesson)),
		Lesson:        lesson,
		Sections:      sections,
		QuestionViews: qviews,
		Exercises:     exercises,
	})
}

func (h *Handler) AnswerQuestion(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r)
	if user == nil {
		http.Error(w, "log in to save answers", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	lessonID := r.PathValue("id")
	questionID := r.PathValue("qid")
	pickedKey := strings.TrimSpace(r.FormValue("pickedKey"))
	pickedLabel := strings.TrimSpace(r.FormValue("pickedLabel"))
	if pickedLabel == "" {
		pickedLabel = pickedKey
	}
	if pickedKey == "" {
		badRequest(w, "pickedKey required")
		return
	}
	if len(pickedKey) > 200 || len(pickedLabel) > 500 {
		badRequest(w, "value too long")
		return
	}

	// Verify the question exists and belongs to the lesson in the URL path.
	q, err := h.Store.GetQuestion(questionID)
	if err != nil {
		internalError(w, "lookup question", err)
		return
	}
	if q == nil || q.LessonID != lessonID {
		http.NotFound(w, r)
		return
	}

	answer, err := h.Store.SaveAnswer(user.ID, questionID, pickedKey, pickedLabel)
	if err != nil {
		slog.Error("save answer", "questionId", questionID, "error", err)
		http.Error(w, "invalid answer", http.StatusBadRequest)
		return
	}
	q.Answer = &answer

	view := QuestionView{
		LessonID: lessonID,
		Question: *q,
		Review:   q.SectionTag == "review",
	}
	if err := h.Views.RenderPartial(w, "quiz_answer_partial", view); err != nil {
		internalError(w, "answer partial", err)
	}
}

func (h *Handler) ProgressPage(w http.ResponseWriter, r *http.Request) {
	answers, err := h.Store.ListAnswers(userIDFrom(r))
	if err != nil {
		internalError(w, "progress page", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "progress.html", ProgressPage{
		PageMeta: h.meta(r, "Progress — go-learn", "Track your quiz answers and learning progress in Go.", base+"/progress", ""),
		Answers:  answers,
	})
}

func (h *Handler) Reference(w http.ResponseWriter, r *http.Request) {
	terms, err := h.Store.ListGlossaryTerms()
	if err != nil {
		internalError(w, "reference terms", err)
		return
	}
	refs, err := h.Store.ListReferences()
	if err != nil {
		internalError(w, "reference refs", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "reference.html", ReferencePage{
		PageMeta:   h.meta(r, "Reference — go-learn", "Go glossary of terms and external learning resources.", base+"/reference", ""),
		Terms:      terms,
		References: refs,
	})
}

func (h *Handler) Practice(w http.ResponseWriter, r *http.Request) {
	userID := userIDFrom(r)
	exercises, err := h.Store.ListExercises(userID)
	if err != nil {
		internalError(w, "practice list", err)
		return
	}
	var evs []ExerciseView
	for _, ex := range exercises {
		ev := ExerciseView{Exercise: ex}
		if ex.Submitted {
			out, ok, err := h.Store.GetExerciseSubmission(userID, ex.ID)
			if err != nil {
				internalError(w, "practice submission", err)
				return
			}
			if ok {
				ev.Output = out
			}
		}
		evs = append(evs, ev)
	}
	base := h.baseURL(r)
	h.Views.Render(w, "practice.html", PracticePage{
		PageMeta:  h.meta(r, "Practice — go-learn", "Hands-on Go programming exercises to reinforce your learning.", base+"/practice", ""),
		Exercises: evs,
	})
}

func (h *Handler) SubmitExercise(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r)
	if user == nil {
		http.Error(w, "log in to submit exercises", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	id := r.PathValue("id")
	output := r.FormValue("output")
	if strings.TrimSpace(output) == "" {
		badRequest(w, "output required")
		return
	}
	if len(output) > 100000 {
		badRequest(w, "output too long")
		return
	}

	exercises, err := h.Store.ListExercises(user.ID)
	if err != nil {
		internalError(w, "list exercises", err)
		return
	}
	var path string
	for _, ex := range exercises {
		if ex.ID == id {
			path = ex.Path
			break
		}
	}
	correct := gradeExercise(path, output)

	if err := h.Store.SaveExerciseSubmission(user.ID, id, output, correct); err != nil {
		internalError(w, "submit exercise", err)
		return
	}
	h.Views.Render(w, "submit_ok.html", map[string]bool{"Correct": correct})
}

// gradeExercise compares submitted output against the embedded expected output
// for a practice module. Trailing whitespace is ignored. No expected file =>
// submission is marked correct (no grading for that exercise).
func gradeExercise(path, output string) bool {
	expected, ok := practice.ExpectedOutput(path)
	if !ok {
		return true
	}
	return expected == strings.TrimSpace(output)
}

// --- SEO helpers ---

// baseURL returns the canonical site URL. Uses h.BaseURL if set (production),
// otherwise derives scheme from X-Forwarded-Proto/TLS and host from the request.
func (h *Handler) baseURL(r *http.Request) string {
	if h.BaseURL != "" {
		return strings.TrimRight(h.BaseURL, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return scheme + "://" + r.Host
}

func courseJSONLD(base string, s store.Store) template.JS {
	lessons, err := s.ListLessons()
	if err != nil {
		return ""
	}
	var itemList []string
	for _, l := range lessons {
		if l.Slug == "" {
			continue
		}
		slug := template.JSEscapeString(l.Slug)
		itemList = append(itemList, fmt.Sprintf(`{"@type":"ListItem","position":%d,"item":"%s/lessons/%s"}`, l.SortOrder, template.JSEscapeString(base), slug))
	}
	items := strings.Join(itemList, ",")
	js := fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"Course",
  "name":"Learn Go",
  "description":"An interactive Go programming course with lessons, quizzes, and exercises.",
  "educationalLevel":"Beginner",
  "teaches":"Go programming language",
  "numberOfLessons":%d,
  "hasCourseInstance":{
    "@type":"CourseInstance",
    "courseMode":"self-paced",
    "inLanguage":"en"
  },
  "itemListElement":[%s]
}`, len(lessons), items)
	return template.JS(js)
}

func lessonJSONLD(base string, l *store.Lesson) template.JS {
	js := fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"LearningResource",
  "name":"%s",
  "description":"%s",
  "educationalLevel":"Beginner",
  "teaches":"%s",
  "inLanguage":"en",
  "url":"%s/lessons/%s"
}`, template.JSEscapeString(l.Title), template.JSEscapeString(l.Summary), template.JSEscapeString(l.Title), template.JSEscapeString(base), template.JSEscapeString(l.Slug))
	return template.JS(js)
}

// Sitemap generates an XML sitemap for search engines.
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	base := h.baseURL(r)
	lessons, err := h.Store.ListLessons()
	if err != nil {
		internalError(w, "sitemap", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>1.0</priority></url>
  <url><loc>%s/lessons</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.9</priority></url>
  <url><loc>%s/progress</loc><changefreq>daily</changefreq><priority>0.3</priority></url>
  <url><loc>%s/reference</loc><changefreq>monthly</changefreq><priority>0.5</priority></url>
  <url><loc>%s/practice</loc><changefreq>weekly</changefreq><priority>0.7</priority></url>
`, base, today, base, today, base, base, base)
	for _, l := range lessons {
		if l.Slug == "" {
			continue
		}
		u := base + "/lessons/" + l.Slug
		fmt.Fprintf(w, `  <url><loc>%s</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>0.8</priority></url>
`, u, today)
	}
	fmt.Fprint(w, `</urlset>
`)
}

// RobotsTXT serves a robots.txt file for search engines.
func (h *Handler) RobotsTXT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "User-agent: *")
	fmt.Fprintln(w, "Allow: /")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Sitemap: "+h.baseURL(r)+"/sitemap.xml")
}

// Favicon serves an inline SVG favicon as a 204 for the classic /favicon.ico path,
// avoiding per-visit 404 noise in logs.
func (h *Handler) Favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="6" fill="#007d9c"/><text x="16" y="22" font-family="Georgia,serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#fff">Go</text></svg>`)
}

// Health returns 200 if the process is alive and the DB is reachable.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if _, err := h.Store.ListLessons(); err != nil {
		slog.Error("health check db probe", "error", err)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

// NotFound renders a friendly 404 page.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	h.Views.Render(w, "error.html", errorPage{
		PageMeta: views.PageMeta{Title: "Not found — go-learn", Canonical: h.baseURL(r) + r.URL.Path},
		Code:     404,
		Message:  "That page doesn't exist.",
	})
}

type errorPage struct {
	views.PageMeta
	Code    int
	Message string
}
