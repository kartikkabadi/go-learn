//go:build js && wasm

// Package d1store implements the store.Store interface using
// Cloudflare D1 as the backing database. Build with GOOS=js GOARCH=wasm
// for deployment on Cloudflare Workers via syumai/workers.
package d1store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kartikkabadi/go-learn/internal/store"
	_ "github.com/syumai/workers/cloudflare/d1"
)

// Compile-time check that Store satisfies store.Store.
var _ store.Store = (*Store)(nil)

// Store implements store.Store backed by Cloudflare D1.
// Static content is cached in memory after first load (see cache.go),
// eliminating D1 round-trips for content that only changes on deploy.
type Store struct {
	db      *sql.DB
	cacheMu sync.Mutex
	cache   *contentCache
	cacheErr error
}

// Open opens a D1-backed store. dbName is the D1 binding name from wrangler.jsonc.
func Open(dbName string) (*Store, error) {
	db, err := sql.Open("d1", dbName)
	if err != nil {
		return nil, fmt.Errorf("d1store open: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB, exposed for migration.
func (s *Store) DB() *sql.DB { return s.db }

// SaveAnswer records a quiz answer, evaluates correctness, and returns the result.
func (s *Store) SaveAnswer(userID, questionID, pickedKey, pickedLabel string) (store.Answer, error) {
	var qType, correctKey string
	err := s.db.QueryRow(
		`SELECT question_type, correct_key FROM questions WHERE id = ?`,
		questionID,
	).Scan(&qType, &correctKey)
	if err == sql.ErrNoRows {
		return store.Answer{}, fmt.Errorf("unknown question %q", questionID)
	}
	if err != nil {
		return store.Answer{}, fmt.Errorf("lookup question: %w", err)
	}

	correct := false
	switch qType {
	case "text":
		correct = strings.EqualFold(strings.TrimSpace(pickedKey), strings.TrimSpace(correctKey))
	case "choice":
		var isCorrect int
		err = s.db.QueryRow(
			`SELECT is_correct FROM question_options WHERE question_id = ? AND option_key = ?`,
			questionID, pickedKey,
		).Scan(&isCorrect)
		if err == sql.ErrNoRows {
			return store.Answer{}, fmt.Errorf("unknown option %q for %q", pickedKey, questionID)
		}
		if err != nil {
			return store.Answer{}, fmt.Errorf("lookup option: %w", err)
		}
		correct = isCorrect == 1
	default:
		return store.Answer{}, fmt.Errorf("unsupported question type %q", qType)
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
		return store.Answer{}, fmt.Errorf("save answer: %w", err)
	}

	return store.Answer{
		QuestionID:  questionID,
		PickedKey:   pickedKey,
		PickedLabel: pickedLabel,
		Correct:     correct,
		AnsweredAt:  now,
	}, nil
}

// GetAnswer retrieves a previously saved answer for a question, or nil if unanswered.
func (s *Store) GetAnswer(userID, questionID string) (*store.Answer, error) {
	var a store.Answer
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

// ListAnswers returns all answers joined with lesson and question data.
func (s *Store) ListAnswers(userID string) ([]store.AnswerRow, error) {
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
		WHERE a.user_id = ?
		ORDER BY l.sort_order, q.sort_order
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()

	var out []store.AnswerRow
	for rows.Next() {
		var row store.AnswerRow
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

// ListLessons returns all lessons from the in-memory cache (0 D1 queries).
func (s *Store) ListLessons() ([]store.Lesson, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	return c.lessons, nil
}

// GetLesson retrieves a single lesson by ID from the in-memory cache.
func (s *Store) GetLesson(id string) (*store.Lesson, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	if l, ok := c.byID[id]; ok {
		return l, nil
	}
	return nil, nil
}

// GetLessonBySlug retrieves a single lesson by slug from the in-memory cache.
func (s *Store) GetLessonBySlug(slug string) (*store.Lesson, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	if l, ok := c.bySlug[slug]; ok {
		return l, nil
	}
	return nil, nil
}

// ListLessonSections returns sections for a lesson from the in-memory cache.
func (s *Store) ListLessonSections(lessonID string) ([]store.LessonSection, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	return c.sections[lessonID], nil
}

// ListQuestionsByLesson returns all questions for a lesson from the cache
// (options populated), then fetches the user's answers in 1 D1 query (if
// logged in). Anonymous requests = 0 D1 queries.
func (s *Store) ListQuestionsByLesson(userID, lessonID string) ([]store.Question, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	cached := c.questions[lessonID]
	// Copy so we can attach per-user answers without mutating the cache.
	out := make([]store.Question, len(cached))
	copy(out, cached)
	if len(out) == 0 {
		return out, nil
	}

	if userID == "" {
		return out, nil
	}

	// Single query for all user answers in this lesson.
	ansRows, err := s.db.Query(`
		SELECT question_id, picked_key, picked_label, correct, answered_at
		FROM answers
		WHERE user_id = ? AND question_id IN (SELECT id FROM questions WHERE lesson_id = ?)
	`, userID, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
	}
	defer ansRows.Close()
	ansByQ := make(map[string]*store.Answer)
	for ansRows.Next() {
		var a store.Answer
		var correct int
		if err := ansRows.Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt); err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
		}
		a.Correct = correct == 1
		ansByQ[a.QuestionID] = &a
	}
	for i := range out {
		out[i].Answer = ansByQ[out[i].ID]
	}
	return out, nil
}

// GetQuestion retrieves a single question by ID with options from the cache.
// It does not populate the user's answer — the caller sets q.Answer if needed.
func (s *Store) GetQuestion(id string) (*store.Question, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	if q, ok := c.questionsByID[id]; ok {
		return q, nil
	}
	return nil, nil
}

// ListGlossaryTerms returns glossary terms from the in-memory cache.
func (s *Store) ListGlossaryTerms() ([]store.GlossaryTerm, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	return c.glossary, nil
}

// ListReferences returns external references from the in-memory cache.
func (s *Store) ListReferences() ([]store.Reference, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	return c.references, nil
}

// ListExercises returns all exercises with submission status for the given user.
// Exercise definitions come from cache; submissions fetched in 1 D1 query.
func (s *Store) ListExercises(userID string) ([]store.Exercise, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	out := make([]store.Exercise, len(c.exercises))
	copy(out, c.exercises)
	if userID == "" {
		return out, nil
	}
	return s.attachSubmissions(out, userID)
}

// ListExercisesByLesson returns exercises for a lesson with submission status.
// Exercise definitions come from cache; submissions fetched in 1 D1 query.
func (s *Store) ListExercisesByLesson(userID, lessonID string) ([]store.Exercise, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	cached := c.exByLesson[lessonID]
	out := make([]store.Exercise, len(cached))
	copy(out, cached)
	if userID == "" {
		return out, nil
	}
	return s.attachSubmissions(out, userID)
}

// attachSubmissions fetches the user's submissions for the given exercises
// in 1 query and merges Submitted/Correct into the slice.
func (s *Store) attachSubmissions(out []store.Exercise, userID string) ([]store.Exercise, error) {
	if len(out) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(`
		SELECT exercise_id, correct FROM exercise_submissions WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("ListExercises submissions: %w", err)
	}
	defer rows.Close()
	subByEx := make(map[string]bool)
	correctByEx := make(map[string]bool)
	for rows.Next() {
		var exID string
		var correct int
		if err := rows.Scan(&exID, &correct); err != nil {
			return nil, fmt.Errorf("ListExercises submissions: %w", err)
		}
		subByEx[exID] = true
		correctByEx[exID] = correct == 1
	}
	for i := range out {
		if subByEx[out[i].ID] {
			out[i].Submitted = true
			out[i].Correct = correctByEx[out[i].ID]
		}
	}
	return out, nil
}

// SaveExerciseSubmission stores terminal output and correctness for an exercise by a user, upserting by (user_id, exercise_id).
func (s *Store) SaveExerciseSubmission(userID, exerciseID, output string, correct bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`
		INSERT INTO exercise_submissions (user_id, exercise_id, output, correct, submitted_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, exercise_id) DO UPDATE SET
			output = excluded.output,
			correct = excluded.correct,
			submitted_at = excluded.submitted_at
	`, userID, exerciseID, output, boolToInt(correct), now)
	return err
}

// GetExerciseSubmission retrieves saved output for an exercise by a user.
func (s *Store) GetExerciseSubmission(userID, exerciseID string) (string, bool, error) {
	var output string
	err := s.db.QueryRow(
		`SELECT output FROM exercise_submissions WHERE user_id = ? AND exercise_id = ?`,
		userID, exerciseID,
	).Scan(&output)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("GetExerciseSubmission: %w", err)
	}
	return output, true, nil
}

// ListActiveInsights returns active insights from the in-memory cache,
// optionally filtered by kind.
func (s *Store) ListActiveInsights(kind string) ([]store.Insight, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	if kind == "" {
		return c.insights, nil
	}
	var out []store.Insight
	for _, ins := range c.insights {
		if ins.Kind == kind {
			out = append(out, ins)
		}
	}
	return out, nil
}

// GetMission returns the learner mission statement from the in-memory cache.
func (s *Store) GetMission() (*store.Mission, error) {
	c, err := s.loadCache()
	if err != nil {
		return nil, err
	}
	return c.mission, nil
}

// LessonProgress returns progress stats for every lesson for the given user.
// Single query with correlated subqueries + grouped LEFT JOINs — avoids N+1.
func (s *Store) LessonProgress(userID string) ([]store.LessonProgress, error) {
	rows, err := s.db.Query(`
		SELECT
			l.id, l.title, l.slug, l.sort_order,
			(SELECT COUNT(*) FROM questions q WHERE q.lesson_id = l.id),
			COALESCE(qa.answered, 0),
			COALESCE(qa.correct, 0),
			(SELECT COUNT(*) FROM exercises e WHERE e.lesson_id = l.id),
			COALESCE(es.done, 0)
		FROM lessons l
		LEFT JOIN (
			SELECT q.lesson_id, COUNT(*) AS answered, COALESCE(SUM(a.correct), 0) AS correct
			FROM answers a JOIN questions q ON q.id = a.question_id
			WHERE a.user_id = ?
			GROUP BY q.lesson_id
		) qa ON qa.lesson_id = l.id
		LEFT JOIN (
			SELECT e.lesson_id, COUNT(*) AS done
			FROM exercise_submissions es JOIN exercises e ON e.id = es.exercise_id
			WHERE es.user_id = ?
			GROUP BY e.lesson_id
		) es ON es.lesson_id = l.id
		ORDER BY l.sort_order
	`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("LessonProgress: %w", err)
	}
	defer rows.Close()
	var out []store.LessonProgress
	for rows.Next() {
		var lp store.LessonProgress
		if err := rows.Scan(
			&lp.LessonID, &lp.Title, &lp.Slug, &lp.SortOrder,
			&lp.QuestionsTotal, &lp.QuestionsAnswered, &lp.QuestionsCorrect,
			&lp.ExerciseTotal, &lp.ExercisesDone,
		); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		out = append(out, lp)
	}
	return out, rows.Err()
}

// DashboardStats returns aggregate progress numbers across all lessons.
// Single query with scalar subqueries — avoids 5 separate round-trips.
func (s *Store) DashboardStats(userID string) (store.DashboardStats, error) {
	var st store.DashboardStats
	err := s.db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM lessons),
			(SELECT COUNT(*) FROM questions),
			(SELECT COUNT(*) FROM exercises),
			COALESCE((SELECT COUNT(*) FROM answers WHERE user_id = ?), 0),
			COALESCE((SELECT SUM(correct) FROM answers WHERE user_id = ?), 0),
			COALESCE((SELECT COUNT(*) FROM exercise_submissions WHERE user_id = ?), 0)
	`, userID, userID, userID).Scan(
		&st.LessonsTotal, &st.QuestionsTotal, &st.ExercisesTotal,
		&st.QuestionsAnswered, &st.QuestionsCorrect, &st.ExercisesSubmitted,
	)
	return st, err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
