package premade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shmanki/internal/card"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, filters ListFilters) ([]Deck, error) {
	query := `
SELECT pd.id, pd.user_id, pd.source, pd.source_deck_id, pd.title, pd.description, pd.language_code, pd.category,
       pd.is_published, pd.rating_avg, pd.rating_count, COALESCE(NULLIF(u.display_name, ''), split_part(u.email, '@', 1), '') AS author_name,
       pd.created_at, pd.updated_at
FROM premade_decks pd
LEFT JOIN users u ON u.id = pd.user_id
WHERE pd.is_published = true`

	args := make([]any, 0)
	if filters.Source == string(SourceOfficial) || filters.Source == string(SourceCommunity) {
		args = append(args, filters.Source)
		query += fmt.Sprintf(" AND pd.source = $%d", len(args))
	}
	if strings.TrimSpace(filters.Category) != "" {
		args = append(args, strings.TrimSpace(filters.Category))
		query += fmt.Sprintf(" AND pd.category = $%d", len(args))
	}
	if strings.TrimSpace(filters.Language) != "" {
		args = append(args, strings.TrimSpace(filters.Language))
		query += fmt.Sprintf(" AND pd.language_code = $%d", len(args))
	}
	if filters.MinRating > 0 {
		args = append(args, filters.MinRating)
		query += fmt.Sprintf(" AND pd.rating_avg >= $%d", len(args))
	}

	switch filters.Sort {
	case "newest":
		query += " ORDER BY pd.created_at DESC"
	case "popular":
		query += " ORDER BY pd.rating_count DESC, pd.rating_avg DESC"
	default:
		query += " ORDER BY pd.rating_avg DESC, pd.rating_count DESC, pd.created_at DESC"
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list premade decks: %w", err)
	}
	defer rows.Close()

	items := make([]Deck, 0)
	for rows.Next() {
		var item Deck
		if err := rows.Scan(&item.ID, &item.UserID, &item.Source, &item.SourceDeckID, &item.Title, &item.Description, &item.LanguageCode, &item.Category,
			&item.IsPublished, &item.RatingAvg, &item.RatingCount, &item.AuthorName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan premade deck: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) ListAdmin(ctx context.Context, source string) ([]Deck, error) {
	query := `
SELECT pd.id, pd.user_id, pd.source, pd.source_deck_id, pd.title, pd.description, pd.language_code, pd.category,
       pd.is_published, pd.rating_avg, pd.rating_count, COALESCE(NULLIF(u.display_name, ''), split_part(u.email, '@', 1), '') AS author_name,
       pd.created_at, pd.updated_at
FROM premade_decks pd
LEFT JOIN users u ON u.id = pd.user_id
WHERE 1=1`
	args := make([]any, 0)
	if source == string(SourceOfficial) || source == string(SourceCommunity) {
		args = append(args, source)
		query += fmt.Sprintf(" AND pd.source = $%d", len(args))
	}
	query += " ORDER BY pd.created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin premade decks: %w", err)
	}
	defer rows.Close()

	items := make([]Deck, 0)
	for rows.Next() {
		var item Deck
		if err := rows.Scan(&item.ID, &item.UserID, &item.Source, &item.SourceDeckID, &item.Title, &item.Description, &item.LanguageCode, &item.Category,
			&item.IsPublished, &item.RatingAvg, &item.RatingCount, &item.AuthorName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan premade deck: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) Categories(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT category FROM premade_decks WHERE is_published = true ORDER BY category ASC`)
	if err != nil {
		return nil, fmt.Errorf("list premade categories: %w", err)
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		items = append(items, category)
	}
	return items, rows.Err()
}

func (r *Repository) GetDetail(ctx context.Context, deckID uuid.UUID, userID uuid.UUID) (*DeckDetail, error) {
	const deckQuery = `
SELECT pd.id, pd.user_id, pd.source, pd.source_deck_id, pd.title, pd.description, pd.language_code, pd.category,
       pd.is_published, pd.rating_avg, pd.rating_count, COALESCE(NULLIF(u.display_name, ''), split_part(u.email, '@', 1), '') AS author_name,
       pd.created_at, pd.updated_at,
       (SELECT score FROM premade_deck_ratings WHERE premade_deck_id = pd.id AND user_id = $2)
FROM premade_decks pd
LEFT JOIN users u ON u.id = pd.user_id
WHERE pd.id = $1 AND (pd.is_published = true OR pd.user_id = $2)
`

	var detail DeckDetail
	var myRating *int
	err := r.db.QueryRow(ctx, deckQuery, deckID, userID).Scan(&detail.ID, &detail.UserID, &detail.Source, &detail.SourceDeckID, &detail.Title, &detail.Description,
		&detail.LanguageCode, &detail.Category, &detail.IsPublished, &detail.RatingAvg, &detail.RatingCount, &detail.AuthorName, &detail.CreatedAt, &detail.UpdatedAt, &myRating)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get premade deck: %w", err)
	}
	detail.MyRating = myRating

	const objectsQuery = `
SELECT id, premade_deck_id, title, content, discipline, content_type
FROM premade_info_objects
WHERE premade_deck_id = $1
ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, objectsQuery, deckID)
	if err != nil {
		return nil, fmt.Errorf("list premade objects: %w", err)
	}
	defer rows.Close()

	objects := make([]InfoObject, 0)
	objIDs := make([]uuid.UUID, 0)
	objMap := make(map[uuid.UUID]int)
	for rows.Next() {
		var item InfoObject
		if err := rows.Scan(&item.ID, &item.DeckID, &item.Title, &item.Content, &item.Discipline, &item.ContentType); err != nil {
			return nil, fmt.Errorf("scan premade object: %w", err)
		}
		objMap[item.ID] = len(objects)
		objIDs = append(objIDs, item.ID)
		objects = append(objects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(objIDs) > 0 {
		const cardsQuery = `
SELECT id, premade_info_object_id, front, card_type, step, correct_answers, distractors, created_at, updated_at
FROM premade_cards
WHERE premade_info_object_id = ANY($1)
ORDER BY step ASC, created_at ASC`
		cardsRows, err := r.db.Query(ctx, cardsQuery, objIDs)
		if err != nil {
			return nil, fmt.Errorf("list premade cards: %w", err)
		}
		defer cardsRows.Close()

		for cardsRows.Next() {
			var cardItem card.Card
			var rawAnswers []byte
			var rawDistractors []byte
			if err := cardsRows.Scan(&cardItem.ID, &cardItem.InfoObjectID, &cardItem.Front, &cardItem.CardType, &cardItem.Step,
				&rawAnswers, &rawDistractors, &cardItem.CreatedAt, &cardItem.UpdatedAt); err != nil {
				return nil, fmt.Errorf("scan premade card: %w", err)
			}
			if err := json.Unmarshal(rawAnswers, &cardItem.CorrectAnswers); err != nil {
				return nil, fmt.Errorf("decode premade card answers: %w", err)
			}
			if err := json.Unmarshal(rawDistractors, &cardItem.Distractors); err != nil {
				return nil, fmt.Errorf("decode premade card distractors: %w", err)
			}
			index := objMap[cardItem.InfoObjectID]
			objects[index].Cards = append(objects[index].Cards, cardItem)
		}
	}

	detail.InfoObjects = objects
	return &detail, nil
}

func (r *Repository) PublishDeck(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, title string, description string, category string) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin publish tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var premadeID uuid.UUID
	const createQuery = `
INSERT INTO premade_decks (user_id, source, source_deck_id, title, description, language_code, category, is_published)
SELECT $1, 'community', d.id, COALESCE(NULLIF($3, ''), d.title), COALESCE(NULLIF($4, ''), d.description), d.language_code, $5, true
FROM decks d
WHERE d.id = $2 AND d.user_id = $1
RETURNING id`
	err = tx.QueryRow(ctx, createQuery, userID, deckID, strings.TrimSpace(title), strings.TrimSpace(description), strings.TrimSpace(category)).Scan(&premadeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrForbidden
		}
		if strings.Contains(err.Error(), "uq_premade_decks_source_deck_id") {
			return uuid.Nil, ErrAlreadyPublished
		}
		return uuid.Nil, fmt.Errorf("create premade deck: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO premade_info_objects (premade_deck_id, title, content, discipline, content_type)
SELECT $1, title, content, discipline, content_type
FROM info_objects
WHERE deck_id = $2
ORDER BY created_at ASC`, premadeID, deckID); err != nil {
		return uuid.Nil, fmt.Errorf("copy premade info objects: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO premade_cards (premade_info_object_id, front, step, correct_answers, distractors, card_type)
SELECT pio.id, c.front, c.step, c.correct_answers, c.distractors, c.card_type
FROM info_objects io
JOIN premade_info_objects pio ON pio.premade_deck_id = $1 AND pio.title = io.title AND pio.content = io.content
JOIN cards c ON c.info_object_id = io.id
WHERE io.deck_id = $2`, premadeID, deckID); err != nil {
		return uuid.Nil, fmt.Errorf("copy premade cards: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit publish tx: %w", err)
	}

	return premadeID, nil
}

func (r *Repository) CreateOfficialFromDeck(ctx context.Context, deckID uuid.UUID, title string, description string, category string) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin create official tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var premadeID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO premade_decks (user_id, source, source_deck_id, title, description, language_code, category, is_published)
SELECT NULL, 'official', NULL, COALESCE(NULLIF($2, ''), d.title), COALESCE(NULLIF($3, ''), d.description), d.language_code, $4, true
FROM decks d
WHERE d.id = $1
RETURNING id`, deckID, strings.TrimSpace(title), strings.TrimSpace(description), strings.TrimSpace(category)).Scan(&premadeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("insert official premade deck: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO premade_info_objects (premade_deck_id, title, content, discipline, content_type)
SELECT $1, title, content, discipline, content_type
FROM info_objects
WHERE deck_id = $2
ORDER BY created_at ASC`, premadeID, deckID); err != nil {
		return uuid.Nil, fmt.Errorf("copy official info objects: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO premade_cards (premade_info_object_id, front, step, correct_answers, distractors, card_type)
SELECT pio.id, c.front, c.step, c.correct_answers, c.distractors, c.card_type
FROM info_objects io
JOIN premade_info_objects pio ON pio.premade_deck_id = $1 AND pio.title = io.title AND pio.content = io.content
JOIN cards c ON c.info_object_id = io.id
WHERE io.deck_id = $2`, premadeID, deckID); err != nil {
		return uuid.Nil, fmt.Errorf("copy official cards: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit create official tx: %w", err)
	}

	return premadeID, nil
}

func (r *Repository) SetPublished(ctx context.Context, premadeID uuid.UUID, isPublished bool) error {
	result, err := r.db.Exec(ctx, `UPDATE premade_decks SET is_published = $2, updated_at = NOW() WHERE id = $1`, premadeID, isPublished)
	if err != nil {
		return fmt.Errorf("set published: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CloneToUser(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var newDeckID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO decks (user_id, title, description, language_code)
SELECT $1, title, description, language_code
FROM premade_decks
WHERE id = $2 AND is_published = true
RETURNING id`, userID, premadeID).Scan(&newDeckID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("clone deck: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO info_objects (deck_id, title, content, discipline, content_type)
SELECT $1, title, content, discipline, content_type
FROM premade_info_objects
WHERE premade_deck_id = $2
ORDER BY created_at ASC`, newDeckID, premadeID); err != nil {
		return uuid.Nil, fmt.Errorf("clone info objects: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO cards (info_object_id, front, step, correct_answers, distractors, card_type)
SELECT io.id, pc.front, pc.step, pc.correct_answers, pc.distractors, pc.card_type
FROM premade_info_objects pio
JOIN info_objects io ON io.deck_id = $1 AND io.title = pio.title AND io.content = pio.content
JOIN premade_cards pc ON pc.premade_info_object_id = pio.id
WHERE pio.premade_deck_id = $2`, newDeckID, premadeID); err != nil {
		return uuid.Nil, fmt.Errorf("clone cards: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}

	return newDeckID, nil
}

func (r *Repository) Rate(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID, score int) error {
	_, err := r.db.Exec(ctx, `
INSERT INTO premade_deck_ratings (premade_deck_id, user_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (premade_deck_id, user_id)
DO UPDATE SET score = EXCLUDED.score, updated_at = NOW()`, premadeID, userID, score)
	if err != nil {
		return fmt.Errorf("rate premade deck: %w", err)
	}
	return r.recomputeRatings(ctx, premadeID)
}

func (r *Repository) RemoveRating(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM premade_deck_ratings WHERE premade_deck_id = $1 AND user_id = $2`, premadeID, userID); err != nil {
		return fmt.Errorf("remove premade rating: %w", err)
	}
	return r.recomputeRatings(ctx, premadeID)
}

func (r *Repository) Delete(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID, isAdmin bool) error {
	query := `DELETE FROM premade_decks WHERE id = $1 AND source = 'community' AND user_id = $2`
	args := []any{premadeID, userID}
	if isAdmin {
		query = `DELETE FROM premade_decks WHERE id = $1`
		args = []any{premadeID}
	}
	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete premade deck: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

func (r *Repository) IsOwner(ctx context.Context, premadeID uuid.UUID, userID uuid.UUID) (bool, error) {
	var ownerID *uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT user_id FROM premade_decks WHERE id = $1`, premadeID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}
	if ownerID == nil {
		return false, nil
	}
	return *ownerID == userID, nil
}

func (r *Repository) recomputeRatings(ctx context.Context, premadeID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
UPDATE premade_decks pd
SET rating_avg = COALESCE(stats.avg_score, 0),
    rating_count = COALESCE(stats.cnt, 0),
    updated_at = NOW()
FROM (
  SELECT premade_deck_id, AVG(score)::numeric(3,2) AS avg_score, COUNT(*)::int AS cnt
  FROM premade_deck_ratings
  WHERE premade_deck_id = $1
  GROUP BY premade_deck_id
) stats
WHERE pd.id = $1`, premadeID)
	if err != nil {
		return fmt.Errorf("recompute ratings: %w", err)
	}
	_, err = r.db.Exec(ctx, `
UPDATE premade_decks
SET rating_avg = 0, rating_count = 0, updated_at = NOW()
WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM premade_deck_ratings WHERE premade_deck_id = $1)`, premadeID)
	if err != nil {
		return fmt.Errorf("reset ratings: %w", err)
	}
	return nil
}
