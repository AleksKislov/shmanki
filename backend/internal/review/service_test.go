package review

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"shmanki/internal/fsrs"
)

func TestSubmitReviewPersistsLearningStepForInitialLearning(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()
	infoObjectID := uuid.New()

	store := &stubStore{
		state: stateRecord{
			CardID:         cardID,
			InfoObjectID:   infoObjectID,
			Step:           0,
			CorrectAnswers: [][]string{{"go", "worker()"}},
			Status:         fsrs.StatusNew,
		},
	}

	svc := NewService(store, fsrs.NewScheduler(fsrs.DefaultWeights, fsrs.DefaultConfig))
	result, err := svc.SubmitReview(context.Background(), userID, ReviewRequest{
		CardID:                cardID,
		AnsweredTokens:        []string{"go", "worker()"},
		WrongAttemptsCount:    0,
		DistractorClicksCount: 1,
	})
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}

	if store.updated == nil {
		t.Fatal("expected UpdateCardState to be called")
	}
	if store.updated.LearningStep != 1 {
		t.Fatalf("expected persisted learning_step=1, got %d", store.updated.LearningStep)
	}
	if store.updated.Status != fsrs.StatusLearning {
		t.Fatalf("expected persisted status learning, got %s", store.updated.Status)
	}
	if result.State.LearningStep != 1 {
		t.Fatalf("expected response learningStep=1, got %d", result.State.LearningStep)
	}
	if !store.committed {
		t.Fatal("expected transaction commit")
	}
}

func TestSubmitReviewPersistsLearningStepForRelearning(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()
	infoObjectID := uuid.New()
	now := time.Now().UTC()
	lastReview := now.Add(-48 * time.Hour)

	store := &stubStore{
		state: stateRecord{
			CardID:         cardID,
			InfoObjectID:   infoObjectID,
			Step:           1,
			CorrectAnswers: [][]string{{"correct"}},
			Stability:      22,
			Difficulty:     5,
			Retrievability: 0.9,
			LastReview:     &lastReview,
			IntervalDays:   7,
			Status:         fsrs.StatusReview,
			Reps:           4,
		},
	}

	svc := NewService(store, fsrs.NewScheduler(fsrs.DefaultWeights, fsrs.DefaultConfig))
	result, err := svc.SubmitReview(context.Background(), userID, ReviewRequest{
		CardID:                cardID,
		AnsweredTokens:        []string{"wrong"},
		WrongAttemptsCount:    1,
		DistractorClicksCount: 0,
	})
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}

	if store.updated == nil {
		t.Fatal("expected UpdateCardState to be called")
	}
	if store.updated.Status != fsrs.StatusRelearning {
		t.Fatalf("expected persisted status relearning, got %s", store.updated.Status)
	}
	if store.updated.LearningStep != 0 {
		t.Fatalf("expected persisted relearning step 0, got %d", store.updated.LearningStep)
	}
	if store.updated.Lapses != 1 {
		t.Fatalf("expected lapses incremented to 1, got %d", store.updated.Lapses)
	}
	if result.State.LearningStep != 0 {
		t.Fatalf("expected response learningStep=0, got %d", result.State.LearningStep)
	}
}

type stubStore struct {
	state     stateRecord
	updated   *stateRecord
	logged    *reviewLogParams
	committed bool
}

func (s *stubStore) EnsureStatesForUser(_ context.Context, _ uuid.UUID) error { return nil }

func (s *stubStore) GetSessionCards(_ context.Context, _ uuid.UUID, _ int) ([]ReviewCard, error) {
	return nil, nil
}

func (s *stubStore) Begin(_ context.Context) (pgx.Tx, error) {
	return &stubTx{onCommit: func() { s.committed = true }}, nil
}

func (s *stubStore) GetStateForUpdate(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ uuid.UUID) (*stateRecord, error) {
	copy := s.state
	return &copy, nil
}

func (s *stubStore) GetPreviousStepStabilities(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ uuid.UUID, _ int) ([]float64, error) {
	return nil, nil
}

func (s *stubStore) UpdateCardState(_ context.Context, _ pgx.Tx, _ uuid.UUID, state stateRecord) error {
	copy := state
	s.updated = &copy
	return nil
}

func (s *stubStore) InsertReviewLog(_ context.Context, _ pgx.Tx, params reviewLogParams) error {
	copy := params
	s.logged = &copy
	return nil
}

func (s *stubStore) ShouldUnlockStep(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ uuid.UUID, _ int, _ float64) (bool, error) {
	return false, nil
}

func (s *stubStore) UnlockStep(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}

func (s *stubStore) GetDeckStats(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*DeckStats, error) {
	return nil, nil
}

func (s *stubStore) GetCardHistory(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]CardHistoryEntry, error) {
	return nil, nil
}

type stubTx struct {
	onCommit func()
}

func (t *stubTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }

func (t *stubTx) Commit(context.Context) error {
	if t.onCommit != nil {
		t.onCommit()
	}
	return nil
}

func (t *stubTx) Rollback(context.Context) error { return nil }

func (t *stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (t *stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }

func (t *stubTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

func (t *stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (t *stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (t *stubTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func (t *stubTx) Conn() *pgx.Conn { return nil }

func TestSubmitReviewLogsOptimizerFields(t *testing.T) {
	userID := uuid.New()
	cardID := uuid.New()
	lastReview := time.Now().UTC().Add(-72 * time.Hour)
	duration := 4200

	store := &stubStore{
		state: stateRecord{
			CardID:         cardID,
			InfoObjectID:   uuid.New(),
			Step:           0,
			CorrectAnswers: [][]string{{"go", "worker()"}},
			Status:         fsrs.StatusReview,
			Stability:      12,
			Difficulty:     5,
			IntervalDays:   10,
			LastReview:     &lastReview,
		},
	}

	svc := NewService(store, fsrs.NewScheduler(fsrs.DefaultWeights, fsrs.DefaultConfig))
	if _, err := svc.SubmitReview(context.Background(), userID, ReviewRequest{
		CardID:         cardID,
		AnsweredTokens: []string{"go", "worker()"},
		DurationMs:     &duration,
		Timezone:       "Europe/Belgrade",
	}); err != nil {
		t.Fatalf("submit review: %v", err)
	}

	if store.logged == nil {
		t.Fatal("expected a review log to be written")
	}
	if store.logged.ParamsVersion != fsrs.DefaultParamsVersion {
		t.Fatalf("params version: got %q want %q", store.logged.ParamsVersion, fsrs.DefaultParamsVersion)
	}
	// Elapsed is what actually happened; interval_before is what was scheduled.
	// The optimizer learns from the gap, so they must not be conflated.
	if math.Abs(store.logged.ElapsedDays-3) > 0.01 {
		t.Fatalf("elapsed days: got %f want ~3", store.logged.ElapsedDays)
	}
	if store.logged.IntervalBefore != 10 {
		t.Fatalf("interval before: got %f want 10", store.logged.IntervalBefore)
	}
	if store.logged.ReviewDurationMs == nil || *store.logged.ReviewDurationMs != duration {
		t.Fatalf("review duration: got %v want %d", store.logged.ReviewDurationMs, duration)
	}
	if store.logged.UserTimezone == nil || *store.logged.UserTimezone != "Europe/Belgrade" {
		t.Fatalf("timezone: got %v want Europe/Belgrade", store.logged.UserTimezone)
	}
}

func TestSubmitReviewDropsImplausibleClientTelemetry(t *testing.T) {
	tooLong := maxReviewDurationMs + 1
	negative := -5

	if got := sanitizeDuration(&tooLong); got != nil {
		t.Fatalf("expected a tab-left-open duration to be dropped, got %d", *got)
	}
	if got := sanitizeDuration(&negative); got != nil {
		t.Fatalf("expected a negative duration to be dropped, got %d", *got)
	}
	if got := sanitizeDuration(nil); got != nil {
		t.Fatal("expected nil duration to stay nil")
	}
	for _, tz := range []string{"", "Europe/Belgrade'; DROP TABLE review_logs--", "Zone With Spaces"} {
		if got := sanitizeTimezone(tz); got != nil {
			t.Fatalf("expected timezone %q to be rejected, got %q", tz, *got)
		}
	}
	if got := sanitizeTimezone("America/Argentina/Buenos_Aires"); got == nil {
		t.Fatal("expected a valid IANA zone to be accepted")
	}
	if got := sanitizeTimezone("UTC"); got == nil {
		t.Fatal("expected UTC to be accepted")
	}
}

func TestElapsedDaysForFirstReviewIsZero(t *testing.T) {
	if got := elapsedDaysSince(nil, time.Now()); got != 0 {
		t.Fatalf("expected 0 elapsed days for a first review, got %f", got)
	}

	// Clock skew or a backdated client must not produce negative elapsed time.
	future := time.Now().Add(time.Hour)
	if got := elapsedDaysSince(&future, time.Now()); got != 0 {
		t.Fatalf("expected negative elapsed time to floor at 0, got %f", got)
	}
}
