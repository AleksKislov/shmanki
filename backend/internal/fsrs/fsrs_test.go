package fsrs

import (
	"math"
	"testing"
	"time"
)

func TestInitialDifficultyRange(t *testing.T) {
	difficulty := InitialDifficulty(RatingEasy, DefaultWeights)
	if difficulty < 1 || difficulty > 10 {
		t.Fatalf("expected clamped difficulty, got %f", difficulty)
	}
}

func TestNextIntervalMinimum(t *testing.T) {
	interval := NextInterval(0.1, 10, 0.9)
	if interval < 1 {
		t.Fatalf("expected minimum interval >= 1, got %f", interval)
	}
}

func TestScheduleInitialReview(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingGood, now, 1)

	if state.Status != StatusLearning {
		t.Fatalf("expected learning status, got %s", state.Status)
	}
	if state.LearningStep != 1 {
		t.Fatalf("expected learning step 1, got %d", state.LearningStep)
	}
	if state.LastReview == nil || state.DueDate == nil {
		t.Fatal("expected timestamps to be set")
	}
	if state.Reps != 1 {
		t.Fatalf("expected reps to be incremented, got %d", state.Reps)
	}
	if state.Stability != 0 {
		t.Fatalf("expected initial learning step to avoid FSRS stability init, got %f", state.Stability)
	}
}

func TestHierarchicalSupportAverageMastery(t *testing.T) {
	support := HierarchicalSupport([]float64{DefaultConfig.ReviewStabilityThresholdDays, DefaultConfig.ReviewStabilityThresholdDays / 2, 0}, DefaultConfig.ReviewStabilityThresholdDays)
	if support != 0.5 {
		t.Fatalf("expected support 0.5, got %f", support)
	}
}

func TestScheduleReducesIntervalWhenSupportWeak(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()
	lastReview := now.Add(-24 * time.Hour)
	base := CardState{
		Stability:  15,
		Difficulty: 5,
		LastReview: &lastReview,
		Status:     StatusReview,
	}

	strong := scheduler.Schedule(base, RatingGood, now, 1)
	weak := scheduler.Schedule(base, RatingGood, now, 0)

	if weak.EffectiveDifficulty <= strong.EffectiveDifficulty {
		t.Fatalf("expected weaker support to increase effective difficulty: weak=%f strong=%f", weak.EffectiveDifficulty, strong.EffectiveDifficulty)
	}
	if weak.IntervalDays >= strong.IntervalDays {
		t.Fatalf("expected weaker support to shorten interval: weak=%f strong=%f", weak.IntervalDays, strong.IntervalDays)
	}
}

func TestScheduleInitialAgainEntersLearningStepZero(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingAgain, now, 1)

	if state.Status != StatusLearning {
		t.Fatalf("expected learning status, got %s", state.Status)
	}
	if state.LearningStep != 0 {
		t.Fatalf("expected learning step 0, got %d", state.LearningStep)
	}
	if state.Reps != 0 {
		t.Fatalf("expected reps to remain 0, got %d", state.Reps)
	}
	assertDueWithinDuration(t, state.DueDate, now, time.Minute)
}

func TestScheduleInitialHardStartsAtLearningStepZero(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingHard, now, 1)

	if state.Status != StatusLearning {
		t.Fatalf("expected learning status, got %s", state.Status)
	}
	if state.LearningStep != 0 {
		t.Fatalf("expected learning step 0, got %d", state.LearningStep)
	}
	if state.Reps != 1 {
		t.Fatalf("expected reps 1, got %d", state.Reps)
	}
	assertDueWithinDuration(t, state.DueDate, now, time.Minute)
}

func TestScheduleInitialEasyGraduatesImmediately(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingEasy, now, 1)

	if state.Status != StatusReview {
		t.Fatalf("expected review status, got %s", state.Status)
	}
	if state.LearningStep != 0 {
		t.Fatalf("expected learning step reset to 0, got %d", state.LearningStep)
	}
	if state.Stability <= 0 {
		t.Fatalf("expected initialized stability, got %f", state.Stability)
	}
	if state.IntervalDays < 1 {
		t.Fatalf("expected FSRS interval >= 1 day, got %f", state.IntervalDays)
	}
}

func TestLearningGoodGraduatesAfterFinalStep(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := CardState{
		Status:       StatusLearning,
		LearningStep: 2,
		LastReview:   ptrTime(now.Add(-time.Hour)),
	}

	next := scheduler.Schedule(state, RatingGood, now, 1)

	if next.Status != StatusReview {
		t.Fatalf("expected graduation to review, got %s", next.Status)
	}
	if next.Stability <= 0 {
		t.Fatalf("expected initialized stability on graduation, got %f", next.Stability)
	}
	if next.LearningStep != 0 {
		t.Fatalf("expected learning step reset to 0, got %d", next.LearningStep)
	}
}

func TestLearningAgainResetsToStepZero(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := CardState{
		Status:       StatusLearning,
		LearningStep: 2,
		LastReview:   ptrTime(now.Add(-time.Hour)),
	}

	next := scheduler.Schedule(state, RatingAgain, now, 1)

	if next.Status != StatusLearning {
		t.Fatalf("expected to stay in learning, got %s", next.Status)
	}
	if next.LearningStep != 0 {
		t.Fatalf("expected reset to step 0, got %d", next.LearningStep)
	}
	assertDueWithinDuration(t, next.DueDate, now, time.Minute)
}

func TestReviewAgainEntersRelearning(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()
	lastReview := now.Add(-48 * time.Hour)

	state := CardState{
		Status:       StatusReview,
		Stability:    20,
		Difficulty:   5,
		LastReview:   &lastReview,
		IntervalDays: 8,
		Reps:         5,
	}

	next := scheduler.Schedule(state, RatingAgain, now, 1)

	if next.Status != StatusRelearning {
		t.Fatalf("expected relearning status, got %s", next.Status)
	}
	if next.LearningStep != 0 {
		t.Fatalf("expected relearning step 0, got %d", next.LearningStep)
	}
	if next.Lapses != 1 {
		t.Fatalf("expected lapses increment, got %d", next.Lapses)
	}
	assertDueWithinDuration(t, next.DueDate, now, 10*time.Minute)
}

func TestRelearningGoodGraduatesWithDecayedStability(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()
	lastReview := now.Add(-48 * time.Hour)

	state := CardState{
		Status:       StatusReview,
		Stability:    30,
		Difficulty:   5,
		LastReview:   &lastReview,
		IntervalDays: 12,
		Reps:         7,
	}

	relearn := scheduler.Schedule(state, RatingAgain, now, 1)
	decayedStability := relearn.Stability
	if decayedStability <= 0 {
		t.Fatalf("expected decayed stability > 0, got %f", decayedStability)
	}

	step1 := scheduler.Schedule(relearn, RatingGood, now.Add(10*time.Minute), 1)
	if step1.Status != StatusRelearning {
		t.Fatalf("expected to stay in relearning at step1, got %s", step1.Status)
	}
	if step1.LearningStep != 1 {
		t.Fatalf("expected relearning step 1, got %d", step1.LearningStep)
	}

	graduated := scheduler.Schedule(step1, RatingGood, now.Add(70*time.Minute), 1)
	if graduated.Status != StatusReview {
		t.Fatalf("expected graduation to review, got %s", graduated.Status)
	}
	if math.Abs(graduated.Stability-decayedStability) > 1e-9 {
		t.Fatalf("expected relearning graduation to keep decayed stability: got %f want %f", graduated.Stability, decayedStability)
	}
	if math.Abs(graduated.Stability-InitialStability(RatingGood, DefaultWeights)) < 1e-9 {
		t.Fatalf("expected stability not to reset to initial good stability")
	}
}

func TestRelearningAgainResetsToStepZero(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := CardState{
		Status:       StatusRelearning,
		Stability:    5,
		Difficulty:   5,
		LearningStep: 1,
		LastReview:   ptrTime(now.Add(-time.Hour)),
	}

	next := scheduler.Schedule(state, RatingAgain, now, 1)
	if next.Status != StatusRelearning {
		t.Fatalf("expected to stay in relearning, got %s", next.Status)
	}
	if next.LearningStep != 0 {
		t.Fatalf("expected relearning step reset to 0, got %d", next.LearningStep)
	}
	assertDueWithinDuration(t, next.DueDate, now, 10*time.Minute)
}

func TestDefaultConfigHasStepIntervals(t *testing.T) {
	if len(DefaultConfig.LearningSteps) != 3 {
		t.Fatalf("expected 3 learning steps, got %d", len(DefaultConfig.LearningSteps))
	}
	if len(DefaultConfig.RelearningSteps) != 2 {
		t.Fatalf("expected 2 relearning steps, got %d", len(DefaultConfig.RelearningSteps))
	}
	if DefaultConfig.LearningSteps[0] != time.Minute {
		t.Fatalf("expected first learning step 1 minute, got %s", DefaultConfig.LearningSteps[0])
	}
	if DefaultConfig.RelearningSteps[0] != 10*time.Minute {
		t.Fatalf("expected first relearning step 10 minutes, got %s", DefaultConfig.RelearningSteps[0])
	}
}

func assertDueWithinDuration(t *testing.T, dueDate *time.Time, now time.Time, expected time.Duration) {
	t.Helper()
	if dueDate == nil {
		t.Fatal("expected due date to be set")
	}

	actual := dueDate.Sub(now)
	if actual < expected-2*time.Second || actual > expected+2*time.Second {
		t.Fatalf("expected due date offset %s, got %s", expected, actual)
	}
}
