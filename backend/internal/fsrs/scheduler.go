package fsrs

import "time"

type CardState struct {
	Stability      float64
	Difficulty     float64
	Retrievability float64
	DueDate        *time.Time
	LastReview     *time.Time
	IntervalDays   float64
	Status         CardStatus
	Reps           int
	Lapses         int
}

type Scheduler struct {
	weights          [19]float64
	desiredRetention float64
	stepUnlockDays   float64
}

func NewScheduler(weights [19]float64, desiredRetention float64, stepUnlockDays float64) *Scheduler {
	if desiredRetention <= 0 || desiredRetention >= 1 {
		desiredRetention = 0.9
	}
	if stepUnlockDays <= 0 {
		stepUnlockDays = 14
	}

	return &Scheduler{
		weights:          weights,
		desiredRetention: desiredRetention,
		stepUnlockDays:   stepUnlockDays,
	}
}

func (s *Scheduler) Schedule(state CardState, rating Rating, now time.Time) CardState {
	if state.Stability <= 0 || state.LastReview == nil || state.Status == StatusNew || state.Status == StatusLocked {
		state.Stability = InitialStability(rating, s.weights)
		state.Difficulty = InitialDifficulty(rating, s.weights)
		state.Retrievability = 1
		state.IntervalDays = NextInterval(state.Stability, s.desiredRetention)
		state.LastReview = ptrTime(now)
		dueDate := now.Add(time.Duration(state.IntervalDays*24) * time.Hour)
		state.DueDate = &dueDate
		if rating == RatingAgain {
			state.Status = StatusRelearning
		} else if state.Stability >= 21 {
			state.Status = StatusReview
			state.Reps++
		} else {
			state.Status = StatusLearning
			state.Reps++
		}
		return state
	}

	daysSinceReview := now.Sub(*state.LastReview).Hours() / 24
	retrievability := Retrievability(daysSinceReview, state.Stability)

	var nextStability float64
	if rating == RatingAgain {
		nextStability = StabilityAfterForgetting(state.Stability, state.Difficulty, retrievability, s.weights)
		state.Difficulty = UpdateDifficulty(state.Difficulty, rating, s.weights)
		if state.Status == StatusReview {
			state.Lapses++
		}
		state.Status = StatusRelearning
	} else {
		nextStability = StabilityAfterRecall(state.Stability, state.Difficulty, retrievability, rating, s.weights)
		state.Difficulty = UpdateDifficulty(state.Difficulty, rating, s.weights)
		state.Reps++
		if nextStability >= 21 {
			state.Status = StatusReview
		} else {
			state.Status = StatusLearning
		}
	}

	state.Stability = nextStability
	state.Retrievability = Retrievability(0, nextStability)
	state.IntervalDays = NextInterval(nextStability, s.desiredRetention)
	state.LastReview = ptrTime(now)
	dueDate := now.Add(time.Duration(state.IntervalDays*24) * time.Hour)
	state.DueDate = &dueDate

	return state
}

func (s *Scheduler) ShouldUnlockStep(stabilities []float64) bool {
	for _, stability := range stabilities {
		if stability < s.stepUnlockDays {
			return false
		}
	}

	return true
}

func (s *Scheduler) StepUnlockDays() float64 {
	return s.stepUnlockDays
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
