//go:build js && wasm

package d1store

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/kartikkabadi/go-learn/internal/store"
)

// contentCache holds all static lesson content in memory.
// On Cloudflare Workers, this persists for the lifetime of the isolate,
// eliminating D1 round-trips for content that only changes on deploy.
type contentCache struct {
	lessons       []store.Lesson
	bySlug        map[string]*store.Lesson
	byID          map[string]*store.Lesson
	sections      map[string][]store.LessonSection // keyed by lessonID
	questions     map[string][]store.Question      // keyed by lessonID (options populated, Answer nil)
	questionsByID map[string]*store.Question       // keyed by question ID
	exercises     []store.Exercise                 // static exercise definitions (Submitted/Correct zero)
	exByLesson    map[string][]store.Exercise      // keyed by lessonID
	glossary      []store.GlossaryTerm
	references    []store.Reference
	insights      []store.Insight
	mission       *store.Mission
}

type cacheState struct {
	data    *contentCache
	loadErr error
}

var (
	cacheMu   sync.Mutex
	cacheOnce cacheState
)

// loadCache loads all static content from D1 into memory. Called once per isolate.
func (s *Store) loadCache() (*contentCache, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cacheOnce.data != nil {
		return cacheOnce.data, nil
	}
	if cacheOnce.loadErr != nil {
		return nil, cacheOnce.loadErr
	}

	c := &contentCache{
		bySlug:        make(map[string]*store.Lesson),
		byID:          make(map[string]*store.Lesson),
		sections:      make(map[string][]store.LessonSection),
		questions:     make(map[string][]store.Question),
		questionsByID: make(map[string]*store.Question),
		exByLesson:    make(map[string][]store.Exercise),
	}

	if err := s.loadLessons(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadSections(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadQuestions(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadExercises(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadGlossary(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadReferences(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadInsights(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}
	if err := s.loadMission(c); err != nil {
		cacheOnce.loadErr = err
		return nil, err
	}

	cacheOnce.data = c
	return c, nil
}

func (s *Store) loadLessons(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, title, slug, summary, sort_order FROM lessons ORDER BY sort_order`)
	if err != nil {
		return fmt.Errorf("cache lessons: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l store.Lesson
		if err := rows.Scan(&l.ID, &l.Title, &l.Slug, &l.Summary, &l.SortOrder); err != nil {
			return fmt.Errorf("cache lessons: %w", err)
		}
		c.lessons = append(c.lessons, l)
	}
	for i := range c.lessons {
		l := &c.lessons[i]
		c.byID[l.ID] = l
		if l.Slug != "" {
			c.bySlug[l.Slug] = l
		}
	}
	return rows.Err()
}

func (s *Store) loadSections(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, lesson_id, heading, body_html, sort_order FROM lesson_sections ORDER BY lesson_id, sort_order`)
	if err != nil {
		return fmt.Errorf("cache sections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sec store.LessonSection
		if err := rows.Scan(&sec.ID, &sec.LessonID, &sec.Heading, &sec.BodyHTML, &sec.SortOrder); err != nil {
			return fmt.Errorf("cache sections: %w", err)
		}
		c.sections[sec.LessonID] = append(c.sections[sec.LessonID], sec)
	}
	return rows.Err()
}

func (s *Store) loadQuestions(c *contentCache) error {
	// Load questions ordered by lesson, then options ordered by question.
	rows, err := s.db.Query(`
		SELECT q.id, q.lesson_id, q.prompt, q.correct_key, q.question_type, q.section_tag, q.sort_order,
			o.question_id, o.option_key, o.label, o.is_correct, o.sort_order
		FROM questions q
		LEFT JOIN question_options o ON o.question_id = q.id
		ORDER BY q.lesson_id, q.sort_order, o.sort_order
	`)
	if err != nil {
		return fmt.Errorf("cache questions: %w", err)
	}
	defer rows.Close()
	// Track question index within each lesson's slice for option appending.
	qIdx := make(map[string]int)
	for rows.Next() {
		var q store.Question
		var optQID sql.NullString
		var optKey, optLabel sql.NullString
		var optCorrect sql.NullInt64
		var optOrder sql.NullInt64
		if err := rows.Scan(
			&q.ID, &q.LessonID, &q.Prompt, &q.CorrectKey, &q.QuestionType, &q.SectionTag, &q.SortOrder,
			&optQID, &optKey, &optLabel, &optCorrect, &optOrder,
		); err != nil {
			return fmt.Errorf("cache questions: %w", err)
		}
		list := c.questions[q.LessonID]
		if len(list) == 0 || list[len(list)-1].ID != q.ID {
			c.questions[q.LessonID] = append(list, q)
			idx := len(c.questions[q.LessonID]) - 1
			qIdx[q.ID] = idx
			c.questionsByID[q.ID] = &c.questions[q.LessonID][idx]
		}
		if optQID.Valid {
			idx := qIdx[q.ID]
			o := store.QuestionOption{
				QuestionID: optQID.String,
				OptionKey:  optKey.String,
				Label:      optLabel.String,
				IsCorrect:  optCorrect.Int64 == 1,
				SortOrder:  int(optOrder.Int64),
			}
			c.questions[q.LessonID][idx].Options = append(c.questions[q.LessonID][idx].Options, o)
		}
	}
	return rows.Err()
}

func (s *Store) loadExercises(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, lesson_id, title, path, instructions, sort_order FROM exercises ORDER BY sort_order`)
	if err != nil {
		return fmt.Errorf("cache exercises: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ex store.Exercise
		if err := rows.Scan(&ex.ID, &ex.LessonID, &ex.Title, &ex.Path, &ex.Instructions, &ex.SortOrder); err != nil {
			return fmt.Errorf("cache exercises: %w", err)
		}
		c.exercises = append(c.exercises, ex)
		c.exByLesson[ex.LessonID] = append(c.exByLesson[ex.LessonID], ex)
	}
	return rows.Err()
}

func (s *Store) loadGlossary(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, term, definition, sort_order FROM glossary_terms ORDER BY sort_order`)
	if err != nil {
		return fmt.Errorf("cache glossary: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t store.GlossaryTerm
		if err := rows.Scan(&t.ID, &t.Term, &t.Definition, &t.SortOrder); err != nil {
			return fmt.Errorf("cache glossary: %w", err)
		}
		c.glossary = append(c.glossary, t)
	}
	return rows.Err()
}

func (s *Store) loadReferences(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, title, url, notes, COALESCE(lesson_id, '') FROM references_ext ORDER BY title`)
	if err != nil {
		return fmt.Errorf("cache references: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r store.Reference
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.Notes, &r.LessonID); err != nil {
			return fmt.Errorf("cache references: %w", err)
		}
		c.references = append(c.references, r)
	}
	return rows.Err()
}

func (s *Store) loadInsights(c *contentCache) error {
	rows, err := s.db.Query(`SELECT id, title, body, kind, active, created_at FROM insights WHERE active = 1 ORDER BY created_at`)
	if err != nil {
		return fmt.Errorf("cache insights: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ins store.Insight
		var active int
		if err := rows.Scan(&ins.ID, &ins.Title, &ins.Body, &ins.Kind, &active, &ins.CreatedAt); err != nil {
			return fmt.Errorf("cache insights: %w", err)
		}
		ins.Active = active == 1
		c.insights = append(c.insights, ins)
	}
	return rows.Err()
}

func (s *Store) loadMission(c *contentCache) error {
	var m store.Mission
	err := s.db.QueryRow(`SELECT why, success_criteria, constraints_text FROM mission WHERE id = 1`).Scan(
		&m.Why, &m.SuccessCriteria, &m.ConstraintsText,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cache mission: %w", err)
	}
	c.mission = &m
	return nil
}
