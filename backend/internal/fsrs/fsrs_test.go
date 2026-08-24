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
	scheduler := newDeterministicScheduler(DefaultWeights, DefaultConfig)
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

func newDeterministicScheduler(weights [19]float64, cfg Config) *Scheduler {
	s := NewScheduler(weights, cfg)
	s.fuzz = func() float64 { return 0.5 } // midpoint => no net perturbation
	return s
}

func TestStabilityAfterRecallUsesMainFormulaWeights(t *testing.T) {
	base := StabilityAfterRecall(10, 5, 0.8, RatingGood, DefaultWeights)

	changedCore := DefaultWeights
	changedCore[8] += 1.0
	if got := StabilityAfterRecall(10, 5, 0.8, RatingGood, changedCore); got == base {
		t.Fatalf("expected weights[8] to affect stability growth, got identical result %f", got)
	}

	changedUnused := DefaultWeights
	changedUnused[17] += 5.0
	changedUnused[18] += 5.0
	if got := StabilityAfterRecall(10, 5, 0.8, RatingGood, changedUnused); got != base {
		t.Fatalf("expected weights[17]/weights[18] to be unused by the main recall formula: base=%f changed=%f", base, got)
	}
}

func TestFinalizeIntervalAppliesFuzz(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)

	scheduler.fuzz = func() float64 { return 0 }
	low := scheduler.finalizeInterval(100)
	scheduler.fuzz = func() float64 { return 1 }
	high := scheduler.finalizeInterval(100)

	if !(low < 100 && high > 100) {
		t.Fatalf("expected fuzz to spread a 100-day interval below and above baseline, got low=%f high=%f", low, high)
	}
	if spread := high - low; spread > 2*100*FuzzFactor+1 {
		t.Fatalf("expected fuzz spread within +/-%.0f%%, got low=%f high=%f", FuzzFactor*100, low, high)
	}
}

func TestFinalizeIntervalSkipsFuzzBelowThreshold(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, DefaultConfig)
	scheduler.fuzz = func() float64 { return 1 }

	if result := scheduler.finalizeInterval(FuzzMinIntervalDays - 1); result != FuzzMinIntervalDays-1 {
		t.Fatalf("expected short intervals to skip fuzz, got %f", result)
	}
}

func TestFinalizeIntervalCapsAtMaximum(t *testing.T) {
	cfg := DefaultConfig
	cfg.MaximumIntervalDays = 30
	scheduler := newDeterministicScheduler(DefaultWeights, cfg)

	if result := scheduler.finalizeInterval(500); result != 30 {
		t.Fatalf("expected interval capped at 30, got %f", result)
	}
}

func TestGraduationFirstIntervalMatchesHandComputedFormula(t *testing.T) {
	scheduler := newDeterministicScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingEasy, now, 1)

	// At DesiredRetention = 0.9, baseInterval = S / Factor * (0.9^-2 - 1) reduces to
	// exactly S, since 0.9^-2 - 1 = 19/81 = Factor by construction. So the first
	// interval is fully determined by algebra, no exp/pow approximation needed.
	expectedStability := DefaultWeights[3]                      // InitialStability(Easy) = w[3]
	expectedDifficulty := DefaultWeights[4] - DefaultWeights[5] // InitialDifficulty(Easy) = w[4] - w[5]*(4-3)
	difficultyFactor := (11 - expectedDifficulty) / 10
	expectedInterval := math.Round(expectedStability * difficultyFactor)

	if math.Abs(state.Stability-expectedStability) > 1e-9 {
		t.Fatalf("stability: got %f want %f", state.Stability, expectedStability)
	}
	if math.Abs(state.EffectiveDifficulty-expectedDifficulty) > 1e-9 {
		t.Fatalf("effective difficulty: got %f want %f", state.EffectiveDifficulty, expectedDifficulty)
	}
	if state.IntervalDays != expectedInterval {
		t.Fatalf("first interval: got %f want %f", state.IntervalDays, expectedInterval)
	}
}

func TestConsequentReviewComposesPrimitivesDirectly(t *testing.T) {
	scheduler := newDeterministicScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()
	lastReview := now.Add(-10 * 24 * time.Hour)

	state := CardState{
		Stability:  10,
		Difficulty: 5,
		LastReview: &lastReview,
		Status:     StatusReview,
	}

	daysSince := now.Sub(lastReview).Hours() / 24
	retrievability := Retrievability(daysSince, state.Stability)
	expectedStability := StabilityAfterRecall(state.Stability, state.Difficulty, retrievability, RatingGood, DefaultWeights)
	expectedDifficulty := UpdateDifficulty(state.Difficulty, RatingGood, DefaultWeights)
	expectedEffectiveDifficulty := EffectiveDifficulty(expectedDifficulty, 1, scheduler.HierarchicalPenalty())
	expectedInterval := scheduler.finalizeInterval(NextInterval(expectedStability, expectedEffectiveDifficulty, DefaultConfig.DesiredRetention))

	got := scheduler.Schedule(state, RatingGood, now, 1)

	if math.Abs(got.Stability-expectedStability) > 1e-9 {
		t.Fatalf("stability: got %f want %f", got.Stability, expectedStability)
	}
	if math.Abs(got.EffectiveDifficulty-expectedEffectiveDifficulty) > 1e-9 {
		t.Fatalf("effective difficulty: got %f want %f", got.EffectiveDifficulty, expectedEffectiveDifficulty)
	}
	if got.IntervalDays != expectedInterval {
		t.Fatalf("consequent interval: got %f want %f", got.IntervalDays, expectedInterval)
	}
}

func TestConsequentReviewIntervalsGrowWithRepeatedSuccess(t *testing.T) {
	scheduler := newDeterministicScheduler(DefaultWeights, DefaultConfig)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingEasy, now, 1)
	if state.Status != StatusReview {
		t.Fatalf("expected immediate graduation, got %s", state.Status)
	}

	reviewTime := now
	for i := 0; i < 3; i++ {
		reviewTime = reviewTime.Add(daysToDuration(state.IntervalDays))
		next := scheduler.Schedule(state, RatingGood, reviewTime, 1)

		if next.IntervalDays <= state.IntervalDays {
			t.Fatalf("review %d: expected interval to grow, got %f after previous %f", i, next.IntervalDays, state.IntervalDays)
		}
		if next.Stability <= state.Stability {
			t.Fatalf("review %d: expected stability to grow, got %f after previous %f", i, next.Stability, state.Stability)
		}

		state = next
	}
}

func TestScheduleReviewRespectsMaximumInterval(t *testing.T) {
	cfg := DefaultConfig
	cfg.MaximumIntervalDays = 10
	scheduler := newDeterministicScheduler(DefaultWeights, cfg)
	now := time.Now().UTC()
	lastReview := now.Add(-24 * time.Hour)

	state := CardState{
		Stability:  400,
		Difficulty: 1,
		LastReview: &lastReview,
		Status:     StatusReview,
	}

	next := scheduler.Schedule(state, RatingEasy, now, 1)
	if next.IntervalDays > 10 {
		t.Fatalf("expected interval capped at 10 days, got %f", next.IntervalDays)
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
