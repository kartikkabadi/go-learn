package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"example.com/go-learn/internal/service"
	"example.com/go-learn/internal/store"
	"example.com/go-learn/internal/web/views"
)

// Handler wires together the store, service layer, and template renderer for web routes.
type Handler struct {
	Store    store.Store
	Progress *service.Progress
	Views    *views.Renderer
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
	Lesson         *store.Lesson
	Sections       []store.LessonSection
	QuestionViews  []QuestionView
	Exercises      []store.Exercise
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
	data, err := h.Progress.Dashboard()
	if err != nil {
		internalError(w, "dashboard", err)
		return
	}
	base := baseURL(r)
	h.Views.Render(w, "dashboard.html", DashboardPage{
		PageMeta: views.PageMeta{
			Title:       "Home — go-learn",
			Description: "Go learning workspace with interactive lessons, quizzes, and practice exercises.",
			Canonical:   base + "/",
			JSONLD:      courseJSONLD(h.Store),
		},
		Data: data,
	})
}

func (h *Handler) LessonsIndex(w http.ResponseWriter, r *http.Request) {
	lessons, err := h.Store.LessonProgress()
	if err != nil {
		internalError(w, "lessons index", err)
		return
	}
	base := baseURL(r)
	h.Views.Render(w, "lessons_index.html", LessonsIndexPage{
		PageMeta: views.PageMeta{
			Title:       "Lessons — go-learn",
			Description: "Browse all Go lessons with progress tracking and interactive quizzes.",
			Canonical:   base + "/lessons",
		},
		Lessons: lessons,
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
		base := baseURL(r)
		http.Redirect(w, r, base+"/lessons/"+lesson.Slug, http.StatusMovedPermanently)
		return
	}

	sections, err := h.Store.ListLessonSections(lesson.ID)
	if err != nil {
		internalError(w, "lesson sections", err)
		return
	}
	questions, err := h.Store.ListQuestionsByLesson(lesson.ID)
	if err != nil {
		internalError(w, "lesson questions", err)
		return
	}
	exercises, err := h.Store.ListExercisesByLesson(lesson.ID)
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

	base := baseURL(r)
	desc := lesson.Summary
	if desc == "" {
		desc = "Go lesson: " + lesson.Title
	}
	h.Views.Render(w, "lesson_show.html", LessonPage{
		PageMeta: views.PageMeta{
			Title:       "Learn Go: " + lesson.Title + " — go-learn",
			Description: desc,
			Canonical:   base + "/lessons/" + lesson.Slug,
			JSONLD:      lessonJSONLD(lesson),
		},
		Lesson:        lesson,
		Sections:      sections,
		QuestionViews: qviews,
		Exercises:     exercises,
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
	pickedLabel := strings.TrimSpace(r.FormValue("pickedLabel"))
	if pickedLabel == "" {
		pickedLabel = pickedKey
	}
	if pickedKey == "" {
		badRequest(w, "pickedKey required")
		return
	}

	answer, err := h.Store.SaveAnswer(questionID, pickedKey, pickedLabel)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	q, err := h.Store.GetQuestion(questionID)
	if err != nil || q == nil {
		http.Error(w, "question not found", http.StatusNotFound)
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
	answers, err := h.Store.ListAnswers()
	if err != nil {
		internalError(w, "progress page", err)
		return
	}
	base := baseURL(r)
	h.Views.Render(w, "progress.html", ProgressPage{
		PageMeta: views.PageMeta{
			Title:       "Progress — go-learn",
			Description: "Track your quiz answers and learning progress in Go.",
			Canonical:   base + "/progress",
		},
		Answers: answers,
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
	base := baseURL(r)
	h.Views.Render(w, "reference.html", ReferencePage{
		PageMeta: views.PageMeta{
			Title:       "Reference — go-learn",
			Description: "Go glossary of terms and external learning resources.",
			Canonical:   base + "/reference",
		},
		Terms: terms,
		References: refs,
	})
}

func (h *Handler) Practice(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.Store.ListExercises()
	if err != nil {
		internalError(w, "practice list", err)
		return
	}
	var evs []ExerciseView
	for _, ex := range exercises {
		ev := ExerciseView{Exercise: ex}
		if ex.Submitted {
			out, ok, err := h.Store.GetExerciseSubmission(ex.ID)
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
	base := baseURL(r)
	h.Views.Render(w, "practice.html", PracticePage{
		PageMeta: views.PageMeta{
			Title:       "Practice — go-learn",
			Description: "Hands-on Go programming exercises to reinforce your learning.",
			Canonical:   base + "/practice",
		},
		Exercises: evs,
	})
}

func (h *Handler) SubmitExercise(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Store.SaveExerciseSubmission(id, output); err != nil {
		internalError(w, "submit exercise", err)
		return
	}
	h.Views.Render(w, "submit_ok.html", nil)
}

// --- SEO helpers ---

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func courseJSONLD(s store.Store) template.JS {
	lessons, err := s.ListLessons()
	if err != nil {
		return ""
	}
	var itemList []string
	for _, l := range lessons {
		itemList = append(itemList, fmt.Sprintf(`{"@type":"ListItem","position":%d,"item":"https://go-learn.example.com/lessons/%s"}`, l.SortOrder, l.Slug))
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

func lessonJSONLD(l *store.Lesson) template.JS {
	js := fmt.Sprintf(`{
  "@context":"https://schema.org",
  "@type":"LearningResource",
  "name":"%s",
  "description":"%s",
  "educationalLevel":"Beginner",
  "teaches":"%s",
  "inLanguage":"en"
}`, template.JSEscapeString(l.Title), template.JSEscapeString(l.Summary), template.JSEscapeString(l.Title))
	return template.JS(js)
}

// Sitemap generates an XML sitemap for search engines.
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	base := "https://go-learn.example.com"
	lessons, err := h.Store.ListLessons()
	if err != nil {
		internalError(w, "sitemap", err)
		return
	}
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc><changefreq>weekly</changefreq><priority>1.0</priority></url>
  <url><loc>%s/lessons</loc><changefreq>weekly</changefreq><priority>0.9</priority></url>
  <url><loc>%s/progress</loc><changefreq>daily</changefreq><priority>0.3</priority></url>
  <url><loc>%s/reference</loc><changefreq>monthly</changefreq><priority>0.5</priority></url>
  <url><loc>%s/practice</loc><changefreq>weekly</changefreq><priority>0.7</priority></url>
`, base, base, base, base, base)
	for _, l := range lessons {
		u := base + "/lessons/" + l.Slug
		fmt.Fprintf(w, `  <url><loc>%s</loc><changefreq>weekly</changefreq><priority>0.8</priority></url>
`, u)
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
	fmt.Fprintln(w, "Sitemap: https://go-learn.example.com/sitemap.xml")
}
