package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shmanki/internal/platform/language"
	"shmanki/internal/platform/token"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidLanguage    = errors.New("invalid language")
)

type creator interface {
	Create(ctx context.Context, params CreateParams) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, string, error)
	UpdatePreferredLanguage(ctx context.Context, userID uuid.UUID, preferredLanguage string) error
}

type Service struct {
	users           creator
	tokens          *token.Manager
	defaultLanguage string
}

func NewService(users creator, tokens *token.Manager, defaultLanguage string) *Service {
	return &Service{users: users, tokens: tokens, defaultLanguage: defaultLanguage}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if !strings.Contains(req.Email, "@") {
		return nil, ErrInvalidEmail
	}
	if len(req.Password) < 8 {
		return nil, ErrInvalidPassword
	}

	preferredLanguage, err := language.Normalize(req.PreferredLanguage, s.defaultLanguage)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidLanguage, err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	createdUser, err := s.users.Create(ctx, CreateParams{
		Email:             req.Email,
		PasswordHash:      string(passwordHash),
		PreferredLanguage: preferredLanguage,
	})
	if err != nil {
		return nil, err
	}

	jwtToken, err := s.tokens.Generate(createdUser.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResponse{Token: jwtToken, User: *createdUser}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	if !strings.Contains(req.Email, "@") {
		return nil, ErrInvalidEmail
	}

	storedUser, passwordHash, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	jwtToken, err := s.tokens.Generate(storedUser.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResponse{Token: jwtToken, User: *storedUser}, nil
}

func (s *Service) UpdatePreferredLanguage(ctx context.Context, userID uuid.UUID, req UpdatePreferredLanguageRequest) error {
	preferredLanguage, err := language.Normalize(req.PreferredLanguage, s.defaultLanguage)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLanguage, err)
	}

	if err := s.users.UpdatePreferredLanguage(ctx, userID, preferredLanguage); err != nil {
		return err
	}

	return nil
}
