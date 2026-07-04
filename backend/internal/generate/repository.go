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

func (r *Repository) UpsertGenerationDraft(ctx context.Context, tx pgx.Tx, draft generationDraft) error {
	const query = `
INSERT INTO generation_drafts (generation_id, user_id, deck_id, objects_raw, model, expires_at)
VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '24 hours')
ON CONFLICT (generation_id) DO UPDATE
SET objects_raw = EXCLUDED.objects_raw,
    model = EXCLUDED.model,
    updated_at = NOW(),
    expires_at = NOW() + INTERVAL '24 hours'
`
	if _, err := tx.Exec(ctx, query, draft.GenerationID, draft.UserID, draft.DeckID, draft.ObjectsRaw, draft.Model); err != nil {
		return fmt.Errorf("upsert generation draft: %w", err)
	}
	return nil
}

func (r *Repository) GetInfoObjectContent(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) (*objectContent, error) {
	const query = `
SELECT io.content, io.content_type, io.discipline, d.language_code
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $1 AND d.user_id = $2
`
	var oc objectContent
	if err := r.db.QueryRow(ctx, query, objectID, userID).Scan(&oc.Content, &oc.ContentType, &oc.Discipline, &oc.LanguageCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInfoObjectNotFound
		}
		return nil, fmt.Errorf("get info object content: %w", err)
	}
	return &oc, nil
}

func (r *Repository) GetGenerationDraft(ctx context.Context, tx pgx.Tx, userID uuid.UUID, generationID uuid.UUID) (*generationDraft, error) {
	const query = `
SELECT generation_id, user_id, deck_id, objects_raw, model
FROM generation_drafts
WHERE generation_id = $1 AND user_id = $2 AND expires_at > NOW()
`
	var item generationDraft
	if err := tx.QueryRow(ctx, query, generationID, userID).Scan(
		&item.GenerationID,
		&item.UserID,
		&item.DeckID,
		&item.ObjectsRaw,
		&item.Model,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("get generation draft: %w", err)
	}
	return &item, nil
}

func (r *Repository) DeleteGenerationDraft(ctx context.Context, tx pgx.Tx, userID uuid.UUID, generationID uuid.UUID) error {
	const query = `DELETE FROM generation_drafts WHERE generation_id = $1 AND user_id = $2`
	if _, err := tx.Exec(ctx, query, generationID, userID); err != nil {
		return fmt.Errorf("delete generation draft: %w", err)
	}
	return nil
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

	const insertCardQuery = `
INSERT INTO cards (info_object_id, front, card_type, step, correct_answers, distractors)
SELECT io.id, $3, $4, $5, $6, $7
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $1 AND d.user_id = $2
RETURNING id, info_object_id, front, card_type, step, correct_answers, distractors, created_at, updated_at
`

	var item SavedCard
	var rawAnswers []byte
	var rawDistractors []byte
	if err := tx.QueryRow(ctx, insertCardQuery, infoObjectID, userID, card.Front, card.CardType, card.Step, answersJSON, distractorsJSON).Scan(
		&item.ID,
		&item.InfoObjectID,
		&item.Front,
		&item.CardType,
		&item.Step,
		&rawAnswers,
		&rawDistractors,
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
