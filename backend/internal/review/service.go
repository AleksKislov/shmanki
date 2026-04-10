package review

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shmanki/internal/fsrs"
)

var (
	ErrReviewCardNotFound = errors.New("card not found")
	ErrInvalidSubmission  = errors.New("invalid review submission")
)

type store interface {
	EnsureStatesForUser(ctx context.Context, userID uuid.UUID) error
	GetSessionCards(ctx context.Context, userID uuid.UUID, limit int) ([]ReviewCard, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	GetStateForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, cardID uuid.UUID) (*stateRecord, error)
	GetPreviousStepStabilities(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, step int) ([]float64, error)
	UpdateCardState(ctx context.Context, tx pgx.Tx, userID uuid.UUID, state stateRecord) error
	InsertReviewLog(ctx context.Context, tx pgx.Tx, params reviewLogParams) error
	ShouldUnlockStep(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, nextStep int, threshold float64) (bool, error)
	UnlockStep(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, step int) error
	GetDeckStats(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*DeckStats, error)
	GetCardHistory(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) ([]CardHistoryEntry, error)
}

type Service struct {
	store     store
	scheduler *fsrs.Scheduler
}

func NewService(store store, scheduler *fsrs.Scheduler) *Service {
	return &Service{store: store, scheduler: scheduler}
}

func (s *Service) GetSession(ctx context.Context, userID uuid.UUID, limit int) ([]ReviewCard, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if err := s.store.EnsureStatesForUser(ctx, userID); err != nil {
		return nil, err
	}

	return s.store.GetSessionCards(ctx, userID, limit)
}

func (s *Service) SubmitReview(ctx context.Context, userID uuid.UUID, req ReviewRequest) (*ReviewResult, error) {
	if req.CardID == uuid.Nil || req.WrongAttemptsCount < 0 || req.DistractorClicksCount < 0 {
		return nil, ErrInvalidSubmission
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	currentState, err := s.store.GetStateForUpdate(ctx, tx, userID, req.CardID)
	if err != nil {
		return nil, err
	}

	wasCorrect := answersMatch(currentState.CorrectAnswers, req.AnsweredTokens)
	rating := deriveRating(wasCorrect, req.WrongAttemptsCount, req.DistractorClicksCount)
	now := nowUTC()

	hierarchicalSupport, err := s.hierarchicalSupport(ctx, tx, userID, currentState.InfoObjectID, currentState.Step)
	if err != nil {
		return nil, err
	}

	before := toSchedulerState(*currentState)
	after := s.scheduler.Schedule(before, rating, now, hierarchicalSupport)

	updatedState := *currentState
	updatedState.Stability = after.Stability
	updatedState.Difficulty = after.Difficulty
	updatedState.Retrievability = after.Retrievability
	updatedState.DueDate = after.DueDate
	updatedState.LastReview = after.LastReview
	updatedState.IntervalDays = after.IntervalDays
	updatedState.Status = after.Status
	updatedState.Reps = after.Reps
	updatedState.Lapses = after.Lapses

	if err := s.store.UpdateCardState(ctx, tx, userID, updatedState); err != nil {
		return nil, err
	}

	if err := s.store.InsertReviewLog(ctx, tx, reviewLogParams{
		CardID:                 req.CardID,
		UserID:                 userID,
		StabilityBefore:        currentState.Stability,
		DifficultyBefore:       currentState.Difficulty,
		RetrievabilityBefore:   currentState.Retrievability,
		IntervalBefore:         currentState.IntervalDays,
		StatusBefore:           string(currentState.Status),
		StabilityAfter:         updatedState.Stability,
		DifficultyAfter:        updatedState.Difficulty,
		IntervalAfter:          updatedState.IntervalDays,
		StatusAfter:            string(updatedState.Status),
		Rating:                 rating,
		AnsweredTokens:         req.AnsweredTokens,
		WasCorrect:             wasCorrect,
		WrongAttemptsCount:     req.WrongAttemptsCount,
		DistractorClicksCount:  req.DistractorClicksCount,
		IncorrectTokensClicked: req.IncorrectTokensClicked,
		Attempts:               req.Attempts,
	}); err != nil {
		return nil, err
	}

	nextStep := currentState.Step + 1
	shouldUnlock, err := s.store.ShouldUnlockStep(ctx, tx, userID, currentState.InfoObjectID, nextStep, s.scheduler.StepUnlockDays())
	if err != nil {
		return nil, err
	}
	if shouldUnlock {
		if err := s.store.UnlockStep(ctx, tx, userID, currentState.InfoObjectID, nextStep); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit review tx: %w", err)
	}

	return &ReviewResult{
		State: CardStateView{
			CardID:              req.CardID,
			Stability:           updatedState.Stability,
			Difficulty:          updatedState.Difficulty,
			Retrievability:      updatedState.Retrievability,
			DueDate:             updatedState.DueDate,
			Status:              string(updatedState.Status),
			Reps:                updatedState.Reps,
			Lapses:              updatedState.Lapses,
			IntervalDays:        updatedState.IntervalDays,
			LastReview:          updatedState.LastReview,
			EffectiveDifficulty: after.EffectiveDifficulty,
			HierarchicalSupport: after.HierarchicalSupport,
		},
		Rating:     rating,
		WasCorrect: wasCorrect,
	}, nil
}

func (s *Service) GetDeckStats(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*DeckStats, error) {
	return s.store.GetDeckStats(ctx, userID, deckID)
}

func (s *Service) GetCardHistory(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) ([]CardHistoryEntry, error) {
	return s.store.GetCardHistory(ctx, userID, cardID)
}

func deriveRating(wasCorrect bool, wrongAttemptsCount int, distractorClicksCount int) fsrs.Rating {
	if !wasCorrect {
		return fsrs.RatingAgain
	}
	if wrongAttemptsCount > 0 {
		return fsrs.RatingHard
	}
	if distractorClicksCount > 0 {
		return fsrs.RatingGood
	}
	return fsrs.RatingEasy
}

func answersMatch(validAnswers [][]string, answeredTokens []string) bool {
	for _, valid := range validAnswers {
		if reflect.DeepEqual(valid, answeredTokens) {
			return true
		}
	}
	return false
}

func toSchedulerState(state stateRecord) fsrs.CardState {
	return fsrs.CardState{
		Stability:      state.Stability,
		Difficulty:     state.Difficulty,
		Retrievability: state.Retrievability,
		DueDate:        state.DueDate,
		LastReview:     state.LastReview,
		IntervalDays:   state.IntervalDays,
		Status:         state.Status,
		Reps:           state.Reps,
		Lapses:         state.Lapses,
	}
}

func (s *Service) hierarchicalSupport(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, step int) (float64, error) {
	if step <= 0 {
		return 1, nil
	}

	stabilities, err := s.store.GetPreviousStepStabilities(ctx, tx, userID, infoObjectID, step)
	if err != nil {
		return 0, err
	}

	return fsrs.HierarchicalSupport(stabilities, s.scheduler.SupportReferenceDays()), nil
}
