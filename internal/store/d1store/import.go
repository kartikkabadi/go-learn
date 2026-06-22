//go:build js && wasm

package d1store

import (
	"fmt"
	"time"

	"github.com/kartikkabadi/go-learn/internal/store"
)

// ImportBundle upserts a lesson, its sections, questions, exercises, references, and seed answers.
// D1 doesn't support transactions, so each operation is an individual Exec.
func (s *Store) ImportBundle(b store.ContentBundle) error {
	var err error

	_, err = s.db.Exec(`
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

	if _, err := s.db.Exec(`DELETE FROM lesson_sections WHERE lesson_id = ?`, b.Lesson.ID); err != nil {
		return fmt.Errorf("delete sections: %w", err)
	}
	for _, sec := range b.Sections {
		_, err = s.db.Exec(`
			INSERT INTO lesson_sections (id, lesson_id, heading, body_html, sort_order)
			VALUES (?, ?, ?, ?, ?)
		`, sec.ID, b.Lesson.ID, sec.Heading, sec.BodyHTML, sec.SortOrder)
		if err != nil {
			return fmt.Errorf("insert section %s: %w", sec.ID, err)
		}
	}

	for _, q := range b.Questions {
		_, err = s.db.Exec(`
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
		if _, err := s.db.Exec(`DELETE FROM question_options WHERE question_id = ?`, q.ID); err != nil {
			return fmt.Errorf("delete options for %s: %w", q.ID, err)
		}
		for _, opt := range q.Options {
			_, err = s.db.Exec(`
				INSERT INTO question_options (question_id, option_key, label, is_correct, sort_order)
				VALUES (?, ?, ?, ?, ?)
			`, q.ID, opt.Key, opt.Label, boolToInt(opt.IsCorrect), opt.SortOrder)
			if err != nil {
				return fmt.Errorf("insert option %s: %w", opt.Key, err)
			}
		}
	}

	for _, ex := range b.Exercises {
		_, err = s.db.Exec(`
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
		_, err = s.db.Exec(`
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

	for _, a := range b.Answers {
		var qType, correctKey string
		if err := s.db.QueryRow(
			`SELECT question_type, correct_key FROM questions WHERE id = ?`, a.QuestionID,
		).Scan(&qType, &correctKey); err != nil {
			return fmt.Errorf("lookup question for answer %s: %w", a.QuestionID, err)
		}
		correct := false
		if qType == "text" {
			correct = a.PickedKey == correctKey
		} else {
			var isCorrect int
			if err := s.db.QueryRow(
				`SELECT is_correct FROM question_options WHERE question_id = ? AND option_key = ?`,
				a.QuestionID, a.PickedKey,
			).Scan(&isCorrect); err != nil {
				return fmt.Errorf("lookup option for answer %s: %w", a.QuestionID, err)
			}
			correct = isCorrect == 1
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
		`, a.QuestionID, a.PickedKey, a.PickedLabel, boolToInt(correct), now)
		if err != nil {
			return fmt.Errorf("upsert answer %s: %w", a.QuestionID, err)
		}
	}

	return nil
}

// ImportMission upserts the learner mission statement.
func (s *Store) ImportMission(b store.MissionBundle) error {
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
func (s *Store) ImportGlossary(b store.GlossaryBundle) error {
	for _, t := range b.Terms {
		_, err := s.db.Exec(`
			INSERT INTO glossary_terms (id, term, definition, sort_order)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				term = excluded.term,
				definition = excluded.definition,
				sort_order = excluded.sort_order
		`, t.ID, t.Term, t.Definition, t.SortOrder)
		if err != nil {
			return fmt.Errorf("upsert glossary term %s: %w", t.ID, err)
		}
	}
	return nil
}

// ImportInsights upserts insights, setting them active.
func (s *Store) ImportInsights(b store.InsightsBundle) error {
	for _, ins := range b.Insights {
		_, err := s.db.Exec(`
			INSERT INTO insights (id, title, body, kind, active, created_at)
			VALUES (?, ?, ?, ?, 1, datetime('now'))
			ON CONFLICT(id) DO UPDATE SET
				title = excluded.title,
				body = excluded.body,
				kind = excluded.kind,
				active = 1
		`, ins.ID, ins.Title, ins.Body, ins.Kind)
		if err != nil {
			return fmt.Errorf("upsert insight %s: %w", ins.ID, err)
		}
	}
	return nil
}
