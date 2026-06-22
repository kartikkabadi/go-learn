package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Store defines the data-access interface for go-learn.
// Implementations include SQLiteStore and, in future, a D1-backed store
// for Cloudflare Workers deployment.
type Store interface {
	Close() error
	SaveAnswer(questionID, pickedKey, pickedLabel string) (Answer, error)
	GetAnswer(questionID string) (*Answer, error)
	ListAnswers() ([]AnswerRow, error)
	ListAnswersByLesson(lessonID string) ([]Answer, error)
	ListLessons() ([]Lesson, error)
	GetLesson(id string) (*Lesson, error)
	GetLessonBySlug(slug string) (*Lesson, error)
	ListLessonSections(lessonID string) ([]LessonSection, error)
	ListQuestionsByLesson(lessonID string) ([]Question, error)
	GetQuestion(id string) (*Question, error)
	ListGlossaryTerms() ([]GlossaryTerm, error)
	ListReferences() ([]Reference, error)
	ListExercises() ([]Exercise, error)
	ListExercisesByLesson(lessonID string) ([]Exercise, error)
	SaveExerciseSubmission(exerciseID, output string) error
	GetExerciseSubmission(exerciseID string) (string, bool, error)
	ListActiveInsights(kind string) ([]Insight, error)
	GetMission() (*Mission, error)
	LessonProgress() ([]LessonProgress, error)
	DashboardStats() (DashboardStats, error)
	ImportBundle(b ContentBundle) error
	ImportMission(b MissionBundle) error
	ImportGlossary(b GlossaryBundle) error
	ImportInsights(b InsightsBundle) error
}

// Ensure SQLiteStore satisfies Store at compile time.
var _ Store = (*SQLiteStore)(nil)

type SQLiteStore struct {
	db *sql.DB
}

// Answer represents a single quiz answer stored in the database.
type Answer struct {
	QuestionID  string `json:"questionId"`
	PickedKey   string `json:"pickedKey"`
	PickedLabel string `json:"pickedLabel"`
	Correct     bool   `json:"correct"`
	AnsweredAt  string `json:"answeredAt"`
}

// AnswerRow extends Answer with lesson and question metadata for progress views.
type AnswerRow struct {
	Answer
	LessonID     string `json:"lessonId"`
	LessonTitle  string `json:"lessonTitle"`
	Prompt       string `json:"prompt"`
	CorrectKey   string `json:"correctKey"`
	CorrectLabel string `json:"correctLabel"`
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// SaveAnswer records a quiz answer, evaluates correctness, and returns the result.
func (s *SQLiteStore) SaveAnswer(questionID, pickedKey, pickedLabel string) (Answer, error) {
	var qType, correctKey string
	err := s.db.QueryRow(
		`SELECT question_type, correct_key FROM questions WHERE id = ?`,
		questionID,
	).Scan(&qType, &correctKey)
	if err == sql.ErrNoRows {
		return Answer{}, fmt.Errorf("unknown question %q", questionID)
	}
	if err != nil {
		return Answer{}, fmt.Errorf("lookup question: %w", err)
	}

	correct := false
	switch qType {
	case "text":
		correct = pickedKey == correctKey
	case "choice":
		var isCorrect int
		err = s.db.QueryRow(
			`SELECT is_correct FROM question_options WHERE question_id = ? AND option_key = ?`,
			questionID, pickedKey,
		).Scan(&isCorrect)
		if err == sql.ErrNoRows {
			return Answer{}, fmt.Errorf("unknown option %q for %q", pickedKey, questionID)
		}
		if err != nil {
			return Answer{}, fmt.Errorf("lookup option: %w", err)
		}
		correct = isCorrect == 1
	default:
		return Answer{}, fmt.Errorf("unsupported question type %q", qType)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO answers (question_id, picked_key, picked_label, correct, answered_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(question_id) DO UPDATE SET
			picked_key = excluded.picked_key,
			picked_label = excluded.picked_label,
			correct = excluded.correct,
			answered_at = excluded.answered_at
	`, questionID, pickedKey, pickedLabel, boolToInt(correct), now)
	if err != nil {
		return Answer{}, fmt.Errorf("save answer: %w", err)
	}

	return Answer{
		QuestionID:  questionID,
		PickedKey:   pickedKey,
		PickedLabel: pickedLabel,
		Correct:     correct,
		AnsweredAt:  now,
	}, nil
}

// GetAnswer retrieves a previously saved answer for a question, or nil if unanswered.
func (s *SQLiteStore) GetAnswer(questionID string) (*Answer, error) {
	var a Answer
	var correct int
	err := s.db.QueryRow(`
		SELECT question_id, picked_key, picked_label, correct, answered_at
		FROM answers WHERE question_id = ?
	`, questionID).Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get answer: %w", err)
	}
	a.Correct = correct == 1
	return &a, nil
}

// ListAnswers returns all answers joined with lesson and question data, ordered by lesson.
func (s *SQLiteStore) ListAnswers() ([]AnswerRow, error) {
	rows, err := s.db.Query(`
		SELECT
			a.question_id, a.picked_key, a.picked_label, a.correct, a.answered_at,
			q.lesson_id, l.title, q.prompt, q.correct_key,
			COALESCE(
				(SELECT o.label FROM question_options o
				 WHERE o.question_id = q.id AND o.is_correct = 1 LIMIT 1),
				q.correct_key
			) AS correct_label
		FROM answers a
		JOIN questions q ON q.id = a.question_id
		JOIN lessons l ON l.id = q.lesson_id
		ORDER BY l.sort_order, q.sort_order
	`)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()

	var out []AnswerRow
	for rows.Next() {
		var row AnswerRow
		var correct int
		if err := rows.Scan(
			&row.QuestionID, &row.PickedKey, &row.PickedLabel, &correct, &row.AnsweredAt,
			&row.LessonID, &row.LessonTitle, &row.Prompt, &row.CorrectKey, &row.CorrectLabel,
		); err != nil {
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		row.Correct = correct == 1
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListAnswersByLesson returns answers for a specific lesson.
func (s *SQLiteStore) ListAnswersByLesson(lessonID string) ([]Answer, error) {
	rows, err := s.db.Query(`
		SELECT a.question_id, a.picked_key, a.picked_label, a.correct, a.answered_at
		FROM answers a
		JOIN questions q ON q.id = a.question_id
		WHERE q.lesson_id = ?
		ORDER BY q.sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list lesson answers: %w", err)
	}
	defer rows.Close()

	var out []Answer
	for rows.Next() {
		var a Answer
		var correct int
		if err := rows.Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt); err != nil {
			return nil, fmt.Errorf("scan lesson answer: %w", err)
		}
		a.Correct = correct == 1
		out = append(out, a)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
