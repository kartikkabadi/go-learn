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
// Uses UserData (2 D1 queries) for all user-specific data, plus store methods for
// static content (0 D1 queries in d1store — returns embedded content).
// Anonymous: 0 D1 queries. Authenticated: 2 D1 queries.
func (p *Progress) Dashboard(userID string) (Dashboard, error) {
	ud, err := p.Store.UserData(userID)
	if err != nil {
		return Dashboard{}, err
	}

	// Static content via store (0 D1 queries in d1store — embedded).
	lessons, err := p.Store.ListLessons()
	if err != nil {
		return Dashboard{}, err
	}
	mission, err := p.Store.GetMission()
	if err != nil {
		return Dashboard{}, err
	}
	weak, err := p.Store.ListActiveInsights("weak_spot")
	if err != nil {
		return Dashboard{}, err
	}
	qCounts, eCounts, err := p.Store.LessonCounts()
	if err != nil {
		return Dashboard{}, err
	}
	// ListExercises("") returns embedded content with no submission status (0 D1 queries).
	// Submission status is attached from UserData below.
	exercises, err := p.Store.ListExercises("")
	if err != nil {
		return Dashboard{}, err
	}

	// Compute DashboardStats from lesson counts + user data.
	stats := store.DashboardStats{
		LessonsTotal:       len(lessons),
		QuestionsAnswered:  ud.TotalAnswered,
		QuestionsCorrect:   ud.TotalCorrect,
		ExercisesSubmitted: ud.TotalSubmitted,
	}
	for _, n := range qCounts {
		stats.QuestionsTotal += n
	}
	for _, n := range eCounts {
		stats.ExercisesTotal += n
	}

	// Compute LessonProgress from lessons + counts + user data.
	progress := make([]store.LessonProgress, len(lessons))
	for i, l := range lessons {
		lp := store.LessonProgress{
			LessonID:       l.ID,
			Title:          l.Title,
			Slug:           l.Slug,
			SortOrder:      l.SortOrder,
			QuestionsTotal: qCounts[l.ID],
			ExerciseTotal:  eCounts[l.ID],
			ExercisesDone:  ud.SubmissionsByLesson[l.ID],
		}
		if ac, ok := ud.AnswersByLesson[l.ID]; ok {
			lp.QuestionsAnswered = ac[0]
			lp.QuestionsCorrect = ac[1]
		}
		progress[i] = lp
	}

	// Attach submission status to exercises from UserData (0 extra D1 queries).
	for i := range exercises {
		if ud.SubmissionsByEx[exercises[i].ID] {
			exercises[i].Submitted = true
			exercises[i].Correct = ud.CorrectByEx[exercises[i].ID]
		}
	}

	var next *store.LessonProgress
	for i := range progress {
		lp := &progress[i]
		if lp.QuestionsAnswered < lp.QuestionsTotal || lp.ExercisesDone < lp.ExerciseTotal {
			next = lp
			break
		}
	}

	return Dashboard{
		Stats:      stats,
		Mission:    mission,
		Lessons:    progress,
		WeakSpots:  weak,
		Exercises:  exercises,
		NextLesson: next,
	}, nil
}
