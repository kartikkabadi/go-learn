package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
	"github.com/kartikkabadi/go-learn/internal/web/cookies"
	handlers "github.com/kartikkabadi/go-learn/internal/web/handlers"
	"github.com/kartikkabadi/go-learn/internal/web/middleware"
	"github.com/kartikkabadi/go-learn/internal/web/views"
)

// withTestUser creates a user and attaches it to the request context for tests
// that exercise auth-required handlers.
func withTestUser(t *testing.T, s store.Store, r *http.Request) *http.Request {
	t.Helper()
	u, err := s.CreateUser("handler-test@example.com", "$2a$10$dummyhashfortestonlyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	if err != nil {
		t.Fatal(err)
	}
	return middleware.WithUser(r, &store.User{ID: u.ID, Email: u.Email})
}

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
		Store:     s,
		Progress:  &service.Progress{Store: s},
		Views:     renderer,
		CookieKey: []byte("test-cookie-key"),
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

func TestDashboard_Anonymous(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.Dashboard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	// Anonymous visitors should see the landing page, not personal stats.
	if strings.Contains(body, "questions answered") {
		t.Fatal("anonymous landing page should not contain 'questions answered'")
	}
	if strings.Contains(body, "accuracy") {
		t.Fatal("anonymous landing page should not contain 'accuracy'")
	}
	if !strings.Contains(body, "Start learning") {
		t.Fatal("anonymous landing page should contain 'Start learning' CTA")
	}
	resp.Body.Close()
}

func TestDashboard_Authed(t *testing.T) {
	h := testHandler(t)
	importTestLesson(t, h.Store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withTestUser(t, h.Store, req)
	w := httptest.NewRecorder()
	h.Dashboard(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	// Authed users should see the dashboard with personal stats.
	if !strings.Contains(body, "questions answered") {
		t.Fatal("authed dashboard should contain 'questions answered'")
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

	req := httptest.NewRequest(http.MethodGet, "/lessons/test", nil)
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
	req.SetPathValue("id", "web-t1")
	req.SetPathValue("qid", "web-t1:q1")
	req = withTestUser(t, h.Store, req)
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
	req.SetPathValue("id", "web-t1")
	req.SetPathValue("qid", "web-t1:q1")
	req = withTestUser(t, h.Store, req)
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

func TestAuthFlow_LoginAndLogout(t *testing.T) {
	h := testHandler(t)
	u, err := h.Store.CreateUser("authflow@example.com", "$2a$10$dummyhashfortestonlyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	if err != nil {
		t.Fatal(err)
	}

	// Manually create a session to simulate login state.
	sess, err := h.Store.CreateSession(u.ID, time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("LoadUser accepts signed cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: cookies.Sign(sess.Token, h.CookieKey),
		})

		loader := middleware.LoadUser(h.Store, h.CookieKey)
		var got *store.User
		handler := loader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = middleware.UserFromContext(r)
			w.WriteHeader(http.StatusOK)
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
		if got == nil || got.ID != u.ID {
			t.Fatalf("expected user %s, got %v", u.ID, got)
		}
	})

	t.Run("LoadUser rejects tampered cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "not-signed"})

		loader := middleware.LoadUser(h.Store, h.CookieKey)
		var got *store.User
		handler := loader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = middleware.UserFromContext(r)
			w.WriteHeader(http.StatusOK)
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if got != nil {
			t.Fatalf("expected no user, got %v", got)
		}
	})
}
