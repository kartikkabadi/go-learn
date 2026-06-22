package service_test

import (
	"path/filepath"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
)

func TestAccuracyPercent_NoAnswers(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p := &service.Progress{Store: s}
	pct, err := p.AccuracyPercent()
	if err != nil {
		t.Fatal(err)
	}
	if pct != 0 {
		t.Fatalf("want 0%% accuracy with no answers, got %d%%", pct)
	}
}

func TestDashboard_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p := &service.Progress{Store: s}
	dash, err := p.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dash.Stats.LessonsTotal != 0 {
		t.Fatalf("want 0 lessons, got %d", dash.Stats.LessonsTotal)
	}
	if dash.Mission != nil {
		t.Fatal("expected nil mission for empty store")
	}
}

func TestLessonCompletionPercent(t *testing.T) {
	tests := []struct {
		qTotal, qAns, eTotal, eDone int
		want                        int
	}{
		{0, 0, 0, 0, 0},
		{2, 1, 0, 0, 50},
		{2, 2, 1, 1, 100},
		{4, 1, 2, 0, 16},
	}

	for _, tt := range tests {
		lp := store.LessonProgress{
			QuestionsTotal:    tt.qTotal,
			QuestionsAnswered: tt.qAns,
			ExerciseTotal:     tt.eTotal,
			ExercisesDone:     tt.eDone,
		}
		got := service.LessonCompletionPercent(lp)
		if got != tt.want {
			t.Fatalf("LessonCompletionPercent(%+v) = %d, want %d", tt, got, tt.want)
		}
	}
}
