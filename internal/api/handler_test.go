package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/api"
	"github.com/kartikkabadi/go-learn/internal/store"
)

func testStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func importTestLesson(t *testing.T, s store.Store) {
	t.Helper()
	bundle := store.ContentBundle{
		Lesson: store.BundleLesson{ID: "api-t1", Title: "Test Lesson", Slug: "test", Summary: "Test", SortOrder: 1},
		Questions: []store.BundleQuestion{
			{
				ID: "api-t1:q1", Prompt: "Pick?", CorrectKey: "x", QuestionType: "choice", SortOrder: 1,
				Options: []store.BundleOption{
					{Key: "x", Label: "Correct", IsCorrect: true, SortOrder: 1},
					{Key: "y", Label: "Wrong", IsCorrect: false, SortOrder: 2},
				},
			},
		},
	}
	if err := s.ImportBundle(bundle); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAnswer_ValidJSON(t *testing.T) {
	s := testStore(t)
	importTestLesson(t, s)
	h := &api.Handler{Store: s}

	body := `{"questionId":"api-t1:q1","pickedKey":"x","pickedLabel":"Correct"}`
	req := httptest.NewRequest(http.MethodPost, "/api/answers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SaveAnswer(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var ans store.Answer
	if err := json.NewDecoder(resp.Body).Decode(&ans); err != nil {
		t.Fatal(err)
	}
	if !ans.Correct {
		t.Fatal("expected correct answer")
	}
	if ans.QuestionID != "api-t1:q1" {
		t.Fatalf("want question api-t1:q1, got %q", ans.QuestionID)
	}
	resp.Body.Close()
}

func TestSaveAnswer_InvalidJSON(t *testing.T) {
	s := testStore(t)
	h := &api.Handler{Store: s}

	req := httptest.NewRequest(http.MethodPost, "/api/answers", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SaveAnswer(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSaveAnswer_MissingQuestionID(t *testing.T) {
	s := testStore(t)
	h := &api.Handler{Store: s}

	body := `{"questionId":"","pickedKey":"x","pickedLabel":"X"}`
	req := httptest.NewRequest(http.MethodPost, "/api/answers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SaveAnswer(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestListAnswers(t *testing.T) {
	s := testStore(t)
	importTestLesson(t, s)
	h := &api.Handler{Store: s}

	if _, err := s.SaveAnswer("api-t1:q1", "x", "Correct"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/answers", nil)
	w := httptest.NewRecorder()
	h.ListAnswers(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var answers []store.AnswerRow
	if err := json.NewDecoder(resp.Body).Decode(&answers); err != nil {
		t.Fatal(err)
	}
	if len(answers) != 1 {
		t.Fatalf("want 1 answer, got %d", len(answers))
	}
	if answers[0].QuestionID != "api-t1:q1" {
		t.Fatalf("want api-t1:q1, got %q", answers[0].QuestionID)
	}
	resp.Body.Close()
}

func TestListAnswers_WithLesson(t *testing.T) {
	s := testStore(t)
	importTestLesson(t, s)
	h := &api.Handler{Store: s}

	if _, err := s.SaveAnswer("api-t1:q1", "x", "Correct"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/answers?lesson=api-t1", nil)
	w := httptest.NewRecorder()
	h.ListAnswers(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var answers []store.Answer
	if err := json.NewDecoder(resp.Body).Decode(&answers); err != nil {
		t.Fatal(err)
	}
	if len(answers) != 1 {
		t.Fatalf("want 1 answer, got %d", len(answers))
	}
	resp.Body.Close()

	// Non-existent lesson returns empty slice.
	req2 := httptest.NewRequest(http.MethodGet, "/api/answers?lesson=nonexistent", nil)
	w2 := httptest.NewRecorder()
	h.ListAnswers(w2, req2)
	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp2.StatusCode)
	}
	var empty []store.Answer
	if err := json.NewDecoder(resp2.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("want 0 answers, got %d", len(empty))
	}
	resp2.Body.Close()
}

func TestGetAnswer_Found(t *testing.T) {
	s := testStore(t)
	importTestLesson(t, s)
	h := &api.Handler{Store: s}

	if _, err := s.SaveAnswer("api-t1:q1", "x", "Correct"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/answers/api-t1:q1", nil)
	req.SetPathValue("questionId", "api-t1:q1")
	w := httptest.NewRecorder()
	h.GetAnswer(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var ans store.Answer
	if err := json.NewDecoder(resp.Body).Decode(&ans); err != nil {
		t.Fatal(err)
	}
	if !ans.Correct {
		t.Fatal("expected correct answer")
	}
	resp.Body.Close()
}

func TestGetAnswer_NotFound(t *testing.T) {
	s := testStore(t)
	importTestLesson(t, s)
	h := &api.Handler{Store: s}

	req := httptest.NewRequest(http.MethodGet, "/api/answers/api-t1:q1", nil)
	req.SetPathValue("questionId", "api-t1:q1")
	w := httptest.NewRecorder()
	h.GetAnswer(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
