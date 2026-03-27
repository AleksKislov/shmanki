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
	interval := NextInterval(0.1, 0.9)
	if interval < 1 {
		t.Fatalf("expected minimum interval >= 1, got %f", interval)
	}
}

func TestScheduleInitialReview(t *testing.T) {
	scheduler := NewScheduler(DefaultWeights, 0.9, 14)
	now := time.Now().UTC()

	state := scheduler.Schedule(CardState{Status: StatusNew}, RatingGood, now)

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
