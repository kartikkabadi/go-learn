package store

import (
	"database/sql"
	"fmt"
	"time"
)

// ListLessons returns all lessons ordered by sort_order.
func (s *SQLiteStore) ListLessons() ([]Lesson, error) {
	rows, err := s.db.Query(`SELECT id, title, slug, summary, sort_order FROM lessons ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("ListLessons: %w", err)
	}
	defer rows.Close()
	var out []Lesson
	for rows.Next() {
		var l Lesson
		if err := rows.Scan(&l.ID, &l.Title, &l.Slug, &l.Summary, &l.SortOrder); err != nil {
			return nil, fmt.Errorf("ListLessons: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetLesson retrieves a single lesson by ID, or nil if not found.
func (s *SQLiteStore) GetLesson(id string) (*Lesson, error) {
	var l Lesson
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
func (s *SQLiteStore) GetLessonBySlug(slug string) (*Lesson, error) {
	var l Lesson
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
func (s *SQLiteStore) ListLessonSections(lessonID string) ([]LessonSection, error) {
	rows, err := s.db.Query(`
		SELECT id, lesson_id, heading, body_html, sort_order
		FROM lesson_sections WHERE lesson_id = ? ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListLessonSections: %w", err)
	}
	defer rows.Close()
	var out []LessonSection
	for rows.Next() {
		var sec LessonSection
		if err := rows.Scan(&sec.ID, &sec.LessonID, &sec.Heading, &sec.BodyHTML, &sec.SortOrder); err != nil {
			return nil, fmt.Errorf("ListLessonSections: %w", err)
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

// ListQuestionsByLesson returns all questions for a lesson, with options and
// the user's answers populated. Uses 3 queries (questions, options, answers)
// instead of 1+2N to avoid N+1 round-trips.
func (s *SQLiteStore) ListQuestionsByLesson(userID, lessonID string) ([]Question, error) {
	rows, err := s.db.Query(`
		SELECT id, lesson_id, prompt, correct_key, question_type, section_tag, sort_order
		FROM questions WHERE lesson_id = ? ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
	}
	defer rows.Close()

	var out []Question
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.ID, &q.LessonID, &q.Prompt, &q.CorrectKey, &q.QuestionType, &q.SectionTag, &q.SortOrder); err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
		}
		out = append(out, q)
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
	defer optRows.Close()
	optsByQ := make(map[string][]QuestionOption)
	for optRows.Next() {
		var o QuestionOption
		var correct int
		if err := optRows.Scan(&o.QuestionID, &o.OptionKey, &o.Label, &correct, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson options: %w", err)
		}
		o.IsCorrect = correct == 1
		optsByQ[o.QuestionID] = append(optsByQ[o.QuestionID], o)
	}
	if err := optRows.Err(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson options: %w", err)
	}

	// Batch-fetch answers for this user+lesson (only if logged in).
	var ansByQ map[string]*Answer
	if userID != "" {
		ansRows, err := s.db.Query(`
			SELECT question_id, picked_key, picked_label, correct, answered_at
			FROM answers
			WHERE user_id = ? AND question_id IN (SELECT id FROM questions WHERE lesson_id = ?)
		`, userID, lessonID)
		if err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
		}
		defer ansRows.Close()
		ansByQ = make(map[string]*Answer)
		for ansRows.Next() {
			var a Answer
			var correct int
			if err := ansRows.Scan(&a.QuestionID, &a.PickedKey, &a.PickedLabel, &correct, &a.AnsweredAt); err != nil {
				return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
			}
			a.Correct = correct == 1
			ansByQ[a.QuestionID] = &a
		}
		if err := ansRows.Err(); err != nil {
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
func (s *SQLiteStore) GetQuestion(id string) (*Question, error) {
	var q Question
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

func (s *SQLiteStore) listOptions(questionID string) ([]QuestionOption, error) {
	rows, err := s.db.Query(`
		SELECT question_id, option_key, label, is_correct, sort_order
		FROM question_options WHERE question_id = ? ORDER BY sort_order
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("listOptions: %w", err)
	}
	defer rows.Close()
	var out []QuestionOption
	for rows.Next() {
		var o QuestionOption
		var correct int
		if err := rows.Scan(&o.QuestionID, &o.OptionKey, &o.Label, &correct, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("listOptions: %w", err)
		}
		o.IsCorrect = correct == 1
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListGlossaryTerms returns all glossary terms ordered by sort_order.
func (s *SQLiteStore) ListGlossaryTerms() ([]GlossaryTerm, error) {
	rows, err := s.db.Query(`SELECT id, term, definition, sort_order FROM glossary_terms ORDER BY sort_order`)
	if err != nil {
		return nil, fmt.Errorf("ListGlossaryTerms: %w", err)
	}
	defer rows.Close()
	var out []GlossaryTerm
	for rows.Next() {
		var t GlossaryTerm
		if err := rows.Scan(&t.ID, &t.Term, &t.Definition, &t.SortOrder); err != nil {
			return nil, fmt.Errorf("ListGlossaryTerms: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListReferences returns all external references ordered by title.
func (s *SQLiteStore) ListReferences() ([]Reference, error) {
	rows, err := s.db.Query(`SELECT id, title, url, notes, COALESCE(lesson_id, '') FROM references_ext ORDER BY title`)
	if err != nil {
		return nil, fmt.Errorf("ListReferences: %w", err)
	}
	defer rows.Close()
	var out []Reference
	for rows.Next() {
		var r Reference
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.Notes, &r.LessonID); err != nil {
			return nil, fmt.Errorf("ListReferences: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListExercises returns all exercises with submission status for the given user, ordered by sort_order.
func (s *SQLiteStore) ListExercises(userID string) ([]Exercise, error) {
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
	var out []Exercise
	for rows.Next() {
		var ex Exercise
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
func (s *SQLiteStore) ListExercisesByLesson(userID, lessonID string) ([]Exercise, error) {
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
	var out []Exercise
	for rows.Next() {
		var ex Exercise
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
func (s *SQLiteStore) SaveExerciseSubmission(userID, exerciseID, output string, correct bool) error {
	now := timeNowRFC3339()
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
func (s *SQLiteStore) GetExerciseSubmission(userID, exerciseID string) (string, bool, error) {
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
func (s *SQLiteStore) ListActiveInsights(kind string) ([]Insight, error) {
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
	var out []Insight
	for rows.Next() {
		var ins Insight
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
func (s *SQLiteStore) GetMission() (*Mission, error) {
	var m Mission
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
func (s *SQLiteStore) LessonProgress(userID string) ([]LessonProgress, error) {
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
	var out []LessonProgress
	for rows.Next() {
		var lp LessonProgress
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

func timeNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
