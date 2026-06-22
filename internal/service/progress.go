package service

import (
	"sync"

	"github.com/kartikkabadi/go-learn/internal/store"
)

// Progress aggregates store data into dashboard views for the web handlers.
type Progress struct {
	Store store.Store
}

// Dashboard bundles all data needed to render the home page.
type Dashboard struct {
	Stats      store.DashboardStats
	Mission    *store.Mission
	Lessons    []store.LessonProgress
	WeakSpots  []store.Insight
	Exercises  []store.Exercise
	NextLesson *store.LessonProgress
}

// Dashboard assembles stats, mission, lessons, weak spots, and the next uncompleted lesson.
// All 5 store queries run in parallel to minimize D1 round-trip latency.
func (p *Progress) Dashboard(userID string) (Dashboard, error) {
	var (
		stats     store.DashboardStats
		mission   *store.Mission
		lessons   []store.LessonProgress
		weak      []store.Insight
		exercises []store.Exercise
	)
	var sErr, mErr, lErr, wErr, eErr error
	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); stats, sErr = p.Store.DashboardStats(userID) }()
	go func() { defer wg.Done(); mission, mErr = p.Store.GetMission() }()
	go func() { defer wg.Done(); lessons, lErr = p.Store.LessonProgress(userID) }()
	go func() { defer wg.Done(); weak, wErr = p.Store.ListActiveInsights("weak_spot") }()
	go func() { defer wg.Done(); exercises, eErr = p.Store.ListExercises(userID) }()
	wg.Wait()

	if sErr != nil {
		return Dashboard{}, sErr
	}
	if mErr != nil {
		return Dashboard{}, mErr
	}
	if lErr != nil {
		return Dashboard{}, lErr
	}
	if wErr != nil {
		return Dashboard{}, wErr
	}
	if eErr != nil {
		return Dashboard{}, eErr
	}

	var next *store.LessonProgress
	for i := range lessons {
		lp := &lessons[i]
		if lp.QuestionsAnswered < lp.QuestionsTotal || lp.ExercisesDone < lp.ExerciseTotal {
			next = lp
			break
		}
	}

	return Dashboard{
		Stats:      stats,
		Mission:    mission,
		Lessons:    lessons,
		WeakSpots:  weak,
		Exercises:  exercises,
		NextLesson: next,
	}, nil
}
