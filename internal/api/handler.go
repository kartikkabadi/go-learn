package api

import (
	"encoding/json"
	"net/http"

	"github.com/kartikkabadi/go-learn/internal/store"
)

// Handler provides JSON API endpoints for quiz answers.
type Handler struct {
	Store store.Store
}

type saveAnswerRequest struct {
	QuestionID  string `json:"questionId"`
	PickedKey   string `json:"pickedKey"`
	PickedLabel string `json:"pickedLabel"`
}

func (h *Handler) SaveAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req saveAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	answer, err := h.Store.SaveAnswer(req.QuestionID, req.PickedKey, req.PickedLabel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

func (h *Handler) ListAnswers(w http.ResponseWriter, r *http.Request) {
	lessonID := r.URL.Query().Get("lesson")
	if lessonID != "" {
		answers, err := h.Store.ListAnswersByLesson(lessonID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(answers)
		return
	}
	answers, err := h.Store.ListAnswers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if answers == nil {
		answers = []store.AnswerRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answers)
}

func (h *Handler) GetAnswer(w http.ResponseWriter, r *http.Request) {
	questionID := r.PathValue("questionId")
	answer, err := h.Store.GetAnswer(questionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if answer == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}
