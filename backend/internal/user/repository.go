package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type CreateParams struct {
	Email             string
	PasswordHash      string
	PreferredLanguage string
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (*User, error) {
	const query = `
INSERT INTO users (email, password_hash, preferred_language)
VALUES ($1, $2, $3)
RETURNING id, email, preferred_language
`

	var user User
	err := r.db.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(params.Email)), params.PasswordHash, params.PreferredLanguage).Scan(
		&user.ID,
		&user.Email,
		&user.PreferredLanguage,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, string, error) {
	const query = `
SELECT id, email, password_hash, preferred_language
FROM users
WHERE email = $1
`

	var user User
	var passwordHash string
	err := r.db.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&user.ID,
		&user.Email,
		&passwordHash,
		&user.PreferredLanguage,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("get user by email: %w", err)
	}

	return &user, passwordHash, nil
}

func (r *Repository) GetByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	const query = `
SELECT id, email, preferred_language
FROM users
WHERE id = $1
`

	var user User
	err := r.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Email, &user.PreferredLanguage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *Repository) GetPreferredLanguage(ctx context.Context, userID uuid.UUID) (string, error) {
	const query = `SELECT preferred_language FROM users WHERE id = $1`

	var preferredLanguage string
	err := r.db.QueryRow(ctx, query, userID).Scan(&preferredLanguage)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("get preferred language: %w", err)
	}

	return preferredLanguage, nil
}
