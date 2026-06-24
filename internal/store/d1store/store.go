//go:build js && wasm

// Package d1store implements the store.Store interface using
// Cloudflare D1 as the backing database. Build with GOOS=js GOARCH=wasm
// for deployment on Cloudflare Workers via syumai/workers.
package d1store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kartikkabadi/go-learn/internal/content"
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

// SaveAnswer records a quiz answer, evaluates correctness, and returns the result.
func (s *Store) SaveAnswer(userID, questionID, pickedKey, pickedLabel string) (store.Answer, error) {
	q := content.QuestionsByID[questionID]
	if q == nil {
		return store.Answer{}, fmt.Errorf("unknown question %q", questionID)
	}

	correct := false
	switch q.QuestionType {
	case "text":
		correct = strings.EqualFold(strings.TrimSpace(pickedKey), strings.TrimSpace(q.CorrectKey))
	case "choice":
		for _, opt := range q.Options {
			if opt.OptionKey == pickedKey {
				correct = opt.IsCorrect
				break
			}
		}
	default:
		return store.Answer{}, fmt.Errorf("unsupported question type %q", q.QuestionType)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`
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

// ListAnswers returns all answers for a user, joined with embedded lesson/question data.
// Queries only the answers table from D1 (1 query); lesson/question metadata comes from
// embedded content (0 D1 queries).
func (s *Store) ListAnswers(userID string) ([]store.AnswerRow, error) {
	rows, err := s.db.Query(`
		SELECT question_id, picked_key, picked_label, correct, answered_at
		FROM answers WHERE user_id = ?
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
		); err != nil {
			return nil, fmt.Errorf("scan answer row: %w", err)
		}
		row.Correct = correct == 1
		q := content.QuestionsByID[row.QuestionID]
		if q != nil {
			row.LessonID = q.LessonID
			row.Prompt = q.Prompt
			row.CorrectKey = q.CorrectKey
			for _, opt := range q.Options {
				if opt.IsCorrect {
					row.CorrectLabel = opt.Label
					break
				}
			}
			if row.CorrectLabel == "" {
				row.CorrectLabel = q.CorrectKey
			}
			if l := content.LessonsByID[q.LessonID]; l != nil {
				row.LessonTitle = l.Title
			}
		}
		out = append(out, row)
	}
	// Sort by lesson sort_order, then question sort_order (matching SQLite store).
	sort.SliceStable(out, func(i, j int) bool {
		li, lj := content.LessonsByID[out[i].LessonID], content.LessonsByID[out[j].LessonID]
		if li == nil || lj == nil {
			return false
		}
		if li.SortOrder != lj.SortOrder {
			return li.SortOrder < lj.SortOrder
		}
		qi, qj := content.QuestionsByID[out[i].QuestionID], content.QuestionsByID[out[j].QuestionID]
		if qi == nil || qj == nil {
			return false
		}
		return qi.SortOrder < qj.SortOrder
	})
	return out, rows.Err()
}

// ListLessons returns all lessons from embedded content (0 D1 queries).
func (s *Store) ListLessons() ([]store.Lesson, error) {
	return content.Lessons, nil
}

// GetLesson retrieves a single lesson by ID from embedded content.
func (s *Store) GetLesson(id string) (*store.Lesson, error) {
	return content.LessonsByID[id], nil
}

// GetLessonBySlug retrieves a single lesson by slug from embedded content.
func (s *Store) GetLessonBySlug(slug string) (*store.Lesson, error) {
	return content.LessonsBySlug[slug], nil
}

// ListLessonSections returns sections for a lesson from embedded content.
func (s *Store) ListLessonSections(lessonID string) ([]store.LessonSection, error) {
	return content.Sections[lessonID], nil
}

// ListQuestionsByLesson returns questions for a lesson from embedded content
// (options populated). Fetches user's answers in 1 D1 query if logged in.
// Anonymous requests = 0 D1 queries.
func (s *Store) ListQuestionsByLesson(userID, lessonID string) ([]store.Question, error) {
	cached := content.Questions[lessonID]
	out := make([]store.Question, len(cached))
	copy(out, cached)
	if len(out) == 0 || userID == "" {
		return out, nil
	}

	// Build question ID list from embedded content (not D1 — content tables may be stale).
	placeholders := make([]string, len(out))
	args := make([]any, 0, len(out)+1)
	args = append(args, userID)
	for i, q := range out {
		placeholders[i] = "?"
		args = append(args, q.ID)
	}
	query := fmt.Sprintf(`
		SELECT question_id, picked_key, picked_label, correct, answered_at
		FROM answers
		WHERE user_id = ? AND question_id IN (%s)
	`, strings.Join(placeholders, ","))
	ansRows, err := s.db.Query(query, args...)
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
	if err := ansRows.Err(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByLesson answers: %w", err)
	}
	for i := range out {
		out[i].Answer = ansByQ[out[i].ID]
	}
	return out, nil
}

// GetQuestion retrieves a single question by ID with options from embedded content.
func (s *Store) GetQuestion(id string) (*store.Question, error) {
	return content.QuestionsByID[id], nil
}

// ListGlossaryTerms returns glossary terms from embedded content.
func (s *Store) ListGlossaryTerms() ([]store.GlossaryTerm, error) {
	return content.Glossary, nil
}

// ListReferences returns external references from embedded content.
func (s *Store) ListReferences() ([]store.Reference, error) {
	return content.References, nil
}

// ListExercises returns all exercises from embedded content. Fetches
// submission status in 1 D1 query if logged in. Anonymous = 0 queries.
func (s *Store) ListExercises(userID string) ([]store.Exercise, error) {
	out := make([]store.Exercise, len(content.Exercises))
	copy(out, content.Exercises)
	if userID == "" {
		return out, nil
	}
	return s.attachSubmissions(out, userID)
}

// ListExercisesByLesson returns exercises for a lesson from embedded content.
// Fetches submission status in 1 D1 query if logged in. Anonymous = 0 queries.
func (s *Store) ListExercisesByLesson(userID, lessonID string) ([]store.Exercise, error) {
	cached := content.ExercisesByLesson[lessonID]
	out := make([]store.Exercise, len(cached))
	copy(out, cached)
	if userID == "" {
		return out, nil
	}
	return s.attachSubmissions(out, userID)
}

// attachSubmissions fetches the user's submissions in 1 D1 query and merges.
func (s *Store) attachSubmissions(out []store.Exercise, userID string) ([]store.Exercise, error) {
	if len(out) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(`SELECT exercise_id, correct FROM exercise_submissions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("exercises submissions: %w", err)
	}
	defer rows.Close()
	subByEx := make(map[string]bool)
	correctByEx := make(map[string]bool)
	for rows.Next() {
		var exID string
		var correct int
		if err := rows.Scan(&exID, &correct); err != nil {
			return nil, fmt.Errorf("exercises submissions: %w", err)
		}
		subByEx[exID] = true
		correctByEx[exID] = correct == 1
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("exercises submissions: %w", err)
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

// ListActiveInsights returns active insights from embedded content,
// optionally filtered by kind. 0 D1 queries.
func (s *Store) ListActiveInsights(kind string) ([]store.Insight, error) {
	if kind == "" {
		return content.Insights, nil
	}
	var out []store.Insight
	for _, ins := range content.Insights {
		if ins.Kind == kind {
			out = append(out, ins)
		}
	}
	return out, nil
}

// GetMission returns the mission from embedded content. 0 D1 queries.
func (s *Store) GetMission() (*store.Mission, error) {
	return content.Mission, nil
}

// LessonProgress returns progress stats per lesson. Lesson/question/exercise
// totals come from embedded content (0 D1 queries). User's answered/correct/
// done counts come from 1 D1 query. Anonymous = 0 D1 queries.
func (s *Store) LessonProgress(userID string) ([]store.LessonProgress, error) {
	out := make([]store.LessonProgress, len(content.Lessons))
	for i, l := range content.Lessons {
		out[i] = store.LessonProgress{
			LessonID:      l.ID,
			Title:         l.Title,
			Slug:          l.Slug,
			SortOrder:     l.SortOrder,
			QuestionsTotal: len(content.Questions[l.ID]),
			ExerciseTotal: len(content.ExercisesByLesson[l.ID]),
		}
	}
	if userID == "" {
		return out, nil
	}

	// 1 query: all user answers (no join — lesson_id from embedded content).
	rows, err := s.db.Query(`
		SELECT question_id, correct FROM answers WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("LessonProgress: %w", err)
	}
	defer rows.Close()
	ansByLesson := make(map[string][2]int) // [answered, correct]
	for rows.Next() {
		var qid string
		var correct int
		if err := rows.Scan(&qid, &correct); err != nil {
			return nil, fmt.Errorf("LessonProgress: %w", err)
		}
		q := content.QuestionsByID[qid]
		if q == nil {
			continue
		}
		ac := ansByLesson[q.LessonID]
		ac[0]++
		ac[1] += correct
		ansByLesson[q.LessonID] = ac
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LessonProgress: %w", err)
	}

	// 1 query: all exercise submissions (no join — lesson_id from embedded content).
	esRows, err := s.db.Query(`
		SELECT exercise_id FROM exercise_submissions WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("LessonProgress exercises: %w", err)
	}
	defer esRows.Close()
	exByLesson := make(map[string]int)
	for esRows.Next() {
		var eid string
		if err := esRows.Scan(&eid); err != nil {
			return nil, fmt.Errorf("LessonProgress exercises: %w", err)
		}
		lid := exerciseLessonID(eid)
		if lid != "" {
			exByLesson[lid]++
		}
	}
	if err := esRows.Err(); err != nil {
		return nil, fmt.Errorf("LessonProgress exercises: %w", err)
	}

	for i := range out {
		if ac, ok := ansByLesson[out[i].LessonID]; ok {
			out[i].QuestionsAnswered = ac[0]
			out[i].QuestionsCorrect = ac[1]
		}
		out[i].ExercisesDone = exByLesson[out[i].LessonID]
	}
	return out, nil
}

// UserData fetches all user-specific data in 2 D1 queries (answers + submissions).
// Anonymous users get an empty UserData with 0 D1 queries.
func (s *Store) UserData(userID string) (*store.UserData, error) {
	ud := &store.UserData{
		AnswersByLesson:     make(map[string][2]int),
		SubmissionsByEx:     make(map[string]bool),
		CorrectByEx:         make(map[string]bool),
		SubmissionsByLesson: make(map[string]int),
	}
	if userID == "" {
		return ud, nil
	}

	// Query 1: all user answers (no join — lesson_id from embedded content).
	rows, err := s.db.Query(`
		SELECT question_id, correct FROM answers WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("UserData answers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var qid string
		var correct int
		if err := rows.Scan(&qid, &correct); err != nil {
			return nil, fmt.Errorf("UserData answers: %w", err)
		}
		q := content.QuestionsByID[qid]
		if q == nil {
			continue
		}
		ac := ud.AnswersByLesson[q.LessonID]
		ac[0]++
		ac[1] += correct
		ud.AnswersByLesson[q.LessonID] = ac
		ud.TotalAnswered++
		ud.TotalCorrect += correct
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserData answers: %w", err)
	}

	// Query 2: all exercise submissions (no join — lesson_id from embedded content).
	esRows, err := s.db.Query(`
		SELECT exercise_id, correct FROM exercise_submissions WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("UserData submissions: %w", err)
	}
	defer esRows.Close()
	for esRows.Next() {
		var exID string
		var correct int
		if err := esRows.Scan(&exID, &correct); err != nil {
			return nil, fmt.Errorf("UserData submissions: %w", err)
		}
		ud.SubmissionsByEx[exID] = true
		ud.CorrectByEx[exID] = correct == 1
		lid := exerciseLessonID(exID)
		if lid != "" {
			ud.SubmissionsByLesson[lid]++
		}
		ud.TotalSubmitted++
	}
	if err := esRows.Err(); err != nil {
		return nil, fmt.Errorf("UserData submissions: %w", err)
	}

	return ud, nil
}

// LessonCounts returns question and exercise counts per lessonID from embedded
// content. 0 D1 queries.
func (s *Store) LessonCounts() (map[string]int, map[string]int, error) {
	qCounts := make(map[string]int, len(content.Lessons))
	eCounts := make(map[string]int, len(content.Lessons))
	for _, l := range content.Lessons {
		qCounts[l.ID] = len(content.Questions[l.ID])
		eCounts[l.ID] = len(content.ExercisesByLesson[l.ID])
	}
	return qCounts, eCounts, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// exerciseLessonID returns the lesson_id for an exercise ID from embedded content.
var exerciseLessonCache map[string]string

func exerciseLessonID(exID string) string {
	if exerciseLessonCache == nil {
		exerciseLessonCache = make(map[string]string, len(content.Exercises))
		for _, ex := range content.Exercises {
			exerciseLessonCache[ex.ID] = ex.LessonID
		}
	}
	return exerciseLessonCache[exID]
}
