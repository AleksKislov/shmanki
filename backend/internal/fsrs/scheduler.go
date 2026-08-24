package fsrs

import (
	"math"
	"math/rand"
	"time"
)

const (
	// FuzzMinIntervalDays is the shortest review-mode interval that gets randomized.
	// Shorter intervals are left exact so short-term scheduling stays predictable.
	FuzzMinIntervalDays = 3.0
	// FuzzFactor is the maximum fractional deviation (+/-) applied to a review interval,
	// spreading out cards that would otherwise all come due on the same day.
	FuzzFactor = 0.05
)

type CardState struct {
	Stability           float64
	Difficulty          float64
	EffectiveDifficulty float64
	HierarchicalSupport float64
	Retrievability      float64
	DueDate             *time.Time
	LastReview          *time.Time
	IntervalDays        float64
	LearningStep        int
	Status              CardStatus
	Reps                int
	Lapses              int
}

type Scheduler struct {
	weights [19]float64
	config  Config
	// fuzz returns a value in [0, 1) used to randomize review intervals.
	// Overridden in tests for deterministic assertions.
	fuzz func() float64
}

func NewScheduler(weights [19]float64, cfg Config) *Scheduler {
	return &Scheduler{
		weights: weights,
		config:  cfg.withDefaults(),
		fuzz:    rand.Float64,
	}
}

// finalizeInterval randomizes an interval by up to +/-FuzzFactor (to avoid cards
// scheduled together from clumping on the same due date) and caps it at
// MaximumIntervalDays, before re-clamping to the minimum interval.
func (s *Scheduler) finalizeInterval(days float64) float64 {
	if days >= FuzzMinIntervalDays {
		spread := days * FuzzFactor
		days += (s.fuzz()*2 - 1) * spread
	}
	if days > s.config.MaximumIntervalDays {
		days = s.config.MaximumIntervalDays
	}
	return math.Max(math.Round(days), MinIntervalDays)
}

func (s *Scheduler) Schedule(state CardState, rating Rating, now time.Time, hierarchicalSupport float64) CardState {
	state.HierarchicalSupport = clamp(hierarchicalSupport, 0, 1)
	state.LearningStep = normalizeStep(state.LearningStep)

	if isInitialState(state) {
		return s.scheduleInitial(state, rating, now)
	}

	switch state.Status {
	case StatusLearning:
		return s.scheduleStepMode(state, rating, now, s.config.LearningSteps, false)
	case StatusRelearning:
		return s.scheduleStepMode(state, rating, now, s.config.RelearningSteps, true)
	default:
		return s.scheduleReview(state, rating, now)
	}
}

func isInitialState(state CardState) bool {
	return state.Status == StatusNew || state.Status == StatusLocked || state.LastReview == nil
}

func (s *Scheduler) scheduleInitial(state CardState, rating Rating, now time.Time) CardState {
	state.LastReview = ptrTime(now)
	state.Retrievability = 1

	if rating == RatingEasy {
		state.Reps++
		return s.graduateFromLearning(state, rating, now)
	}

	nextStep := 0
	if rating == RatingGood {
		state.Reps++
		nextStep = 1
	}
	if rating == RatingHard {
		state.Reps++
	}

	if nextStep >= len(s.config.LearningSteps) {
		return s.graduateFromLearning(state, rating, now)
	}

	return setStepSchedule(state, StatusLearning, nextStep, now, s.config.LearningSteps[nextStep])
}

func (s *Scheduler) scheduleStepMode(state CardState, rating Rating, now time.Time, steps []time.Duration, fromRelearning bool) CardState {
	state.LastReview = ptrTime(now)
	state.Retrievability = 1

	nextStep, graduated := applyStepTransition(state.LearningStep, rating, len(steps))
	if rating != RatingAgain {
		state.Reps++
	}

	if graduated {
		if fromRelearning {
			return s.graduateFromRelearning(state, rating, now)
		}
		return s.graduateFromLearning(state, rating, now)
	}

	status := StatusLearning
	if fromRelearning {
		status = StatusRelearning
	}

	return setStepSchedule(state, status, nextStep, now, steps[nextStep])
}

func (s *Scheduler) scheduleReview(state CardState, rating Rating, now time.Time) CardState {
	lastReview := now
	if state.LastReview != nil {
		lastReview = *state.LastReview
	}

	daysSinceReview := now.Sub(lastReview).Hours() / 24
	retrievability := Retrievability(daysSinceReview, state.Stability)

	if rating == RatingAgain {
		state.Stability = StabilityAfterForgetting(state.Stability, state.Difficulty, retrievability, s.weights)
		state.Difficulty = UpdateDifficulty(state.Difficulty, rating, s.weights)
		state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
		state.Retrievability = Retrievability(0, state.Stability)
		state.Lapses++
		state.LastReview = ptrTime(now)

		return setStepSchedule(state, StatusRelearning, 0, now, s.config.RelearningSteps[0])
	}

	nextStability := StabilityAfterRecall(state.Stability, state.Difficulty, retrievability, rating, s.weights)
	state.Stability = nextStability
	state.Difficulty = UpdateDifficulty(state.Difficulty, rating, s.weights)
	state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
	state.Retrievability = Retrievability(0, nextStability)
	state.IntervalDays = s.finalizeInterval(NextInterval(nextStability, state.EffectiveDifficulty, s.config.DesiredRetention))
	state.DueDate = ptrTime(now.Add(daysToDuration(state.IntervalDays)))
	state.Status = StatusReview
	state.LearningStep = 0
	state.Reps++
	state.LastReview = ptrTime(now)

	return state
}

func (s *Scheduler) graduateFromLearning(state CardState, rating Rating, now time.Time) CardState {
	state.Stability = InitialStability(rating, s.weights)
	state.Difficulty = InitialDifficulty(rating, s.weights)
	state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
	state.Retrievability = Retrievability(0, state.Stability)
	state.IntervalDays = s.finalizeInterval(NextInterval(state.Stability, state.EffectiveDifficulty, s.config.DesiredRetention))
	state.DueDate = ptrTime(now.Add(daysToDuration(state.IntervalDays)))
	state.Status = StatusReview
	state.LearningStep = 0
	return state
}

func (s *Scheduler) graduateFromRelearning(state CardState, rating Rating, now time.Time) CardState {
	state.Difficulty = UpdateDifficulty(state.Difficulty, rating, s.weights)
	state.EffectiveDifficulty = EffectiveDifficulty(state.Difficulty, state.HierarchicalSupport, s.config.HierarchicalDifficultyPenalty)
	state.Retrievability = Retrievability(0, state.Stability)
	state.IntervalDays = s.finalizeInterval(NextInterval(state.Stability, state.EffectiveDifficulty, s.config.DesiredRetention))
	state.DueDate = ptrTime(now.Add(daysToDuration(state.IntervalDays)))
	state.Status = StatusReview
	state.LearningStep = 0
	return state
}

func applyStepTransition(currentStep int, rating Rating, stepCount int) (int, bool) {
	if stepCount <= 0 {
		return 0, true
	}

	step := normalizeStep(currentStep)
	if step >= stepCount {
		step = stepCount - 1
	}

	switch rating {
	case RatingAgain:
		return 0, false
	case RatingHard:
		return step, false
	case RatingEasy:
		return 0, true
	default:
		next := step + 1
		if next >= stepCount {
			return 0, true
		}
		return next, false
	}
}

func setStepSchedule(state CardState, status CardStatus, step int, now time.Time, interval time.Duration) CardState {
	state.Status = status
	state.LearningStep = normalizeStep(step)
	state.IntervalDays = interval.Hours() / 24
	state.DueDate = ptrTime(now.Add(interval))
	return state
}

func normalizeStep(step int) int {
	if step < 0 {
		return 0
	}
	return step
}

func daysToDuration(days float64) time.Duration {
	return time.Duration(days * 24 * float64(time.Hour))
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
	return s.config.ReviewStabilityThresholdDays
}

func (s *Scheduler) ReviewStabilityThresholdDays() float64 {
	return s.config.ReviewStabilityThresholdDays
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
