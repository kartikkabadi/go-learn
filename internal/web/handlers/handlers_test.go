package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
	handlers "github.com/kartikkabadi/go-learn/internal/web/handlers"
	"github.com/kartikkabadi/go-learn/internal/web/views"
)

func testHandler(t *testing.T) *handlers.Handler {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	renderer, err := views.New()
	if err != nil {
		t.Fatal(err)
	}

	return &handlers.Handler{
		Store:    s,
		Progress: &service.Progress{Store: s},
		Views:    renderer,
	}
}

func importTestLesson(t *testing.T, s store.Store) {
	t.Helper()
	bundle := store.ContentBundle{
		Lesson: store.BundleLesson{ID: "web-t1", Title: "Test Lesson", Slug: "test", Summary: "Test lesson summary", SortOrder: 1},
		Sections: []store.BundleSection{
			{ID: "web-t1:s1", Heading: "Section 1", BodyHTML: "<p>content</p>", SortOrder: 1},
		},
		Questions: []store.BundleQuestion{
			{
				ID: "web-t1:q1", Prompt: "Pick?", CorrectKey: "a", QuestionType: "choice", SortOrder: 1,
				SectionTag: "quiz",
				Options: []store.BundleOption{
					{Key: "a", Label: "Right", IsCorrect: true, SortOrder: 1},
					{Key: "b", Label: "Wrong", IsCorrect: false, SortOrder: 2},
				},
			},
		},
		Exercises: []store.BundleExercise{
			{ID: "web-t1:ex1", Title: "Test Ex", Path: "practice/test", Instructions: "Do it", SortOrder: 1},
		},
	}
	if err := s.ImportBundle(bundle); err != nil {
		t.Fatal(err)
	}
}

func TestDashboard(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.Dashboard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLessonsIndex(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/lessons", nil)
	w := httptest.NewRecorder()
	h.LessonsIndex(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLessonShow_Valid(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/lessons/web-t1", nil)
	req.SetPathValue("slug", "test")
	w := httptest.NewRecorder()
	h.LessonShow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestLessonShow_NotFound(t *testing.T) {
	h := testHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lessons/nonexistent", nil)
	req.SetPathValue("slug", "nonexistent")
	w := httptest.NewRecorder()
	h.LessonShow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAnswerQuestion_Valid(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	form := url.Values{"pickedKey": {"a"}, "pickedLabel": {"Right"}}
	req := httptest.NewRequest(http.MethodPost, "/lessons/web-t1/questions/web-t1:q1/answer",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("slug", "test")
	req.SetPathValue("qid", "web-t1:q1")
	w := httptest.NewRecorder()
	h.AnswerQuestion(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAnswerQuestion_EmptyKey(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	form := url.Values{"pickedKey": {""}}
	req := httptest.NewRequest(http.MethodPost, "/lessons/web-t1/questions/web-t1:q1/answer",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("slug", "test")
	req.SetPathValue("qid", "web-t1:q1")
	w := httptest.NewRecorder()
	h.AnswerQuestion(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPractice(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/practice", nil)
	w := httptest.NewRecorder()
	h.Practice(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestProgressPage(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/progress", nil)
	w := httptest.NewRecorder()
	h.ProgressPage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
