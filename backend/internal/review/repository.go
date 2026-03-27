package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shmanki/internal/fsrs"
)

type Repository struct {
	db *pgxpool.Pool
}

type stateRecord struct {
	CardID         uuid.UUID
	InfoObjectID   uuid.UUID
	Step           int
	CorrectAnswers [][]string
	Stability      float64
	Difficulty     float64
	Retrievability float64
	DueDate        *time.Time
	LastReview     *time.Time
	IntervalDays   float64
	Status         fsrs.CardStatus
	Reps           int
	Lapses         int
}

type reviewLogParams struct {
	CardID                 uuid.UUID
	UserID                 uuid.UUID
	StabilityBefore        float64
	DifficultyBefore       float64
	RetrievabilityBefore   float64
	IntervalBefore         float64
	StatusBefore           string
	StabilityAfter         float64
	DifficultyAfter        float64
	IntervalAfter          float64
	StatusAfter            string
	Rating                 fsrs.Rating
	AnsweredTokens         []string
	WasCorrect             bool
	WrongAttemptsCount     int
	DistractorClicksCount  int
	IncorrectTokensClicked []string
	Attempts               []ReviewAttempt
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureStatesForUser(ctx context.Context, userID uuid.UUID) error {
	const query = `
INSERT INTO card_states (card_id, user_id, due_date, status)
SELECT c.id, $1,
       CASE WHEN c.step = 0 THEN NOW() ELSE NULL END,
       CASE WHEN c.step = 0 THEN 'new' ELSE 'locked' END
FROM cards c
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d ON d.id = io.deck_id
LEFT JOIN card_states cs ON cs.card_id = c.id AND cs.user_id = $1
WHERE d.user_id = $1 AND cs.id IS NULL
`

	if _, err := r.db.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("ensure card states: %w", err)
	}

	return nil
}

func (r *Repository) GetSessionCards(ctx context.Context, userID uuid.UUID, limit int) ([]ReviewCard, error) {
	const query = `
SELECT
    c.id,
    c.front,
    c.correct_answers,
    c.distractors,
    c.highlight_lines,
    c.step,
    cs.stability,
    cs.difficulty,
    cs.retrievability,
    cs.due_date,
    cs.status,
    cs.reps,
    cs.lapses,
    cs.interval_days,
    cs.last_review,
    io.content,
    io.content_type,
    d.language_code,
    io.id
FROM card_states cs
JOIN cards c ON c.id = cs.card_id
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d ON d.id = io.deck_id
WHERE cs.user_id = $1
  AND d.user_id = $1
  AND (
      (cs.status IN ('learning', 'review', 'relearning') AND cs.due_date <= NOW())
      OR cs.status = 'new'
  )
ORDER BY cs.due_date ASC NULLS FIRST, c.created_at ASC
LIMIT $2
`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get session cards: %w", err)
	}
	defer rows.Close()

	items := make([]ReviewCard, 0)
	for rows.Next() {
		item, err := scanReviewCard(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session cards: %w", err)
	}

	return items, nil
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin review tx: %w", err)
	}
	return tx, nil
}

func (r *Repository) GetStateForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, cardID uuid.UUID) (*stateRecord, error) {
	const query = `
SELECT
    c.id,
    io.id,
    c.step,
    c.correct_answers,
    cs.stability,
    cs.difficulty,
    cs.retrievability,
    cs.due_date,
    cs.last_review,
    cs.interval_days,
    cs.status,
    cs.reps,
    cs.lapses
FROM card_states cs
JOIN cards c ON c.id = cs.card_id
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d ON d.id = io.deck_id
WHERE cs.user_id = $1 AND cs.card_id = $2 AND d.user_id = $1
FOR UPDATE
`

	var item stateRecord
	var rawAnswers []byte
	err := tx.QueryRow(ctx, query, userID, cardID).Scan(
		&item.CardID,
		&item.InfoObjectID,
		&item.Step,
		&rawAnswers,
		&item.Stability,
		&item.Difficulty,
		&item.Retrievability,
		&item.DueDate,
		&item.LastReview,
		&item.IntervalDays,
		&item.Status,
		&item.Reps,
		&item.Lapses,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReviewCardNotFound
		}
		return nil, fmt.Errorf("get state for update: %w", err)
	}
	if err := json.Unmarshal(rawAnswers, &item.CorrectAnswers); err != nil {
		return nil, fmt.Errorf("decode review correct answers: %w", err)
	}

	return &item, nil
}

func (r *Repository) UpdateCardState(ctx context.Context, tx pgx.Tx, userID uuid.UUID, state stateRecord) error {
	const query = `
UPDATE card_states
SET stability = $3,
    difficulty = $4,
    retrievability = $5,
    due_date = $6,
    last_review = $7,
    interval_days = $8,
    status = $9,
    reps = $10,
    lapses = $11
WHERE card_id = $1 AND user_id = $2
`

	_, err := tx.Exec(ctx, query, state.CardID, userID, state.Stability, state.Difficulty, state.Retrievability, state.DueDate, state.LastReview, state.IntervalDays, state.Status, state.Reps, state.Lapses)
	if err != nil {
		return fmt.Errorf("update card state: %w", err)
	}

	return nil
}

func (r *Repository) InsertReviewLog(ctx context.Context, tx pgx.Tx, params reviewLogParams) error {
	answeredTokens, err := json.Marshal(params.AnsweredTokens)
	if err != nil {
		return fmt.Errorf("marshal answered tokens: %w", err)
	}
	incorrectTokens, err := json.Marshal(params.IncorrectTokensClicked)
	if err != nil {
		return fmt.Errorf("marshal incorrect tokens: %w", err)
	}
	attempts, err := json.Marshal(params.Attempts)
	if err != nil {
		return fmt.Errorf("marshal attempts: %w", err)
	}

	const query = `
INSERT INTO review_logs (
    card_id, user_id,
    stability_before, difficulty_before, retrievability_before, interval_before, status_before,
    stability_after, difficulty_after, interval_after, status_after,
    rating, answered_tokens, was_correct, wrong_attempts_count,
    distractor_clicks_count, incorrect_tokens_clicked, attempts
)
VALUES (
    $1, $2,
    $3, $4, $5, $6, $7,
    $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17, $18
)
`

	_, err = tx.Exec(ctx, query,
		params.CardID,
		params.UserID,
		params.StabilityBefore,
		params.DifficultyBefore,
		params.RetrievabilityBefore,
		params.IntervalBefore,
		params.StatusBefore,
		params.StabilityAfter,
		params.DifficultyAfter,
		params.IntervalAfter,
		params.StatusAfter,
		params.Rating,
		answeredTokens,
		params.WasCorrect,
		params.WrongAttemptsCount,
		params.DistractorClicksCount,
		incorrectTokens,
		attempts,
	)
	if err != nil {
		return fmt.Errorf("insert review log: %w", err)
	}

	return nil
}

func (r *Repository) ShouldUnlockStep(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, nextStep int, threshold float64) (bool, error) {
	if nextStep <= 0 {
		return false, nil
	}

	const nextStepExistsQuery = `
SELECT EXISTS(
    SELECT 1
    FROM cards c
    JOIN info_objects io ON io.id = c.info_object_id
    JOIN decks d ON d.id = io.deck_id
    WHERE c.info_object_id = $1 AND c.step = $2 AND d.user_id = $3
)
`

	var exists bool
	if err := tx.QueryRow(ctx, nextStepExistsQuery, infoObjectID, nextStep, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check next step existence: %w", err)
	}
	if !exists {
		return false, nil
	}

	const shouldUnlockQuery = `
SELECT COUNT(*) = 0
FROM cards c
JOIN card_states cs ON cs.card_id = c.id AND cs.user_id = $1
WHERE c.info_object_id = $2 AND c.step = $3 AND cs.stability < $4
`

	var shouldUnlock bool
	if err := tx.QueryRow(ctx, shouldUnlockQuery, userID, infoObjectID, nextStep-1, threshold).Scan(&shouldUnlock); err != nil {
		return false, fmt.Errorf("check step unlock: %w", err)
	}

	return shouldUnlock, nil
}

func (r *Repository) UnlockStep(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, step int) error {
	const query = `
UPDATE card_states
SET status = 'new', due_date = NOW()
WHERE user_id = $1
  AND status = 'locked'
  AND card_id IN (
      SELECT id FROM cards
      WHERE info_object_id = $2 AND step = $3
  )
`

	if _, err := tx.Exec(ctx, query, userID, infoObjectID, step); err != nil {
		return fmt.Errorf("unlock step: %w", err)
	}

	return nil
}

func (r *Repository) GetDeckStats(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*DeckStats, error) {
	const totalsQuery = `
SELECT
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE cs.status = 'new') AS new_now,
    COUNT(*) FILTER (WHERE cs.status IN ('learning', 'review', 'relearning') AND cs.due_date <= NOW()) AS due_now
FROM card_states cs
JOIN cards c ON c.id = cs.card_id
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d ON d.id = io.deck_id
WHERE cs.user_id = $1 AND d.user_id = $1 AND d.id = $2
`

	stats := &DeckStats{DeckID: deckID, Levels: map[string]int64{"new": 0, "learning": 0, "learned": 0, "mastered": 0, "expert": 0}}
	if err := r.db.QueryRow(ctx, totalsQuery, userID, deckID).Scan(&stats.Total, &stats.NewNow, &stats.DueNow); err != nil {
		return nil, fmt.Errorf("get deck stats totals: %w", err)
	}

	const levelsQuery = `
SELECT
    CASE
        WHEN cs.stability < 7 THEN 'new'
        WHEN cs.stability < 21 THEN 'learning'
        WHEN cs.stability < 90 THEN 'learned'
        WHEN cs.stability < 365 THEN 'mastered'
        ELSE 'expert'
    END AS level,
    COUNT(*)
FROM card_states cs
JOIN cards c ON c.id = cs.card_id
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d ON d.id = io.deck_id
WHERE cs.user_id = $1 AND d.user_id = $1 AND d.id = $2
GROUP BY level
`

	rows, err := r.db.Query(ctx, levelsQuery, userID, deckID)
	if err != nil {
		return nil, fmt.Errorf("get deck stats levels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan deck stats level: %w", err)
		}
		stats.Levels[level] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deck stats levels: %w", err)
	}

	return stats, nil
}

func (r *Repository) GetCardHistory(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) ([]CardHistoryEntry, error) {
	const query = `
SELECT reviewed_at, rating, was_correct, stability_before, stability_after,
       difficulty_before, difficulty_after, wrong_attempts_count,
       distractor_clicks_count, answered_tokens, incorrect_tokens_clicked
FROM review_logs
WHERE user_id = $1 AND card_id = $2
ORDER BY reviewed_at DESC
`

	rows, err := r.db.Query(ctx, query, userID, cardID)
	if err != nil {
		return nil, fmt.Errorf("get card history: %w", err)
	}
	defer rows.Close()

	history := make([]CardHistoryEntry, 0)
	for rows.Next() {
		var item CardHistoryEntry
		var answeredTokens []byte
		var incorrectTokens []byte
		if err := rows.Scan(&item.ReviewedAt, &item.Rating, &item.WasCorrect, &item.StabilityBefore, &item.StabilityAfter, &item.DifficultyBefore, &item.DifficultyAfter, &item.WrongAttemptsCount, &item.DistractorClicksCount, &answeredTokens, &incorrectTokens); err != nil {
			return nil, fmt.Errorf("scan card history: %w", err)
		}
		if err := json.Unmarshal(answeredTokens, &item.AnsweredTokens); err != nil {
			return nil, fmt.Errorf("decode answered tokens: %w", err)
		}
		if err := json.Unmarshal(incorrectTokens, &item.IncorrectTokens); err != nil {
			return nil, fmt.Errorf("decode incorrect tokens: %w", err)
		}
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate card history: %w", err)
	}

	return history, nil
}

func scanReviewCard(rows pgx.Rows) (ReviewCard, error) {
	var item ReviewCard
	var rawAnswers []byte
	var rawDistractors []byte
	var rawHighlight []byte
	if err := rows.Scan(
		&item.CardID,
		&item.Front,
		&rawAnswers,
		&rawDistractors,
		&rawHighlight,
		&item.Step,
		&item.State.Stability,
		&item.State.Difficulty,
		&item.State.Retrievability,
		&item.State.DueDate,
		&item.State.Status,
		&item.State.Reps,
		&item.State.Lapses,
		&item.State.IntervalDays,
		&item.State.LastReview,
		&item.Content,
		&item.ContentType,
		&item.LanguageCode,
		&item.InfoObjectID,
	); err != nil {
		return ReviewCard{}, fmt.Errorf("scan review card: %w", err)
	}
	item.State.CardID = item.CardID
	if err := json.Unmarshal(rawAnswers, &item.CorrectAnswers); err != nil {
		return ReviewCard{}, fmt.Errorf("decode review card answers: %w", err)
	}
	if err := json.Unmarshal(rawDistractors, &item.Distractors); err != nil {
		return ReviewCard{}, fmt.Errorf("decode review card distractors: %w", err)
	}
	if err := json.Unmarshal(rawHighlight, &item.HighlightLines); err != nil {
		return ReviewCard{}, fmt.Errorf("decode review card highlight lines: %w", err)
	}
	return item, nil
}
