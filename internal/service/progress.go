package service

import "github.com/kartikkabadi/go-learn/internal/store"

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
func (p *Progress) Dashboard(userID string) (Dashboard, error) {
	stats, err := p.Store.DashboardStats(userID)
	if err != nil {
		return Dashboard{}, err
	}
	mission, err := p.Store.GetMission()
	if err != nil {
		return Dashboard{}, err
	}
	lessons, err := p.Store.LessonProgress(userID)
	if err != nil {
		return Dashboard{}, err
	}
	weak, err := p.Store.ListActiveInsights("weak_spot")
	if err != nil {
		return Dashboard{}, err
	}
	exercises, err := p.Store.ListExercises(userID)
	if err != nil {
		return Dashboard{}, err
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
