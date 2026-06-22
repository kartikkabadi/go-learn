package service_test

import (
	"path/filepath"
	"testing"

	"github.com/kartikkabadi/go-learn/internal/service"
	"github.com/kartikkabadi/go-learn/internal/store"
)

func TestDashboard_EmptyStore(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "go-learn.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	p := &service.Progress{Store: s}
	dash, err := p.Dashboard("")
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
