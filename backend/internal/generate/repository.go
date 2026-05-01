package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin generation tx: %w", err)
	}
	return tx, nil
}

func (r *Repository) GetDeckLanguage(ctx context.Context, tx pgx.Tx, userID uuid.UUID, deckID uuid.UUID) (string, error) {
	const query = `SELECT language_code FROM decks WHERE id = $1 AND user_id = $2`
	var languageCode string
	if err := tx.QueryRow(ctx, query, deckID, userID).Scan(&languageCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrDeckNotFound
		}
		return "", fmt.Errorf("get deck language: %w", err)
	}
	return languageCode, nil
}

func (r *Repository) InsertGenerationLog(ctx context.Context, tx pgx.Tx, log generationLog) (*generationLog, error) {
	const query = `
INSERT INTO generation_logs (deck_id, user_id, prompt, provider, model, objects_raw, cards_count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, deck_id, user_id, prompt, provider, model, objects_raw, cards_count, created_at
`

	var item generationLog
	if err := tx.QueryRow(ctx, query, log.DeckID, log.UserID, log.Prompt, log.Provider, log.Model, log.ObjectsRaw, log.CardsCount).Scan(
		&item.ID,
		&item.DeckID,
		&item.UserID,
		&item.Prompt,
		&item.Provider,
		&item.Model,
		&item.ObjectsRaw,
		&item.CardsCount,
		&item.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert generation log: %w", err)
	}

	return &item, nil
}

func (r *Repository) CreateInfoObject(ctx context.Context, tx pgx.Tx, userID uuid.UUID, deckID uuid.UUID, object SuggestedObject) (*SavedObject, error) {
	const query = `
INSERT INTO info_objects (deck_id, title, content, discipline, content_type)
SELECT $1, $3, $4, $5, $6
FROM decks
WHERE id = $1 AND user_id = $2
RETURNING id, deck_id, title, content, discipline, content_type, created_at, updated_at
`

	var item SavedObject
	if err := tx.QueryRow(ctx, query, deckID, userID, object.Title, object.Content, object.Discipline, object.ContentType).Scan(
		&item.ID,
		&item.DeckID,
		&item.Title,
		&item.Content,
		&item.Discipline,
		&item.ContentType,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeckNotFound
		}
		return nil, fmt.Errorf("create generated info object: %w", err)
	}

	return &item, nil
}

func (r *Repository) CreateCard(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, card SuggestedCard) (*SavedCard, error) {
	answersJSON, err := json.Marshal(card.CorrectAnswers)
	if err != nil {
		return nil, fmt.Errorf("marshal card correct answers: %w", err)
	}
	distractorsJSON, err := json.Marshal(card.Distractors)
	if err != nil {
		return nil, fmt.Errorf("marshal card distractors: %w", err)
	}
	highlightJSON, err := json.Marshal(card.HighlightLines)
	if err != nil {
		return nil, fmt.Errorf("marshal card highlight lines: %w", err)
	}

	const insertCardQuery = `
INSERT INTO cards (info_object_id, front, card_type, step, correct_answers, distractors, highlight_lines)
SELECT io.id, $3, $4, $5, $6, $7, $8
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $1 AND d.user_id = $2
RETURNING id, info_object_id, front, card_type, step, correct_answers, distractors, highlight_lines, created_at, updated_at
`

	var item SavedCard
	var rawAnswers []byte
	var rawDistractors []byte
	var rawHighlights []byte
	if err := tx.QueryRow(ctx, insertCardQuery, infoObjectID, userID, card.Front, card.CardType, card.Step, answersJSON, distractorsJSON, highlightJSON).Scan(
		&item.ID,
		&item.InfoObjectID,
		&item.Front,
		&item.CardType,
		&item.Step,
		&rawAnswers,
		&rawDistractors,
		&rawHighlights,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidSuggestion
		}
		return nil, fmt.Errorf("create generated card: %w", err)
	}

	if err := json.Unmarshal(rawAnswers, &item.CorrectAnswers); err != nil {
		return nil, fmt.Errorf("decode saved correct answers: %w", err)
	}
	if err := json.Unmarshal(rawDistractors, &item.Distractors); err != nil {
		return nil, fmt.Errorf("decode saved distractors: %w", err)
	}
	if err := json.Unmarshal(rawHighlights, &item.HighlightLines); err != nil {
		return nil, fmt.Errorf("decode saved highlight lines: %w", err)
	}

	const insertStatesQuery = `
INSERT INTO card_states (card_id, user_id, due_date, status)
SELECT $1, d.user_id, CASE WHEN $2 = 0 THEN NOW() ELSE NULL END, CASE WHEN $2 = 0 THEN 'new' ELSE 'locked' END
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $3
ON CONFLICT (card_id, user_id) DO NOTHING
`

	if _, err := tx.Exec(ctx, insertStatesQuery, item.ID, item.Step, infoObjectID); err != nil {
		return nil, fmt.Errorf("insert generated card states: %w", err)
	}

	return &item, nil
}
