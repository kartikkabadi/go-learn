package store

import (
	"encoding/json"
	"fmt"
	"os"
)

// ContentBundle is a version-controlled JSON file that defines a lesson with all its content.
type ContentBundle struct {
	Lesson     BundleLesson      `json:"lesson"`
	Sections   []BundleSection   `json:"sections"`
	Questions  []BundleQuestion  `json:"questions"`
	Exercises  []BundleExercise  `json:"exercises"`
	References []BundleReference `json:"references"`
}

// BundleLesson is the JSON structure for lesson metadata in a content bundle.
type BundleLesson struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Summary   string `json:"summary"`
	SortOrder int    `json:"sortOrder"`
}

// BundleSection is a named HTML section within a bundle lesson.
type BundleSection struct {
	ID        string `json:"id"`
	Heading   string `json:"heading"`
	BodyHTML  string `json:"bodyHtml"`
	SortOrder int    `json:"sortOrder"`
}

// BundleQuestion is the JSON structure for a quiz question in a bundle.
type BundleQuestion struct {
	ID           string         `json:"id"`
	Prompt       string         `json:"prompt"`
	CorrectKey   string         `json:"correctKey"`
	QuestionType string         `json:"questionType"`
	SectionTag   string         `json:"sectionTag"`
	SortOrder    int            `json:"sortOrder"`
	Options      []BundleOption `json:"options"`
}

type BundleOption struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	IsCorrect bool   `json:"isCorrect"`
	SortOrder int    `json:"sortOrder"`
}

// BundleExercise links a practice module to a lesson in the bundle format.
type BundleExercise struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	Instructions string `json:"instructions"`
	SortOrder    int    `json:"sortOrder"`
}

type BundleReference struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Notes string `json:"notes"`
}

// MissionBundle is the JSON structure for importing mission data.
type MissionBundle struct {
	Why             string   `json:"why"`
	SuccessCriteria []string `json:"successCriteria"`
	Constraints     []string `json:"constraints"`
}

// InsightsBundle wraps a list of insights for JSON import.
type InsightsBundle struct {
	Insights []BundleInsight `json:"insights"`
}

type BundleInsight struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Kind  string `json:"kind"`
}

// GlossaryBundle wraps a list of glossary terms for JSON import.
type GlossaryBundle struct {
	Terms []BundleGlossaryTerm `json:"terms"`
}

type BundleGlossaryTerm struct {
	ID         string `json:"id"`
	Term       string `json:"term"`
	Definition string `json:"definition"`
	SortOrder  int    `json:"sortOrder"`
}

// LoadContentBundle reads and parses a JSON content bundle from disk.
func LoadContentBundle(path string) (ContentBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ContentBundle{}, err
	}
	var b ContentBundle
	if err := json.Unmarshal(data, &b); err != nil {
		return ContentBundle{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return b, nil
}

// ImportBundle upserts a lesson, its sections, questions, exercises, references, and seed answers.
func (s *SQLiteStore) ImportBundle(b ContentBundle) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO lessons (id, title, slug, summary, sort_order)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			slug = excluded.slug,
			summary = excluded.summary,
			sort_order = excluded.sort_order
	`, b.Lesson.ID, b.Lesson.Title, b.Lesson.Slug, b.Lesson.Summary, b.Lesson.SortOrder)
	if err != nil {
		return fmt.Errorf("upsert lesson: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM lesson_sections WHERE lesson_id = ?`, b.Lesson.ID); err != nil {
		return err
	}
	for _, sec := range b.Sections {
		_, err = tx.Exec(`
			INSERT INTO lesson_sections (id, lesson_id, heading, body_html, sort_order)
			VALUES (?, ?, ?, ?, ?)
		`, sec.ID, b.Lesson.ID, sec.Heading, sec.BodyHTML, sec.SortOrder)
		if err != nil {
			return fmt.Errorf("insert section %s: %w", sec.ID, err)
		}
	}

	for _, q := range b.Questions {
		_, err = tx.Exec(`
			INSERT INTO questions (id, lesson_id, prompt, correct_key, question_type, section_tag, sort_order)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				lesson_id = excluded.lesson_id,
				prompt = excluded.prompt,
				correct_key = excluded.correct_key,
				question_type = excluded.question_type,
				section_tag = excluded.section_tag,
				sort_order = excluded.sort_order
		`, q.ID, b.Lesson.ID, q.Prompt, q.CorrectKey, q.QuestionType, q.SectionTag, q.SortOrder)
		if err != nil {
			return fmt.Errorf("upsert question %s: %w", q.ID, err)
		}
		if _, err := tx.Exec(`DELETE FROM question_options WHERE question_id = ?`, q.ID); err != nil {
			return err
		}
		for _, opt := range q.Options {
			_, err = tx.Exec(`
				INSERT INTO question_options (question_id, option_key, label, is_correct, sort_order)
				VALUES (?, ?, ?, ?, ?)
			`, q.ID, opt.Key, opt.Label, boolToInt(opt.IsCorrect), opt.SortOrder)
			if err != nil {
				return fmt.Errorf("insert option %s: %w", opt.Key, err)
			}
		}
	}

	for _, ex := range b.Exercises {
		_, err = tx.Exec(`
			INSERT INTO exercises (id, lesson_id, title, path, instructions, sort_order)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				lesson_id = excluded.lesson_id,
				title = excluded.title,
				path = excluded.path,
				instructions = excluded.instructions,
				sort_order = excluded.sort_order
		`, ex.ID, b.Lesson.ID, ex.Title, ex.Path, ex.Instructions, ex.SortOrder)
		if err != nil {
			return fmt.Errorf("upsert exercise %s: %w", ex.ID, err)
		}
	}

	for _, ref := range b.References {
		_, err = tx.Exec(`
			INSERT INTO references_ext (id, title, url, notes, lesson_id)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				title = excluded.title,
				url = excluded.url,
				notes = excluded.notes,
				lesson_id = excluded.lesson_id
		`, ref.ID, ref.Title, ref.URL, ref.Notes, b.Lesson.ID)
		if err != nil {
			return fmt.Errorf("upsert reference %s: %w", ref.ID, err)
		}
	}

	// Answers are per-user and created at runtime via SaveAnswer.

	return tx.Commit()
}

// ImportMission upserts the learner mission statement.
func (s *SQLiteStore) ImportMission(b MissionBundle) error {
	criteria := ""
	for i, c := range b.SuccessCriteria {
		if i > 0 {
			criteria += "\n"
		}
		criteria += "- " + c
	}
	constraints := ""
	for i, c := range b.Constraints {
		if i > 0 {
			constraints += "\n"
		}
		constraints += "- " + c
	}
	_, err := s.db.Exec(`
		INSERT INTO mission (id, why, success_criteria, constraints_text)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			why = excluded.why,
			success_criteria = excluded.success_criteria,
			constraints_text = excluded.constraints_text
	`, b.Why, criteria, constraints)
	return err
}

// ImportGlossary upserts glossary terms from a bundle.
func (s *SQLiteStore) ImportGlossary(b GlossaryBundle) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range b.Terms {
		_, err = tx.Exec(`
			INSERT INTO glossary_terms (id, term, definition, sort_order)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				term = excluded.term,
				definition = excluded.definition,
				sort_order = excluded.sort_order
		`, t.ID, t.Term, t.Definition, t.SortOrder)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ImportInsights upserts insights, setting them active.
func (s *SQLiteStore) ImportInsights(b InsightsBundle) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, ins := range b.Insights {
		_, err = tx.Exec(`
			INSERT INTO insights (id, title, body, kind, active, created_at)
			VALUES (?, ?, ?, ?, 1, datetime('now'))
			ON CONFLICT(id) DO UPDATE SET
				title = excluded.title,
				body = excluded.body,
				kind = excluded.kind,
				active = 1
		`, ins.ID, ins.Title, ins.Body, ins.Kind)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
