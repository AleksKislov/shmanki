package review

import (
	"context"
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

func (s *stubStore) InsertReviewLog(_ context.Context, _ pgx.Tx, _ reviewLogParams) error { return nil }

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
