// Package store provides the SQLite-backed data access layer for go-learn.
// It handles schema migrations, content import, lesson and quiz CRUD,
// progress tracking, and the bundle-based authoring workflow.
package store

import "html/template"


// Lesson represents a single lesson in the curriculum.
// Lessons are ordered by SortOrder.
type Lesson struct {
	ID        string
	Title     string
	Slug      string
	Summary   string
	SortOrder int
}

// LessonSection is a named section within a lesson containing HTML body.
type LessonSection struct {
	ID        string
	LessonID  string
	Heading   string
	BodyHTML  template.HTML
	SortOrder int
}

// Question is a quiz question. QuestionType is "choice" or "text".
// SectionTag is "quiz" or "review".
type Question struct {
	ID           string
	LessonID     string
	Prompt       string
	CorrectKey   string
	QuestionType string
	SectionTag   string
	SortOrder    int
	Options      []QuestionOption
	Answer       *Answer
}

// QuestionOption is one answer choice for a multiple-choice question.
type QuestionOption struct {
	QuestionID string
	OptionKey  string
	Label      string
	IsCorrect  bool
	SortOrder  int
}

// GlossaryTerm maps a term to its definition for the reference page.
type GlossaryTerm struct {
	ID         string
	Term       string
	Definition string
	SortOrder  int
}

// Reference is an external link associated with a lesson.
type Reference struct {
	ID       string
	Title    string
	URL      string
	Notes    string
	LessonID string
}

// Exercise is a hands-on programming exercise tied to a lesson.
// Submitted indicates the learner has submitted output for this exercise.
type Exercise struct {
	ID           string
	LessonID     string
	Title        string
	Path         string
	Instructions string
	SortOrder    int
	Submitted    bool
}

// Insight is a teaching observation surfaced on the dashboard.
type Insight struct {
	ID        string
	Title     string
	Body      string
	Kind      string
	Active    bool
	CreatedAt string
}

// Mission holds the learner goal statement, success criteria, and constraints.
type Mission struct {
	Why              string
	SuccessCriteria  string
	ConstraintsText  string
}

// LessonProgress aggregates a learner progress through a single lesson.
type LessonProgress struct {
	LessonID         string
	Title            string
	Slug             string
	SortOrder        int
	QuestionsTotal   int
	QuestionsAnswered  int
	QuestionsCorrect int
	ExerciseTotal    int
	ExercisesDone    int
}

// DashboardStats provides roll-up numbers across all lessons for the home page.
type DashboardStats struct {
	LessonsTotal       int
	QuestionsTotal     int
	QuestionsAnswered  int
	QuestionsCorrect   int
	ExercisesTotal     int
	ExercisesSubmitted int
}
