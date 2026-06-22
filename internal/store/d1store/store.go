//go:build js && wasm

// Package d1store implements the store.Store interface using
// Cloudflare D1 as the backing database. Build with GOOS=js GOARCH=wasm
// for deployment on Cloudflare Workers via syumai/workers.
package d1store

import (
	"database/sql"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/kartikkabadi/go-learn/internal/store"
	_ "github.com/syumai/workers/cloudflare/d1"
)

// Compile-time check that Store satisfies store.Store.
var _ store.Store = (*Store)(nil)

// Store implements store.Store backed by Cloudflare D1.
type Store struct {
	db *sql.DB
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

// listOptions returns all options for a question, ordered by sort_order.
func (s *Store) listOptions(questionID string) ([]store.QuestionOption, error) {
	rows, err := s.db.Query(`
		SELECT question_id, option_key, label, is_correct, sort_order
		FROM question_options WHERE question_id = ? ORDER BY sort_order
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("listOptions: %w", err)
	}
	defer rows.Close()
	var out []store.QuestionOption
	for rows.Next() {
		var o store.QuestionOption
		var correct int
		if err := rows.Scan(&o.QuestionID, &o.OptionKey, &o.Label, &correct, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("listOptions: %w", err)
		}
		o.IsCorrect = correct == 1
		out = append(out, o)
	}
	return out, rows.Err()
}

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

// ListLessons returns all lessons ordered by sort_order.
func (s *Store) ListLessons() ([]store.Lesson, error) {
	rows, err := s.db.Query(`SELECT id, title, slug, summary, sort_order FROM lessons ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("ListLessons: %w", err)
	}
	defer rows.Close()
	var out []store.Lesson
	for rows.Next() {
		var l store.Lesson
		if err := rows.Scan(&l.ID, &l.Title, &l.Slug, &l.Summary, &l.SortOrder); err != nil {
			return nil, fmt.Errorf("ListLessons: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetLesson retrieves a single lesson by ID, or nil if not found.
func (s *Store) GetLesson(id string) (*store.Lesson, error) {
	var l store.Lesson
	err := s.db.QueryRow(
		`SELECT id, title, slug, summary, sort_order FROM lessons WHERE id = ?`, id,
	).Scan(&l.ID, &l.Title, &l.Slug, &l.Summary, &l.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetLesson: %w", err)
	}
	return &l, nil
}

// GetLessonBySlug retrieves a single lesson by slug, or nil if not found.
func (s *Store) GetLessonBySlug(slug string) (*store.Lesson, error) {
	var l store.Lesson
	err := s.db.QueryRow(
		`SELECT id, title, slug, summary, sort_order FROM lessons WHERE slug = ?`, slug,
	).Scan(&l.ID, &l.Title, &l.Slug, &l.Summary, &l.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetLessonBySlug: %w", err)
	}
	return &l, nil
}

// ListLessonSections returns all sections for a lesson, ordered by sort_order.
func (s *Store) ListLessonSections(lessonID string) ([]store.LessonSection, error) {
	rows, err := s.db.Query(`
		SELECT id, lesson_id, heading, body_html, sort_order
		FROM lesson_sections WHERE lesson_id = ? ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListLessonSections: %w", err)
	}
	defer rows.Close()
	var out []store.LessonSection
	for rows.Next() {
		var sec store.LessonSection
		var body string
		if err := rows.Scan(&sec.ID, &sec.LessonID, &sec.Heading, &body, &sec.SortOrder); err != nil {
			return nil, fmt.Errorf("ListLessonSections: %w", err)
		}
		sec.BodyHTML = template.HTML(body)
		out = append(out, sec)
	}
	return out, rows.Err()
}

// ListQuestionsByLesson returns all questions for a lesson, with options and
// the user's answers populated. Uses 3 queries (questions, options, answers)
// instead of 1+2N to avoid N+1 round-trips on D1.
func (s *Store) ListQuestionsByLesson(userID, lessonID string) ([]store.Question, error) {
	rows, err := s.db.Query(`
		SELECT id, lesson_id, prompt, correct_key, question_type, section_tag, sort_order
		FROM questions WHERE lesson_id = ? ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
	}
	var out []store.Question
	for rows.Next() {
		var q store.Question
		if err := rows.Scan(&q.ID, &q.LessonID, &q.Prompt, &q.CorrectKey, &q.QuestionType, &q.SectionTag, &q.SortOrder); err != nil {
			rows.Close()
			return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
	}
	if len(out) == 0 {
		return out, nil
	}

	// Batch-fetch options for all questions in this lesson.
	optRows, err := s.db.Query(`
		SELECT question_id, option_key, label, is_correct, sort_order
		FROM question_options
		WHERE question_id IN (SELECT id FROM questions WHERE lesson_id = ?)
		ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson options: %w", err)
	}
	optsByQ := make(map[string][]store.QuestionOption)
	for optRows.Next() {
		var o store.QuestionOption
		var correct int
		if err := optRows.Scan(&o.QuestionID, &o.OptionKey, &o.Label, &correct, &o.SortOrder); err != nil {
			optRows.Close()
			return nil, fmt.Errorf("ListQuestionsByLesson options: %w", err)
		}
		o.IsCorrect = correct == 1
		optsByQ[o.QuestionID] = append(optsByQ[o.QuestionID], o)
	}
	if err := optRows.Close(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson options: %w", err)
	}

	// Batch-fetch answers for this user+lesson (only if logged in).
	var ansByQ map[string]*store.Answer
	if userID != "" {
		ansRows, err := s.db.Query(`
			SELECT question_id, picked_key, picked_label, correct, answered_at
			FROM answers
			WHERE user_id = ? AND question_id IN (SELECT id FROM questions WHERE lesson_id = ?)
		`, userID, lessonID)
		if err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
		}
		ansByQ = make(map[string]*store.Answer)
		for ansRows.Next() {
			var a store.Answer
			var correct int
			if err := ansRows.Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt); err != nil {
				ansRows.Close()
				return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
			}
			a.Correct = correct == 1
			ansByQ[a.QuestionID] = &a
		}
		if err := ansRows.Close(); err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
		}
	}

	for i := range out {
		out[i].Options = optsByQ[out[i].ID]
		if ansByQ != nil {
			out[i].Answer = ansByQ[out[i].ID]
		}
	}
	return out, nil
}

// GetQuestion retrieves a single question by ID with options. It does not populate
// the user's answer — the caller sets q.Answer if needed.
func (s *Store) GetQuestion(id string) (*store.Question, error) {
	var q store.Question
	err := s.db.QueryRow(`
		SELECT id, lesson_id, prompt, correct_key, question_type, section_tag, sort_order
		FROM questions WHERE id = ?
	`, id).Scan(&q.ID, &q.LessonID, &q.Prompt, &q.CorrectKey, &q.QuestionType, &q.SectionTag, &q.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetQuestion: %w", err)
	}
	q.Options, err = s.listOptions(q.ID)
	if err != nil {
		return nil, fmt.Errorf("GetQuestion: %w", err)
	}
	return &q, nil
}

// ListGlossaryTerms returns all glossary terms ordered by sort_order.
func (s *Store) ListGlossaryTerms() ([]store.GlossaryTerm, error) {
	rows, err := s.db.Query(`SELECT id, term, definition, sort_order FROM glossary_terms ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("ListGlossaryTerms: %w", err)
	}
	defer rows.Close()
	var out []store.GlossaryTerm
	for rows.Next() {
		var t store.GlossaryTerm
		if err := rows.Scan(&t.ID, &t.Term, &t.Definition, &t.SortOrder); err != nil {
			return nil, fmt.Errorf("ListGlossaryTerms: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListReferences returns all external references ordered by title.
func (s *Store) ListReferences() ([]store.Reference, error) {
	rows, err := s.db.Query(`SELECT id, title, url, notes, COALESCE(lesson_id, '') FROM references_ext ORDER BY title`)
	if err != nil {
		return nil, fmt.Errorf("ListReferences: %w", err)
	}
	defer rows.Close()
	var out []store.Reference
	for rows.Next() {
		var r store.Reference
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.Notes, &r.LessonID); err != nil {
			return nil, fmt.Errorf("ListReferences: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListExercises returns all exercises with submission status for the given user, ordered by sort_order.
func (s *Store) ListExercises(userID string) ([]store.Exercise, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.lesson_id, e.title, e.path, e.instructions, e.sort_order,
			CASE WHEN s.exercise_id IS NOT NULL THEN 1 ELSE 0 END,
			COALESCE(s.correct, 1)
		FROM exercises e
		LEFT JOIN exercise_submissions s ON s.exercise_id = e.id AND s.user_id = ?
		ORDER BY e.sort_order
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("ListExercises: %w", err)
	}
	defer rows.Close()
	var out []store.Exercise
	for rows.Next() {
		var ex store.Exercise
		var submitted int
		var correct int
		if err := rows.Scan(&ex.ID, &ex.LessonID, &ex.Title, &ex.Path, &ex.Instructions, &ex.SortOrder, &submitted, &correct); err != nil {
			return nil, fmt.Errorf("ListExercises: %w", err)
		}
		ex.Submitted = submitted == 1
		ex.Correct = correct == 1
		out = append(out, ex)
	}
	return out, rows.Err()
}

// ListExercisesByLesson returns exercises filtered by lesson ID for the given user.
func (s *Store) ListExercisesByLesson(userID, lessonID string) ([]store.Exercise, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.lesson_id, e.title, e.path, e.instructions, e.sort_order,
			CASE WHEN s.exercise_id IS NOT NULL THEN 1 ELSE 0 END,
			COALESCE(s.correct, 1)
		FROM exercises e
		LEFT JOIN exercise_submissions s ON s.exercise_id = e.id AND s.user_id = ?
		WHERE e.lesson_id = ?
		ORDER BY e.sort_order
	`, userID, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListExercisesByLesson: %w", err)
	}
	defer rows.Close()
	var out []store.Exercise
	for rows.Next() {
		var ex store.Exercise
		var submitted int
		var correct int
		if err := rows.Scan(&ex.ID, &ex.LessonID, &ex.Title, &ex.Path, &ex.Instructions, &ex.SortOrder, &submitted, &correct); err != nil {
			return nil, fmt.Errorf("ListExercisesByLesson: %w", err)
		}
		ex.Submitted = submitted == 1
		ex.Correct = correct == 1
		out = append(out, ex)
	}
	return out, rows.Err()
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

// ListActiveInsights returns all active insights, optionally filtered by kind.
func (s *Store) ListActiveInsights(kind string) ([]store.Insight, error) {
	q := `SELECT id, title, body, kind, active, created_at FROM insights WHERE active = 1`
	args := []any{}
	if kind != "" {
		q += ` AND kind = ?`
		args = append(args, kind)
	}
	q += ` ORDER BY created_at`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("ListActiveInsights: %w", err)
	}
	defer rows.Close()
	var out []store.Insight
	for rows.Next() {
		var ins store.Insight
		var active int
		if err := rows.Scan(&ins.ID, &ins.Title, &ins.Body, &ins.Kind, &active, &ins.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListActiveInsights: %w", err)
		}
		ins.Active = active == 1
		out = append(out, ins)
	}
	return out, rows.Err()
}

// GetMission returns the learner mission statement.
func (s *Store) GetMission() (*store.Mission, error) {
	var m store.Mission
	err := s.db.QueryRow(`SELECT why, success_criteria, constraints_text FROM mission WHERE id = 1`).Scan(
		&m.Why, &m.SuccessCriteria, &m.ConstraintsText,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetMission: %w", err)
	}
	return &m, nil
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
