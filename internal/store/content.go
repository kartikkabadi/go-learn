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

// ListQuestionsByLesson returns all questions for a lesson, with options and answers populated.
func (s *SQLiteStore) ListQuestionsByLesson(lessonID string) ([]Question, error) {
	rows, err := s.db.Query(`
		SELECT id, lesson_id, prompt, correct_key, question_type, section_tag, sort_order
		FROM questions WHERE lesson_id = ? ORDER BY sort_order
	`, lessonID)
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
	}

	var out []Question
	for rows.Next() {
		var q Question
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

	for i := range out {
		out[i].Options, err = s.listOptions(out[i].ID)
		if err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
		}
		out[i].Answer, err = s.GetAnswer(out[i].ID)
		if err != nil {
			return nil, fmt.Errorf("ListQuestionsByLesson: %w", err)
		}
	}
	return out, nil
}

// GetQuestion retrieves a single question by ID with options and answer.
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
	q.Answer, err = s.GetAnswer(q.ID)
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

// ListExercises returns all exercises with submission status, ordered by sort_order.
func (s *SQLiteStore) ListExercises() ([]Exercise, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.lesson_id, e.title, e.path, e.instructions, e.sort_order,
			CASE WHEN s.exercise_id IS NOT NULL THEN 1 ELSE 0 END
		FROM exercises e
		LEFT JOIN exercise_submissions s ON s.exercise_id = e.id
		ORDER BY e.sort_order
	`)
	if err != nil {
		return nil, fmt.Errorf("ListExercises: %w", err)
	}
	defer rows.Close()
	var out []Exercise
	for rows.Next() {
		var ex Exercise
		var submitted int
		if err := rows.Scan(&ex.ID, &ex.LessonID, &ex.Title, &ex.Path, &ex.Instructions, &ex.SortOrder, &submitted); err != nil {
			return nil, fmt.Errorf("ListExercises: %w", err)
		}
		ex.Submitted = submitted == 1
		out = append(out, ex)
	}
	return out, rows.Err()
}

// ListExercisesByLesson returns exercises filtered by lesson ID.
func (s *SQLiteStore) ListExercisesByLesson(lessonID string) ([]Exercise, error) {
	all, err := s.ListExercises()
	if err != nil {
		return nil, fmt.Errorf("ListExercisesByLesson: %w", err)
	}
	var out []Exercise
	for _, ex := range all {
		if ex.LessonID == lessonID {
			out = append(out, ex)
		}
	}
	return out, nil
}

// SaveExerciseSubmission stores terminal output for an exercise, upserting by exercise ID.
func (s *SQLiteStore) SaveExerciseSubmission(exerciseID, output string) error {
	now := timeNowRFC3339()
	_, err := s.db.Exec(`
		INSERT INTO exercise_submissions (exercise_id, output, submitted_at)
		VALUES (?, ?, ?)
		ON CONFLICT(exercise_id) DO UPDATE SET
			output = excluded.output,
			submitted_at = excluded.submitted_at
	`, exerciseID, output, now)
	return err
}

// GetExerciseSubmission retrieves saved output for an exercise.
func (s *SQLiteStore) GetExerciseSubmission(exerciseID string) (string, bool, error) {
	var output string
	err := s.db.QueryRow(`SELECT output FROM exercise_submissions WHERE exercise_id = ?`, exerciseID).Scan(&output)
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

// LessonProgress returns progress stats for every lesson.
func (s *SQLiteStore) LessonProgress() ([]LessonProgress, error) {
	lessons, err := s.ListLessons()
	if err != nil {
		return nil, fmt.Errorf("LessonProgress: %w", err)
	}
	var out []LessonProgress
	for _, l := range lessons {
		lp := LessonProgress{LessonID: l.ID, Title: l.Title, Slug: l.Slug, SortOrder: l.SortOrder}
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM questions WHERE lesson_id = ?`, l.ID).Scan(&lp.QuestionsTotal); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		if err := s.db.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(a.correct), 0)
			FROM answers a JOIN questions q ON q.id = a.question_id
			WHERE q.lesson_id = ?
		`, l.ID).Scan(&lp.QuestionsAnswered, &lp.QuestionsCorrect); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM exercises WHERE lesson_id = ?`, l.ID).Scan(&lp.ExerciseTotal); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		if err := s.db.QueryRow(`
			SELECT COUNT(*) FROM exercise_submissions s
			JOIN exercises e ON e.id = s.exercise_id WHERE e.lesson_id = ?
		`, l.ID).Scan(&lp.ExercisesDone); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		out = append(out, lp)
	}
	return out, nil
}

// DashboardStats returns aggregate progress numbers across all lessons.
func (s *SQLiteStore) DashboardStats() (DashboardStats, error) {
	var st DashboardStats
	err := s.db.QueryRow(`SELECT COUNT(*) FROM lessons`).Scan(&st.LessonsTotal)
	if err != nil {
		return st, err
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM questions`).Scan(&st.QuestionsTotal)
	if err != nil {
		return st, err
	}
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(correct), 0) FROM answers`).Scan(
		&st.QuestionsAnswered, &st.QuestionsCorrect,
	)
	if err != nil {
		return st, err
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM exercises`).Scan(&st.ExercisesTotal)
	if err != nil {
		return st, err
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM exercise_submissions`).Scan(&st.ExercisesSubmitted)
	return st, err
}

func timeNowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
