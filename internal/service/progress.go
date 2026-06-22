package service

import "example.com/go-learn/internal/store"

// Progress aggregates store data into dashboard views for the web handlers.
type Progress struct {
	Store store.Store
}

// Dashboard bundles all data needed to render the home page.
type Dashboard struct {
	Stats       store.DashboardStats
	Mission     *store.Mission
	Lessons     []store.LessonProgress
	WeakSpots   []store.Insight
	Exercises   []store.Exercise
	NextLesson  *store.LessonProgress
}

// Dashboard assembles stats, mission, lessons, weak spots, and the next uncompleted lesson.
func (p *Progress) Dashboard() (Dashboard, error) {
	stats, err := p.Store.DashboardStats()
	if err != nil {
		return Dashboard{}, err
	}
	mission, err := p.Store.GetMission()
	if err != nil {
		return Dashboard{}, err
	}
	lessons, err := p.Store.LessonProgress()
	if err != nil {
		return Dashboard{}, err
	}
	weak, err := p.Store.ListActiveInsights("weak_spot")
	if err != nil {
		return Dashboard{}, err
	}
	exercises, err := p.Store.ListExercises()
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

// AccuracyPercent returns overall quiz accuracy as a percentage (0-100).
func (p *Progress) AccuracyPercent() (int, error) {
	stats, err := p.Store.DashboardStats()
	if err != nil {
		return 0, err
	}
	if stats.QuestionsAnswered == 0 {
		return 0, nil
	}
	return (stats.QuestionsCorrect * 100) / stats.QuestionsAnswered, nil
}

// LessonCompletionPercent returns a lesson completion percentage including both quizzes and exercises.
func LessonCompletionPercent(lp store.LessonProgress) int {
	total := lp.QuestionsTotal + lp.ExerciseTotal
	if total == 0 {
		return 0
	}
	done := lp.QuestionsAnswered + lp.ExercisesDone
	return (done * 100) / total
}
