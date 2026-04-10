package fsrs

import "time"

type CardState struct {
	Stability           float64
	Difficulty          float64
	EffectiveDifficulty float64
	HierarchicalSupport float64
	Retrievability      float64
	DueDate             *time.Time
	LastReview          *time.Time
	IntervalDays        float64
	Status              CardStatus
	Reps                int
	Lapses              int
}

type Scheduler struct {
	weights [19]float64
	config  Config
}

func NewScheduler(weights [19]float64, cfg Config) *Scheduler {
	return &Scheduler{
		weights: weights,
		config:  cfg.withDefaults(),
	}
}

func (s *Scheduler) Schedule(state CardState, rating Rating, now time.Time, hierarchicalSupport float64) CardState {
	state.HierarchicalSupport = clamp(hierarchicalSupport, 0, 1)
	if state.Stability <= 0 || state.LastReview == nil || state.Status == StatusNew || state.Status == StatusLocked {
		state.Stability = InitialStability(rating, s.weights)
		state.Difficulty = InitialDifficulty(rating, s.weights)
		state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
		state.Retrievability = 1
		state.IntervalDays = NextInterval(state.Stability, state.EffectiveDifficulty, s.config.DesiredRetention)
		state.LastReview = ptrTime(now)
		dueDate := now.Add(time.Duration(state.IntervalDays*24) * time.Hour)
		state.DueDate = &dueDate
		if rating == RatingAgain {
			state.Status = StatusRelearning
		} else if state.Stability >= s.config.ReviewStabilityThresholdDays {
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
		if nextStability >= s.config.ReviewStabilityThresholdDays {
			state.Status = StatusReview
		} else {
			state.Status = StatusLearning
		}
	}

	state.Stability = nextStability
	state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
	state.Retrievability = Retrievability(0, nextStability)
	state.IntervalDays = NextInterval(nextStability, state.EffectiveDifficulty, s.config.DesiredRetention)
	state.LastReview = ptrTime(now)
	dueDate := now.Add(time.Duration(state.IntervalDays*24) * time.Hour)
	state.DueDate = &dueDate

	return state
}

func (s *Scheduler) ShouldUnlockStep(stabilities []float64) bool {
	for _, stability := range stabilities {
		if stability < s.config.StepUnlockStabilityDays {
			return false
		}
	}

	return true
}

func (s *Scheduler) StepUnlockDays() float64 {
	return s.config.StepUnlockStabilityDays
}

func (s *Scheduler) HierarchicalPenalty() float64 {
	return s.config.HierarchicalDifficultyPenalty
}

func (s *Scheduler) SupportReferenceDays() float64 {
	return s.config.SupportReferenceStabilityDays
}

func (s *Scheduler) ReviewStabilityThresholdDays() float64 {
	return s.config.ReviewStabilityThresholdDays
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
