package deck

import (
	"context"
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

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]Deck, error) {
	const query = `
SELECT id, title, description, language_code, created_at, updated_at
FROM decks
WHERE user_id = $1
ORDER BY created_at DESC
`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list decks: %w", err)
	}
	defer rows.Close()

	decks := make([]Deck, 0)
	for rows.Next() {
		var item Deck
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.LanguageCode, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan deck: %w", err)
		}
		decks = append(decks, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decks: %w", err)
	}

	return decks, nil
}

func (r *Repository) GetByID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*Deck, error) {
	const query = `
SELECT id, title, description, language_code, created_at, updated_at
FROM decks
WHERE id = $1 AND user_id = $2
`

	var item Deck
	err := r.db.QueryRow(ctx, query, deckID, userID).Scan(&item.ID, &item.Title, &item.Description, &item.LanguageCode, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeckNotFound
		}
		return nil, fmt.Errorf("get deck: %w", err)
	}

	return &item, nil
}

func (r *Repository) ListInfoObjectsByDeckID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) ([]InfoObjectSummary, error) {
	const query = `
SELECT io.id, io.title, io.discipline, io.content_type
FROM info_objects io
JOIN decks d ON d.id = io.deck_id
WHERE io.deck_id = $1 AND d.user_id = $2
ORDER BY io.created_at ASC
`

	rows, err := r.db.Query(ctx, query, deckID, userID)
	if err != nil {
		return nil, fmt.Errorf("list info objects by deck: %w", err)
	}
	defer rows.Close()

	objects := make([]InfoObjectSummary, 0)
	for rows.Next() {
		var item InfoObjectSummary
		if err := rows.Scan(&item.ID, &item.Title, &item.Discipline, &item.ContentType); err != nil {
			return nil, fmt.Errorf("scan info object summary: %w", err)
		}
		objects = append(objects, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate info object summaries: %w", err)
	}

	return objects, nil
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, req CreateDeckRequest, languageCode string) (*Deck, error) {
	const query = `
INSERT INTO decks (user_id, title, description, language_code)
VALUES ($1, $2, $3, $4)
RETURNING id, title, description, language_code, created_at, updated_at
`

	var item Deck
	err := r.db.QueryRow(ctx, query, userID, req.Title, req.Description, languageCode).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.LanguageCode,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create deck: %w", err)
	}

	return &item, nil
}

func (r *Repository) Update(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req UpdateDeckRequest, languageCode string) (*Deck, error) {
	const query = `
UPDATE decks
SET title = $3, description = $4, language_code = $5, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING id, title, description, language_code, created_at, updated_at
`

	var item Deck
	err := r.db.QueryRow(ctx, query, deckID, userID, req.Title, req.Description, languageCode).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.LanguageCode,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeckNotFound
		}
		return nil, fmt.Errorf("update deck: %w", err)
	}

	return &item, nil
}

func (r *Repository) Delete(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) error {
	const query = `DELETE FROM decks WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, deckID, userID)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrDeckNotFound
	}

	return nil
}
