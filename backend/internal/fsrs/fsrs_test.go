package fsrs

import (
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
	if state.LastReview == nil || state.DueDate == nil {
		t.Fatal("expected timestamps to be set")
	}
	if state.Reps != 1 {
		t.Fatalf("expected reps to be incremented, got %d", state.Reps)
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
		Status:     StatusLearning,
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
