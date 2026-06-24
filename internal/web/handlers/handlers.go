package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
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
	LessonID  string
	Question  store.Question
	Review    bool
	Ephemeral bool // graded but not saved (anonymous)
}

type LessonBlockKind string

const (
	BlockSection  LessonBlockKind = "section"
	BlockQuestion LessonBlockKind = "question"
	BlockExercise LessonBlockKind = "exercise"
)

type LessonBlock struct {
	Kind      LessonBlockKind
	SortOrder int
	Section   *store.LessonSection
	Question  *QuestionView
	Exercise  *ExerciseView
}

type LessonPage struct {
	views.PageMeta
	Lesson     *store.Lesson
	Blocks     []LessonBlock
	Progress   LessonProgressSummary
	PrevLesson *store.Lesson
	NextLesson *store.Lesson
}

// LessonProgressSummary counts interactive checkpoints (quizzes + exercises) in a lesson.
type LessonProgressSummary struct {
	CheckpointsTotal int
	CheckpointsDone  int
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
	allLessons, err := h.Store.ListLessons()
	if err != nil {
		internalError(w, "lessons list", err)
		return
	}
	base := h.baseURL(r)
	h.Views.Render(w, "lessons_index.html", LessonsIndexPage{
		PageMeta: h.meta(r, "Go Lessons — Interactive Course — go-learn", "All Go programming lessons in order: variables, if statements, loops, functions, slices, maps, structs, pointers, and methods. Each lesson includes quizzes and exercises.", base+"/lessons", lessonsIndexJSONLD(base, allLessons)),
		Lessons:  lessons,
	})
}

func (h *Handler) LessonShow(w http.ResponseWriter, r *http.Request) {
	slugOrID := r.PathValue("slug")
	lesson, err := h.Store.GetLessonBySlug(slugOrID)
	if err != nil {
		internalError(w, "lesson show", err)
		return
	}
	if lesson == nil {
		lesson, err = h.Store.GetLesson(slugOrID)
		if err != nil {
			internalError(w, "lesson show", err)
			return
		}
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

	// Fetch sections, questions, and exercises sequentially.
	// In WASM, goroutines don't parallelize D1 calls — cooperative scheduling serializes them.
	uid := userIDFrom(r)
	sections, err := h.Store.ListLessonSections(lesson.ID)
	if err != nil {
		internalError(w, "lesson sections", err)
		return
	}
	questions, err := h.Store.ListQuestionsByLesson(uid, lesson.ID)
	if err != nil {
		internalError(w, "lesson questions", err)
		return
	}
	exercises, err := h.Store.ListExercisesByLesson(uid, lesson.ID)
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

	var exviews []ExerciseView
	for _, ex := range exercises {
		ev := ExerciseView{Exercise: ex}
		if ex.Submitted && uid != "" {
			out, ok, err := h.Store.GetExerciseSubmission(uid, ex.ID)
			if err != nil {
				internalError(w, "lesson exercise submission", err)
				return
			}
			if ok {
				ev.Output = out
			}
		}
		exviews = append(exviews, ev)
	}

	base := h.baseURL(r)
	desc := lesson.Summary
	if desc == "" {
		desc = "Go lesson: " + lesson.Title
	}

	// Compute prev/next lesson links for internal linking.
	allLessons, err := h.Store.ListLessons()
	if err != nil {
		internalError(w, "lesson list", err)
		return
	}
	var prev, next *store.Lesson
	for i, l := range allLessons {
		if l.ID == lesson.ID {
			if i > 0 {
				prev = &allLessons[i-1]
			}
			if i < len(allLessons)-1 {
				next = &allLessons[i+1]
			}
			break
		}
	}

	lessonMeta := h.meta(r, "Learn Go: "+lesson.Title+" — go-learn", desc, base+"/lessons/"+lesson.Slug, lessonJSONLD(base, lesson, questions))
	lessonMeta.HTMX = true
	lessonMeta.OgType = "article"

	var checkpointProg LessonProgressSummary
	for _, q := range questions {
		checkpointProg.CheckpointsTotal++
		if q.Answer != nil {
			checkpointProg.CheckpointsDone++
		}
	}
	for _, ex := range exercises {
		checkpointProg.CheckpointsTotal++
		if ex.Submitted {
			checkpointProg.CheckpointsDone++
		}
	}

	h.Views.Render(w, "lesson_show.html", LessonPage{
		PageMeta:   lessonMeta,
		Lesson:     lesson,
		Blocks:     mergeLessonBlocks(sections, qviews, exviews),
		Progress:   checkpointProg,
		PrevLesson: prev,
		NextLesson: next,
	})
}

func (h *Handler) AnswerQuestion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		badRequest(w, "bad form")
		return
	}
	lessonID := r.PathValue("id")
	questionID := r.PathValue("qid")
	pickedKey := strings.TrimSpace(r.FormValue("pickedKey"))
	if pickedKey == "" {
		badRequest(w, "pickedKey required")
		return
	}
	if len(pickedKey) > 200 {
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

	// Look up the label server-side from the question's options for choice
	// questions. This avoids trusting client-provided labels and avoids
	// JSON-escaping issues in hx-vals attributes.
	pickedLabel := pickedKey
	if q.QuestionType == "choice" {
		for _, opt := range q.Options {
			if opt.OptionKey == pickedKey {
				pickedLabel = opt.Label
				break
			}
		}
	}

	user := middleware.UserFromContext(r)
	view := QuestionView{
		LessonID: lessonID,
		Question: *q,
		Review:   q.SectionTag == "review",
	}

	if user == nil {
		correct, ok := evaluateAnswer(*q, pickedKey)
		if !ok {
			badRequest(w, "invalid option")
			return
		}
		view.Ephemeral = true
		view.Question.Answer = &store.Answer{
			QuestionID:  questionID,
			PickedKey:   pickedKey,
			PickedLabel: pickedLabel,
			Correct:     correct,
		}
	} else {
		answer, err := h.Store.SaveAnswer(user.ID, questionID, pickedKey, pickedLabel)
		if err != nil {
			slog.Error("save answer", "questionId", questionID, "error", err)
			http.Error(w, "invalid answer", http.StatusBadRequest)
			return
		}
		view.Question.Answer = &answer
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
		PageMeta:   h.meta(r, "Go Reference — Glossary and Resources — go-learn", "Go programming glossary: definitions for variables, functions, slices, maps, structs, pointers, and methods. Plus curated Go learning resources.", base+"/reference", glossaryJSONLD(base, terms)),
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
	practiceMeta := h.meta(r, "Practice — go-learn", "Hands-on Go programming exercises to reinforce your learning.", base+"/practice", "")
	practiceMeta.HTMX = true
	h.Views.Render(w, "practice.html", PracticePage{
		PageMeta:  practiceMeta,
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
	if path == "" {
		http.NotFound(w, r)
		return
	}
	correct := gradeExercise(path, output)

	if err := h.Store.SaveExerciseSubmission(user.ID, id, output, correct); err != nil {
		internalError(w, "submit exercise", err)
		return
	}
	h.Views.Render(w, "submit_ok.html", map[string]bool{"Correct": correct})
}

func mergeLessonBlocks(sections []store.LessonSection, questions []QuestionView, exercises []ExerciseView) []LessonBlock {
	var merged []LessonBlock
	for i := range sections {
		s := sections[i]
		merged = append(merged, LessonBlock{Kind: BlockSection, SortOrder: s.SortOrder, Section: &s})
	}
	for i := range questions {
		q := questions[i]
		merged = append(merged, LessonBlock{Kind: BlockQuestion, SortOrder: q.Question.SortOrder, Question: &q})
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].SortOrder != merged[j].SortOrder {
			return merged[i].SortOrder < merged[j].SortOrder
		}
		return merged[i].Kind == BlockSection && merged[j].Kind == BlockQuestion
	})

	exs := append([]ExerciseView(nil), exercises...)
	sort.Slice(exs, func(i, j int) bool { return exs[i].SortOrder < exs[j].SortOrder })

	blocks := append([]LessonBlock(nil), merged...)
	for i := range exs {
		ex := exs[i]
		blocks = append(blocks, LessonBlock{Kind: BlockExercise, SortOrder: ex.SortOrder, Exercise: &ex})
	}
	return blocks
}

func evaluateAnswer(q store.Question, pickedKey string) (correct, ok bool) {
	switch q.QuestionType {
	case "text":
		return strings.EqualFold(strings.TrimSpace(pickedKey), strings.TrimSpace(q.CorrectKey)), true
	case "choice":
		for _, opt := range q.Options {
			if opt.OptionKey == pickedKey {
				return opt.IsCorrect, true
			}
		}
	}
	return false, false
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

func lessonJSONLD(base string, l *store.Lesson, questions []store.Question) template.JS {
	learningResource := fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"LearningResource",
  "name":"%s",
  "description":"%s",
  "educationalLevel":"Beginner",
  "teaches":"%s",
  "inLanguage":"en",
  "url":"%s/lessons/%s"
}`, template.JSEscapeString(l.Title), template.JSEscapeString(l.Summary), template.JSEscapeString(l.Title), template.JSEscapeString(base), template.JSEscapeString(l.Slug))

	breadcrumb := fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"BreadcrumbList",
  "itemListElement":[
    {"@type":"ListItem","position":1,"name":"Home","item":"%s/"},
    {"@type":"ListItem","position":2,"name":"Lessons","item":"%s/lessons"},
    {"@type":"ListItem","position":3,"name":"%s","item":"%s/lessons/%s"}
  ]
}`, template.JSEscapeString(base), template.JSEscapeString(base), template.JSEscapeString(l.Title), template.JSEscapeString(base), template.JSEscapeString(l.Slug))

	// Build FAQPage from quiz questions — each question becomes a Q&A pair
	// that AI answer engines can extract and cite.
	var faqs []string
	for _, q := range questions {
		if q.SectionTag == "review" {
			continue
		}
		prompt := template.JSEscapeString(q.Prompt)
		// Find the correct answer label
		var answer string
		for _, opt := range q.Options {
			if opt.IsCorrect {
				answer = template.JSEscapeString(opt.Label)
				break
			}
		}
		if q.QuestionType == "text" {
			answer = template.JSEscapeString(q.CorrectKey)
		}
		if answer == "" {
			continue
		}
		faqs = append(faqs, fmt.Sprintf(`{"@type":"Question","name":"%s","acceptedAnswer":{"@type":"Answer","text":"%s"}}`, prompt, answer))
	}

	parts := []string{learningResource, breadcrumb}
	if len(faqs) > 0 {
		parts = append(parts, fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"FAQPage",
  "mainEntity":[%s]
}`, strings.Join(faqs, ",")))
	}

	return template.JS("[" + strings.Join(parts, ",") + "]")
}

func lessonsIndexJSONLD(base string, lessons []store.Lesson) template.JS {
	var items []string
	for _, l := range lessons {
		if l.Slug == "" {
			continue
		}
		items = append(items, fmt.Sprintf(`{"@type":"ListItem","position":%d,"name":"%s","url":"%s/lessons/%s"}`,
			l.SortOrder, template.JSEscapeString(l.Title), template.JSEscapeString(base), template.JSEscapeString(l.Slug)))
	}
	if len(items) == 0 {
		return ""
	}
	return template.JS(fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"ItemList",
  "name":"Go Lessons",
  "description":"Interactive Go programming lessons with quizzes and exercises.",
  "itemListElement":[%s]
}`, strings.Join(items, ",")))
}

func glossaryJSONLD(base string, terms []store.GlossaryTerm) template.JS {
	var defs []string
	for _, t := range terms {
		defs = append(defs, fmt.Sprintf(`{"@type":"DefinedTerm","name":"%s","description":"%s"}`,
			template.JSEscapeString(t.Term), template.JSEscapeString(t.Definition)))
	}
	if len(defs) == 0 {
		return ""
	}
	return template.JS(fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"DefinedTermSet",
  "name":"Go Programming Glossary",
  "url":"%s/reference",
  "hasDefinedTerm":[%s]
}`, template.JSEscapeString(base), strings.Join(defs, ",")))
}

// Sitemap generates an XML sitemap for search engines.
// Auth-only pages (/progress, /login, /signup) are excluded to conserve crawl budget.
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
  <url><loc>%s/reference</loc><changefreq>monthly</changefreq><priority>0.5</priority></url>
  <url><loc>%s/practice</loc><changefreq>weekly</changefreq><priority>0.7</priority></url>
`, base, today, base, today, base, base)
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

// RobotsTXT serves a robots.txt file for search engines and AI crawlers.
// Explicitly allows AI input (grounding/RAG) via Content-Signal.
func (h *Handler) RobotsTXT(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "User-agent: *")
	fmt.Fprintln(w, "Content-Signal: search=yes, ai-input=yes, ai-train=no")
	fmt.Fprintln(w, "Allow: /")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Sitemap: "+h.baseURL(r)+"/sitemap.xml")
}

// Favicon serves an inline SVG favicon for the classic /favicon.ico path,
// avoiding per-visit 404 noise in logs.
func (h *Handler) Favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="6" fill="#007d9c"/><text x="16" y="22" font-family="Georgia,serif" font-size="18" font-weight="bold" text-anchor="middle" fill="#fff">Go</text></svg>`)
}

// Health returns 200 if the process is alive and content is loaded.
// For D1 (production), ListLessons returns embedded content (0 D1 queries),
// so this verifies the Worker is running and content initialized — not D1 connectivity.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if _, err := h.Store.ListLessons(); err != nil {
		slog.Error("health check", "error", err)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

// NotFound renders a friendly 404 page.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	msg := "That page doesn't exist."
	meta := h.meta(r, "Not found — go-learn", msg, h.baseURL(r)+r.URL.Path, "")
	meta.NoIndex = true
	if err := h.Views.RenderPartial(w, "error.html", errorPage{
		PageMeta: meta,
		Code:     404,
		Message:  msg,
	}); err != nil {
		slog.Error("render 404", "error", err)
	}
}

type errorPage struct {
	views.PageMeta
	Code    int
	Message string
}
