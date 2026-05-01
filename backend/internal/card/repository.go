package card

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

func (r *Repository) ListInfoObjectsByDeckID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) ([]InfoObject, error) {
	const query = `
SELECT io.id, io.deck_id, io.title, io.content, io.discipline, io.content_type, io.created_at, io.updated_at
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.deck_id = $1 AND d.user_id = $2
ORDER BY io.created_at ASC
`

	rows, err := r.db.Query(ctx, query, deckID, userID)
	if err != nil {
		return nil, fmt.Errorf("list info objects: %w", err)
	}
	defer rows.Close()

	objects := make([]InfoObject, 0)
	for rows.Next() {
		var item InfoObject
		if err := rows.Scan(&item.ID, &item.DeckID, &item.Title, &item.Content, &item.Discipline, &item.ContentType, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan info object: %w", err)
		}
		objects = append(objects, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate info objects: %w", err)
	}

	return objects, nil
}

func (r *Repository) CreateInfoObject(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req CreateInfoObjectRequest) (*InfoObject, error) {
	const query = `
INSERT INTO info_objects (deck_id, title, content, discipline, content_type)
SELECT $1, $3, $4, $5, $6
FROM decks
WHERE id = $1 AND user_id = $2
RETURNING id, deck_id, title, content, discipline, content_type, created_at, updated_at
`

	var item InfoObject
	err := r.db.QueryRow(ctx, query, deckID, userID, req.Title, req.Content, req.Discipline, req.ContentType).Scan(
		&item.ID,
		&item.DeckID,
		&item.Title,
		&item.Content,
		&item.Discipline,
		&item.ContentType,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInfoObjectNotFound
		}
		return nil, fmt.Errorf("create info object: %w", err)
	}

	return &item, nil
}

func (r *Repository) GetInfoObjectByID(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) (*InfoObjectDetail, error) {
	const objectQuery = `
SELECT io.id, io.deck_id, io.title, io.content, io.discipline, io.content_type, io.created_at, io.updated_at
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $1 AND d.user_id = $2
`

	var detail InfoObjectDetail
	err := r.db.QueryRow(ctx, objectQuery, objectID, userID).Scan(
		&detail.ID,
		&detail.DeckID,
		&detail.Title,
		&detail.Content,
		&detail.Discipline,
		&detail.ContentType,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInfoObjectNotFound
		}
		return nil, fmt.Errorf("get info object: %w", err)
	}

	const cardsQuery = `
SELECT id, info_object_id, front, card_type, step, correct_answers, distractors, highlight_lines, created_at, updated_at
FROM cards
WHERE info_object_id = $1
ORDER BY step ASC, created_at ASC
`

	rows, err := r.db.Query(ctx, cardsQuery, objectID)
	if err != nil {
		return nil, fmt.Errorf("list cards for info object: %w", err)
	}
	defer rows.Close()

	detail.Cards = make([]Card, 0)
	for rows.Next() {
		cardItem, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		detail.Cards = append(detail.Cards, cardItem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cards: %w", err)
	}

	return &detail, nil
}

func (r *Repository) UpdateInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req UpdateInfoObjectRequest) (*InfoObject, error) {
	const query = `
UPDATE info_objects io
SET title = $3, content = $4, discipline = $5, content_type = $6, updated_at = NOW()
FROM decks d
WHERE io.deck_id = d.id AND io.id = $1 AND d.user_id = $2
RETURNING io.id, io.deck_id, io.title, io.content, io.discipline, io.content_type, io.created_at, io.updated_at
`

	var item InfoObject
	err := r.db.QueryRow(ctx, query, objectID, userID, req.Title, req.Content, req.Discipline, req.ContentType).Scan(
		&item.ID,
		&item.DeckID,
		&item.Title,
		&item.Content,
		&item.Discipline,
		&item.ContentType,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInfoObjectNotFound
		}
		return nil, fmt.Errorf("update info object: %w", err)
	}

	return &item, nil
}

func (r *Repository) DeleteInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) error {
	const query = `
DELETE FROM info_objects io
USING decks d
WHERE io.deck_id = d.id AND io.id = $1 AND d.user_id = $2
`

	result, err := r.db.Exec(ctx, query, objectID, userID)
	if err != nil {
		return fmt.Errorf("delete info object: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrInfoObjectNotFound
	}

	return nil
}

func (r *Repository) CreateCard(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req CreateCardRequest) (*Card, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create card tx: %w", err)
	}
	defer tx.Rollback(ctx)

	answersJSON, err := json.Marshal(req.CorrectAnswers)
	if err != nil {
		return nil, fmt.Errorf("marshal correct answers: %w", err)
	}
	distractorsJSON, err := json.Marshal(req.Distractors)
	if err != nil {
		return nil, fmt.Errorf("marshal distractors: %w", err)
	}
	highlightJSON, err := json.Marshal(req.HighlightLines)
	if err != nil {
		return nil, fmt.Errorf("marshal highlight lines: %w", err)
	}

	const insertCardQuery = `
INSERT INTO cards (info_object_id, front, card_type, step, correct_answers, distractors, highlight_lines)
SELECT io.id, $3, $4, $5, $6, $7, $8
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $1 AND d.user_id = $2
RETURNING id, info_object_id, front, card_type, step, correct_answers, distractors, highlight_lines, created_at, updated_at
`

	var created Card
	var rawAnswers []byte
	var rawDistractors []byte
	var rawHighlight []byte
	err = tx.QueryRow(ctx, insertCardQuery, objectID, userID, req.Front, req.CardType, req.Step, answersJSON, distractorsJSON, highlightJSON).Scan(
		&created.ID,
		&created.InfoObjectID,
		&created.Front,
		&created.CardType,
		&created.Step,
		&rawAnswers,
		&rawDistractors,
		&rawHighlight,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInfoObjectNotFound
		}
		return nil, fmt.Errorf("insert card: %w", err)
	}

	if err := decodeCardJSON(&created, rawAnswers, rawDistractors, rawHighlight); err != nil {
		return nil, err
	}

	const insertStateQuery = `
INSERT INTO card_states (card_id, user_id, due_date, status)
SELECT $1, d.user_id, CASE WHEN $2 = 0 THEN NOW() ELSE NULL END, CASE WHEN $2 = 0 THEN 'new' ELSE 'locked' END
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.id = $3
ON CONFLICT (card_id, user_id) DO NOTHING
`

	if _, err := tx.Exec(ctx, insertStateQuery, created.ID, created.Step, objectID); err != nil {
		return nil, fmt.Errorf("insert initial card state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create card tx: %w", err)
	}

	return &created, nil
}

func (r *Repository) UpdateCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, req UpdateCardRequest) (*Card, error) {
	answersJSON, err := json.Marshal(req.CorrectAnswers)
	if err != nil {
		return nil, fmt.Errorf("marshal correct answers: %w", err)
	}
	distractorsJSON, err := json.Marshal(req.Distractors)
	if err != nil {
		return nil, fmt.Errorf("marshal distractors: %w", err)
	}
	highlightJSON, err := json.Marshal(req.HighlightLines)
	if err != nil {
		return nil, fmt.Errorf("marshal highlight lines: %w", err)
	}

	const query = `
UPDATE cards c
SET front = $3, card_type = $4, step = $5, correct_answers = $6, distractors = $7, highlight_lines = $8, updated_at = NOW()
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE c.info_object_id = io.id AND c.id = $1 AND d.user_id = $2
RETURNING c.id, c.info_object_id, c.front, c.card_type, c.step, c.correct_answers, c.distractors, c.highlight_lines, c.created_at, c.updated_at
`

	var updated Card
	var rawAnswers []byte
	var rawDistractors []byte
	var rawHighlight []byte
	err = r.db.QueryRow(ctx, query, cardID, userID, req.Front, req.CardType, req.Step, answersJSON, distractorsJSON, highlightJSON).Scan(
		&updated.ID,
		&updated.InfoObjectID,
		&updated.Front,
		&updated.CardType,
		&updated.Step,
		&rawAnswers,
		&rawDistractors,
		&rawHighlight,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("update card: %w", err)
	}

	if err := decodeCardJSON(&updated, rawAnswers, rawDistractors, rawHighlight); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (r *Repository) DeleteCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) error {
	const query = `
DELETE FROM cards c
USING info_objects io, decks d
WHERE c.info_object_id = io.id AND io.deck_id = d.id AND c.id = $1 AND d.user_id = $2
`

	result, err := r.db.Exec(ctx, query, cardID, userID)
	if err != nil {
		return fmt.Errorf("delete card: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCardNotFound
	}

	return nil
}

func scanCard(rows pgx.Rows) (Card, error) {
	var item Card
	var rawAnswers []byte
	var rawDistractors []byte
	var rawHighlight []byte
	if err := rows.Scan(&item.ID, &item.InfoObjectID, &item.Front, &item.CardType, &item.Step, &rawAnswers, &rawDistractors, &rawHighlight, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Card{}, fmt.Errorf("scan card: %w", err)
	}
	if err := decodeCardJSON(&item, rawAnswers, rawDistractors, rawHighlight); err != nil {
		return Card{}, err
	}
	return item, nil
}

func decodeCardJSON(item *Card, rawAnswers []byte, rawDistractors []byte, rawHighlight []byte) error {
	if err := json.Unmarshal(rawAnswers, &item.CorrectAnswers); err != nil {
		return fmt.Errorf("decode correct answers: %w", err)
	}
	if err := json.Unmarshal(rawDistractors, &item.Distractors); err != nil {
		return fmt.Errorf("decode distractors: %w", err)
	}
	if err := json.Unmarshal(rawHighlight, &item.HighlightLines); err != nil {
		return fmt.Errorf("decode highlight lines: %w", err)
	}
	return nil
}
