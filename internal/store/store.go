package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store defines the data-access interface for go-learn.
// Implementations include SQLiteStore and, in future, a D1-backed store
// for Cloudflare Workers deployment.
type Store interface {
	Close() error

	// Auth
	CreateUser(email, passwordHash string) (User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	CreateSession(userID string, expiresAt string) (Session, error)
	GetSession(token string) (*Session, error)
	DeleteSession(token string) error

	// Answers (user-scoped)
	SaveAnswer(userID, questionID, pickedKey, pickedLabel string) (Answer, error)
	GetAnswer(userID, questionID string) (*Answer, error)
	ListAnswers(userID string) ([]AnswerRow, error)

	// Lessons / questions / sections (global content)
	ListLessons() ([]Lesson, error)
	GetLesson(id string) (*Lesson, error)
	GetLessonBySlug(slug string) (*Lesson, error)
	ListLessonSections(lessonID string) ([]LessonSection, error)
	ListQuestionsByLesson(userID, lessonID string) ([]Question, error)
	GetQuestion(id string) (*Question, error)

	// Reference content (global)
	ListGlossaryTerms() ([]GlossaryTerm, error)
	ListReferences() ([]Reference, error)
	ListActiveInsights(kind string) ([]Insight, error)
	GetMission() (*Mission, error)

	// Exercises (user-scoped submission status)
	ListExercises(userID string) ([]Exercise, error)
	ListExercisesByLesson(userID, lessonID string) ([]Exercise, error)
	SaveExerciseSubmission(userID, exerciseID, output string, correct bool) error
	GetExerciseSubmission(userID, exerciseID string) (string, bool, error)

	// Progress (user-scoped)
	LessonProgress(userID string) ([]LessonProgress, error)

	// UserData fetches all user-specific data in 2 queries (answers + submissions).
	// Used to compute dashboard stats, lesson progress, and exercise status in one shot.
	UserData(userID string) (*UserData, error)

	// LessonCounts returns question and exercise counts per lessonID in 1 call.
	// d1store: 0 D1 queries (embedded content). SQLiteStore: 2 SQL queries.
	LessonCounts() (questionCounts, exerciseCounts map[string]int, err error)

	// Content import
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
	QuestionType string `json:"questionType"`
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// SaveAnswer records a quiz answer, evaluates correctness, and returns the result.
func (s *SQLiteStore) SaveAnswer(userID, questionID, pickedKey, pickedLabel string) (Answer, error) {
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
		// Case-insensitive, trimmed comparison — beginner-friendly.
		correct = strings.EqualFold(strings.TrimSpace(pickedKey), strings.TrimSpace(correctKey))
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
		INSERT INTO answers (user_id, question_id, picked_key, picked_label, correct, answered_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, question_id) DO UPDATE SET
			picked_key = excluded.picked_key,
			picked_label = excluded.picked_label,
			correct = excluded.correct,
			answered_at = excluded.answered_at
	`, userID, questionID, pickedKey, pickedLabel, boolToInt(correct), now)
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

// GetAnswer retrieves a previously saved answer for a question by a user, or nil if unanswered.
func (s *SQLiteStore) GetAnswer(userID, questionID string) (*Answer, error) {
	var a Answer
	var correct int
	err := s.db.QueryRow(`
		SELECT question_id, picked_key, picked_label, correct, answered_at
		FROM answers WHERE user_id = ? AND question_id = ?
	`, userID, questionID).Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get answer: %w", err)
	}
	a.Correct = correct == 1
	return &a, nil
}

// ListAnswers returns all answers for a user joined with lesson and question data.
func (s *SQLiteStore) ListAnswers(userID string) ([]AnswerRow, error) {
	rows, err := s.db.Query(`
		SELECT
			a.question_id, a.picked_key, a.picked_label, a.correct, a.answered_at,
			q.lesson_id, l.title, q.prompt, q.correct_key, q.question_type,
			COALESCE(
				(SELECT o.label FROM question_options o
				 WHERE o.question_id = q.id AND o.is_correct = 1 LIMIT 1),
				q.correct_key
			) AS correct_label
		FROM answers a
		JOIN questions q ON q.id = a.question_id
		JOIN lessons l ON l.id = q.lesson_id
		WHERE a.user_id = ?
		ORDER BY l.sort_order, q.sort_order
	`, userID)
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
			&row.LessonID, &row.LessonTitle, &row.Prompt, &row.CorrectKey, &row.QuestionType, &row.CorrectLabel,
		); err != nil {
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		row.Correct = correct == 1
		out = append(out, row)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// UserData fetches all user-specific data in 2 queries (answers + submissions).
func (s *SQLiteStore) UserData(userID string) (*UserData, error) {
	ud := &UserData{
		AnswersByLesson:     make(map[string][2]int),
		SubmissionsByEx:     make(map[string]bool),
		CorrectByEx:         make(map[string]bool),
		SubmissionsByLesson: make(map[string]int),
	}
	if userID == "" {
		return ud, nil
	}

	rows, err := s.db.Query(`
		SELECT q.lesson_id, COUNT(*), COALESCE(SUM(a.correct), 0)
		FROM answers a JOIN questions q ON q.id = a.question_id
		WHERE a.user_id = ?
		GROUP BY q.lesson_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("UserData answers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lid string
		var answered, correct int
		if err := rows.Scan(&lid, &answered, &correct); err != nil {
			return nil, fmt.Errorf("UserData answers: %w", err)
		}
		ud.AnswersByLesson[lid] = [2]int{answered, correct}
		ud.TotalAnswered += answered
		ud.TotalCorrect += correct
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserData answers: %w", err)
	}

	esRows, err := s.db.Query(`
		SELECT es.exercise_id, e.lesson_id, es.correct
		FROM exercise_submissions es JOIN exercises e ON e.id = es.exercise_id
		WHERE es.user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("UserData submissions: %w", err)
	}
	defer esRows.Close()
	for esRows.Next() {
		var exID, lessonID string
		var correct int
		if err := esRows.Scan(&exID, &lessonID, &correct); err != nil {
			return nil, fmt.Errorf("UserData submissions: %w", err)
		}
		ud.SubmissionsByEx[exID] = true
		ud.CorrectByEx[exID] = correct == 1
		ud.SubmissionsByLesson[lessonID]++
		ud.TotalSubmitted++
	}
	if err := esRows.Err(); err != nil {
		return nil, fmt.Errorf("UserData submissions: %w", err)
	}

	return ud, nil
}

// LessonCounts returns question and exercise counts per lessonID.
func (s *SQLiteStore) LessonCounts() (map[string]int, map[string]int, error) {
	qRows, err := s.db.Query(`SELECT lesson_id, COUNT(*) FROM questions GROUP BY lesson_id`)
	if err != nil {
		return nil, nil, fmt.Errorf("LessonCounts questions: %w", err)
	}
	defer qRows.Close()
	qCounts := make(map[string]int)
	for qRows.Next() {
		var lid string
		var n int
		if err := qRows.Scan(&lid, &n); err != nil {
			return nil, nil, fmt.Errorf("LessonCounts questions: %w", err)
		}
		qCounts[lid] = n
	}
	if err := qRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("LessonCounts questions: %w", err)
	}

	eRows, err := s.db.Query(`SELECT lesson_id, COUNT(*) FROM exercises GROUP BY lesson_id`)
	if err != nil {
		return nil, nil, fmt.Errorf("LessonCounts exercises: %w", err)
	}
	defer eRows.Close()
	eCounts := make(map[string]int)
	for eRows.Next() {
		var lid string
		var n int
		if err := eRows.Scan(&lid, &n); err != nil {
			return nil, nil, fmt.Errorf("LessonCounts exercises: %w", err)
		}
		eCounts[lid] = n
	}
	if err := eRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("LessonCounts exercises: %w", err)
	}

	return qCounts, eCounts, nil
}
