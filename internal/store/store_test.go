package store_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/store"
)

func TestSaveAnswerChoice(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	bundle := store.ContentBundle{
		Lesson: store.BundleLesson{ID: "t1", Title: "Test", SortOrder: 1},
		Questions: []store.BundleQuestion{{
			ID: "t1:q1", Prompt: "pick", CorrectKey: "a", QuestionType: "choice", SortOrder: 1,
			Options: []store.BundleOption{
				{Key: "a", Label: "right", IsCorrect: true, SortOrder: 1},
				{Key: "b", Label: "wrong", IsCorrect: false, SortOrder: 2},
			},
		}},
	}
	if err := st.ImportBundle(bundle); err != nil {
		t.Fatal(err)
	}

	ans, err := st.SaveAnswer(testUser(t, st), "t1:q1", "a", "right")
	if err != nil {
		t.Fatal(err)
	}
	if !ans.Correct {
		t.Fatal("expected correct")
	}

	ans, err = st.SaveAnswer(testUser(t, st), "t1:q1", "b", "wrong")
	if err != nil {
		t.Fatal(err)
	}
	if ans.Correct {
		t.Fatal("expected incorrect")
	}
}

func TestSaveAnswerText(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	bundle := store.ContentBundle{
		Lesson: store.BundleLesson{ID: "t2", Title: "Test", SortOrder: 1},
		Questions: []store.BundleQuestion{{
			ID: "t2:q1", Prompt: "type", CorrectKey: "big", QuestionType: "text", SortOrder: 1,
		}},
	}
	if err := st.ImportBundle(bundle); err != nil {
		t.Fatal(err)
	}

	ans, err := st.SaveAnswer(testUser(t, st), "t2:q1", "big", "big")
	if err != nil {
		t.Fatal(err)
	}
	if !ans.Correct {
		t.Fatal("expected correct text answer")
	}
}

func testStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// testUser creates a fresh user for the calling test (idempotent within a test)
// and returns its ID for use in user-scoped calls. The same test gets the same
// user ID on repeated calls within that test.
func testUser(t *testing.T, s store.Store) string {
	t.Helper()
	email := fmt.Sprintf("%s@example.com", sanitizeEmail(t.Name()))
	u, err := s.GetUserByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	if u == nil {
		created, err := s.CreateUser(email, "$2a$10$dummyhashfortestonlyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		if err != nil {
			t.Fatal(err)
		}
		return created.ID
	}
	return u.ID
}

func sanitizeEmail(s string) string {
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, 'x')
		}
	}
	return string(out)
}

var testBundle = store.ContentBundle{
	Lesson: store.BundleLesson{ID: "t1", Title: "Variables", Slug: "variables", Summary: "Learn about variables", SortOrder: 1},
	Sections: []store.BundleSection{
		{ID: "t1:s1", Heading: "Intro", BodyHTML: "<p>test</p>", SortOrder: 1},
	},
	Questions: []store.BundleQuestion{
		{
			ID: "t1:q1", Prompt: "What is a variable?", CorrectKey: "a", QuestionType: "choice", SortOrder: 1,
			Options: []store.BundleOption{
				{Key: "a", Label: "A named memory location", IsCorrect: true, SortOrder: 1},
				{Key: "b", Label: "A type of loop", IsCorrect: false, SortOrder: 2},
			},
		},
		{
			ID: "t1:q2", Prompt: "Type the keyword to declare a variable", CorrectKey: "var", QuestionType: "text", SortOrder: 2,
		},
	},
	Exercises: []store.BundleExercise{
		{ID: "t1:ex1", Title: "Declare a variable", Path: "practice/variables", Instructions: "Run the exercise", SortOrder: 1},
	},
	References: []store.BundleReference{
		{ID: "t1:ref1", Title: "Go Tour", URL: "https://go.dev/tour", Notes: "official"},
	},
}

func TestListLessons(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	lessons, err := s.ListLessons()
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 1 {
		t.Fatalf("want 1 lesson, got %d", len(lessons))
	}
	if lessons[0].ID != "t1" {
		t.Fatalf("want lesson ID t1, got %q", lessons[0].ID)
	}
}

func TestGetLesson_Found(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	lesson, err := s.GetLesson("t1")
	if err != nil {
		t.Fatal(err)
	}
	if lesson == nil {
		t.Fatal("expected lesson, got nil")
	}
	if lesson.Title != "Variables" {
		t.Fatalf("want 'Variables', got %q", lesson.Title)
	}
}

func TestGetLesson_NotFound(t *testing.T) {
	s := testStore(t)
	lesson, err := s.GetLesson("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if lesson != nil {
		t.Fatal("expected nil for nonexistent lesson")
	}
}

func TestGetQuestion_Found(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	q, err := s.GetQuestion("t1:q1")
	if err != nil {
		t.Fatal(err)
	}
	if q == nil {
		t.Fatal("expected question, got nil")
	}
	if len(q.Options) != 2 {
		t.Fatalf("want 2 options, got %d", len(q.Options))
	}
}

func TestGetQuestion_NotFound(t *testing.T) {
	s := testStore(t)
	q, err := s.GetQuestion("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if q != nil {
		t.Fatal("expected nil for nonexistent question")
	}
}

func TestListQuestionsByLesson(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	questions, err := s.ListQuestionsByLesson(testUser(t, s), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 {
		t.Fatalf("want 2 questions, got %d", len(questions))
	}
	if questions[0].ID != "t1:q1" {
		t.Fatalf("want first question ID t1:q1, got %q", questions[0].ID)
	}
	if questions[1].ID != "t1:q2" {
		t.Fatalf("want second question ID t1:q2, got %q", questions[1].ID)
	}
}

func TestGetAnswer_Found(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAnswer(testUser(t, s), "t1:q1", "a", "A named memory location"); err != nil {
		t.Fatal(err)
	}
	ans, err := s.GetAnswer(testUser(t, s), "t1:q1")
	if err != nil {
		t.Fatal(err)
	}
	if ans == nil {
		t.Fatal("expected answer, got nil")
	}
	if !ans.Correct {
		t.Fatal("expected correct answer")
	}
}

func TestGetAnswer_NotFound(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	ans, err := s.GetAnswer(testUser(t, s), "t1:q1")
	if err != nil {
		t.Fatal(err)
	}
	if ans != nil {
		t.Fatal("expected nil for unanswered question")
	}
}

func TestListAnswers(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAnswer(testUser(t, s), "t1:q1", "a", "A named memory location"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAnswer(testUser(t, s), "t1:q2", "var", "var"); err != nil {
		t.Fatal(err)
	}
	answers, err := s.ListAnswers(testUser(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 2 {
		t.Fatalf("want 2 answers, got %d", len(answers))
	}
	if answers[0].QuestionID != "t1:q1" {
		t.Fatalf("want first question t1:q1, got %q", answers[0].QuestionID)
	}
	if !answers[0].Correct {
		t.Fatal("expected first answer correct")
	}
	if !answers[1].Correct {
		t.Fatal("expected second answer correct")
	}
	// Verify QuestionType is populated (needed for safeHTML rendering in progress.html).
	if answers[0].QuestionType != "choice" {
		t.Fatalf("want QuestionType 'choice', got %q", answers[0].QuestionType)
	}
	if answers[1].QuestionType != "text" {
		t.Fatalf("want QuestionType 'text', got %q", answers[1].QuestionType)
	}
}

func TestLessonProgress(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	progress, err := s.LessonProgress(testUser(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 1 {
		t.Fatalf("want 1 lesson, got %d", len(progress))
	}
	lp := progress[0]
	if lp.QuestionsTotal != 2 {
		t.Fatalf("want 2 total questions, got %d", lp.QuestionsTotal)
	}
	if lp.QuestionsAnswered != 0 {
		t.Fatalf("want 0 answered, got %d", lp.QuestionsAnswered)
	}
	if lp.ExerciseTotal != 1 {
		t.Fatalf("want 1 total exercise, got %d", lp.ExerciseTotal)
	}
	if _, err := s.SaveAnswer(testUser(t, s), "t1:q1", "a", "A named memory location"); err != nil {
		t.Fatal(err)
	}
	progress, err = s.LessonProgress(testUser(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if progress[0].QuestionsAnswered != 1 {
		t.Fatalf("want 1 answered after saving answer, got %d", progress[0].QuestionsAnswered)
	}
	if progress[0].QuestionsCorrect != 1 {
		t.Fatalf("want 1 correct, got %d", progress[0].QuestionsCorrect)
	}
}

func TestUserData(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	uid := testUser(t, s)

	// Anonymous: empty data, no error.
	ud, err := s.UserData("")
	if err != nil {
		t.Fatal(err)
	}
	if ud.TotalAnswered != 0 || ud.TotalCorrect != 0 || ud.TotalSubmitted != 0 {
		t.Fatalf("want zeros for anonymous, got answered=%d correct=%d submitted=%d",
			ud.TotalAnswered, ud.TotalCorrect, ud.TotalSubmitted)
	}

	// Authenticated with no answers: still zeros.
	ud, err = s.UserData(uid)
	if err != nil {
		t.Fatal(err)
	}
	if ud.TotalAnswered != 0 {
		t.Fatalf("want 0 answered, got %d", ud.TotalAnswered)
	}

	// Answer one question correctly.
	if _, err := s.SaveAnswer(uid, "t1:q1", "a", "A named memory location"); err != nil {
		t.Fatal(err)
	}
	ud, err = s.UserData(uid)
	if err != nil {
		t.Fatal(err)
	}
	if ud.TotalAnswered != 1 {
		t.Fatalf("want 1 answered, got %d", ud.TotalAnswered)
	}
	if ud.TotalCorrect != 1 {
		t.Fatalf("want 1 correct, got %d", ud.TotalCorrect)
	}
	if ac := ud.AnswersByLesson["t1"]; ac[0] != 1 || ac[1] != 1 {
		t.Fatalf("want [1,1] for lesson, got %v", ac)
	}

	// Submit an exercise.
	if err := s.SaveExerciseSubmission(uid, "t1:ex1", "output", true); err != nil {
		t.Fatal(err)
	}
	ud, err = s.UserData(uid)
	if err != nil {
		t.Fatal(err)
	}
	if ud.TotalSubmitted != 1 {
		t.Fatalf("want 1 submitted, got %d", ud.TotalSubmitted)
	}
	if !ud.SubmissionsByEx["t1:ex1"] {
		t.Fatal("want exercise marked submitted")
	}
	if !ud.CorrectByEx["t1:ex1"] {
		t.Fatal("want exercise marked correct")
	}
	if ud.SubmissionsByLesson["t1"] != 1 {
		t.Fatalf("want 1 submission for lesson, got %d", ud.SubmissionsByLesson["t1"])
	}
}

func TestLessonCounts(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	qCounts, eCounts, err := s.LessonCounts()
	if err != nil {
		t.Fatal(err)
	}
	if qCounts["t1"] != 2 {
		t.Fatalf("want 2 questions, got %d", qCounts["t1"])
	}
	if eCounts["t1"] != 1 {
		t.Fatalf("want 1 exercise, got %d", eCounts["t1"])
	}
}

func TestImportBundle_Idempotent(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatalf("second import should succeed: %v", err)
	}
	lessons, err := s.ListLessons()
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 1 {
		t.Fatalf("want 1 lesson after duplicate import, got %d", len(lessons))
	}
}

func TestMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "go-learn.db")
	s1, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()
	s2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("second open should succeed: %v", err)
	}
	s2.Close()
}

func TestSaveExerciseSubmission(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveExerciseSubmission(testUser(t, s), "t1:ex1", "Hello, world!", true); err != nil {
		t.Fatal(err)
	}
	output, ok, err := s.GetExerciseSubmission(testUser(t, s), "t1:ex1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected submission to exist")
	}
	if output != "Hello, world!" {
		t.Fatalf("want 'Hello, world!', got %q", output)
	}
}

func TestGetExerciseSubmission_NotFound(t *testing.T) {
	s := testStore(t)
	_, ok, err := s.GetExerciseSubmission(testUser(t, s), "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for nonexistent submission")
	}
}

func TestListExercisesByLesson(t *testing.T) {
	s := testStore(t)
	if err := s.ImportBundle(testBundle); err != nil {
		t.Fatal(err)
	}
	exercises, err := s.ListExercisesByLesson(testUser(t, s), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(exercises) != 1 {
		t.Fatalf("want 1 exercise, got %d", len(exercises))
	}
	if exercises[0].ID != "t1:ex1" {
		t.Fatalf("want exercise ID t1:ex1, got %q", exercises[0].ID)
	}
	if exercises[0].Submitted {
		t.Fatal("expected exercise not yet submitted")
	}
}

func TestImportMission_GetMission(t *testing.T) {
	s := testStore(t)
	b := store.MissionBundle{
		Why:             "Learn Go",
		SuccessCriteria: []string{"Write a program", "Understand types"},
		Constraints:     []string{"Beginner", "Self-paced"},
	}
	if err := s.ImportMission(b); err != nil {
		t.Fatal(err)
	}
	m, err := s.GetMission()
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected mission, got nil")
	}
	if m.Why != "Learn Go" {
		t.Fatalf("want 'Learn Go', got %q", m.Why)
	}
}

func TestImportGlossary_ListGlossaryTerms(t *testing.T) {
	s := testStore(t)
	b := store.GlossaryBundle{
		Terms: []store.BundleGlossaryTerm{
			{ID: "g1", Term: "Variable", Definition: "A named value", SortOrder: 1},
			{ID: "g2", Term: "Function", Definition: "A block of code", SortOrder: 2},
		},
	}
	if err := s.ImportGlossary(b); err != nil {
		t.Fatal(err)
	}
	terms, err := s.ListGlossaryTerms()
	if err != nil {
		t.Fatal(err)
	}
	if len(terms) != 2 {
		t.Fatalf("want 2 terms, got %d", len(terms))
	}
	if terms[0].Term != "Variable" {
		t.Fatalf("want 'Variable', got %q", terms[0].Term)
	}
}

func TestImportInsights_ListActiveInsights(t *testing.T) {
	s := testStore(t)
	b := store.InsightsBundle{
		Insights: []store.BundleInsight{
			{ID: "i1", Title: "Weak on types", Body: "Review type system", Kind: "weak_spot"},
			{ID: "i2", Title: "Great pace", Body: "Keep going", Kind: "milestone"},
		},
	}
	if err := s.ImportInsights(b); err != nil {
		t.Fatal(err)
	}
	insights, err := s.ListActiveInsights("")
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) != 2 {
		t.Fatalf("want 2 insights, got %d", len(insights))
	}
	weak, err := s.ListActiveInsights("weak_spot")
	if err != nil {
		t.Fatal(err)
	}
	if len(weak) != 1 {
		t.Fatalf("want 1 weak_spot, got %d", len(weak))
	}
}
