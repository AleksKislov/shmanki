package deck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"shmanki/internal/platform/language"
)

var (
	ErrDeckNotFound = errors.New("deck not found")
	ErrInvalidDeck  = errors.New("deck title is required")
	ErrInvalidLang  = errors.New("invalid language code")
)

type store interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]Deck, error)
	GetByID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*Deck, error)
	ListInfoObjectsByDeckID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) ([]InfoObjectSummary, error)
	Create(ctx context.Context, userID uuid.UUID, req CreateDeckRequest, languageCode string) (*Deck, error)
	Update(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req UpdateDeckRequest, languageCode string) (*Deck, error)
	Delete(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) error
}

type userLanguageReader interface {
	GetPreferredLanguage(ctx context.Context, userID uuid.UUID) (string, error)
}

type Service struct {
	decks           store
	users           userLanguageReader
	defaultLanguage string
}

func NewService(decks store, users userLanguageReader, defaultLanguage string) *Service {
	return &Service{decks: decks, users: users, defaultLanguage: defaultLanguage}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Deck, error) {
	return s.decks.ListByUserID(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) (*DeckDetail, error) {
	deckItem, err := s.decks.GetByID(ctx, userID, deckID)
	if err != nil {
		return nil, err
	}

	objects, err := s.decks.ListInfoObjectsByDeckID(ctx, userID, deckID)
	if err != nil {
		return nil, err
	}

	return &DeckDetail{Deck: *deckItem, InfoObjects: objects}, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateDeckRequest) (*Deck, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrInvalidDeck
	}

	languageCode, err := s.resolveLanguage(ctx, userID, req.LanguageCode)
	if err != nil {
		return nil, err
	}

	return s.decks.Create(ctx, userID, req, languageCode)
}

func (s *Service) Update(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req UpdateDeckRequest) (*Deck, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrInvalidDeck
	}

	languageCode, err := s.resolveLanguage(ctx, userID, req.LanguageCode)
	if err != nil {
		return nil, err
	}

	return s.decks.Update(ctx, userID, deckID, req, languageCode)
}

func (s *Service) Delete(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) error {
	return s.decks.Delete(ctx, userID, deckID)
}

func (s *Service) resolveLanguage(ctx context.Context, userID uuid.UUID, requested string) (string, error) {
	fallback := s.defaultLanguage
	if strings.TrimSpace(requested) == "" {
		preferredLanguage, err := s.users.GetPreferredLanguage(ctx, userID)
		if err == nil {
			fallback = preferredLanguage
		}
	}

	normalized, err := language.Normalize(requested, fallback)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidLang, err)
	}

	return normalized, nil
}
