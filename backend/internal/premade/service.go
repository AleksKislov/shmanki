package premade

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("premade deck not found")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidCategory  = errors.New("category is required")
	ErrInvalidRating    = errors.New("rating must be between 1 and 5")
	ErrAlreadyPublished = errors.New("this deck is already published")
	ErrRateOwnDeck      = errors.New("you cannot rate your own deck")
)

type Service struct {
	repo        *Repository
	adminEmails map[string]struct{}
	getUserMail func(context.Context, uuid.UUID) (string, error)
}

func NewService(repo *Repository, adminEmails map[string]struct{}, getUserMail func(context.Context, uuid.UUID) (string, error)) *Service {
	return &Service{repo: repo, adminEmails: adminEmails, getUserMail: getUserMail}
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]Deck, error) {
	return s.repo.List(ctx, filters)
}
func (s *Service) Categories(ctx context.Context) ([]string, error) { return s.repo.Categories(ctx) }
func (s *Service) Get(ctx context.Context, deckID uuid.UUID, userID uuid.UUID) (*DeckDetail, error) {
	return s.repo.GetDetail(ctx, deckID, userID)
}

func (s *Service) Publish(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req PublishDeckRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Category) == "" {
		return uuid.Nil, ErrInvalidCategory
	}
	return s.repo.PublishDeck(ctx, userID, deckID, req.Title, req.Description, req.Category)
}

func (s *Service) Clone(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID) (uuid.UUID, error) {
	return s.repo.CloneToUser(ctx, userID, premadeID)
}

func (s *Service) Rate(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID, req RateDeckRequest) error {
	if req.Score < 1 || req.Score > 5 {
		return ErrInvalidRating
	}
	owner, err := s.repo.IsOwner(ctx, premadeID, userID)
	if err != nil {
		return err
	}
	if owner {
		return ErrRateOwnDeck
	}
	return s.repo.Rate(ctx, userID, premadeID, req.Score)
}

func (s *Service) RemoveRating(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID) error {
	return s.repo.RemoveRating(ctx, userID, premadeID)
}

func (s *Service) Delete(ctx context.Context, userID uuid.UUID, premadeID uuid.UUID) error {
	email, err := s.getUserMail(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	_, isAdmin := s.adminEmails[strings.ToLower(strings.TrimSpace(email))]
	return s.repo.Delete(ctx, userID, premadeID, isAdmin)
}

func (s *Service) AdminList(ctx context.Context, source string) ([]Deck, error) {
	return s.repo.ListAdmin(ctx, source)
}

func (s *Service) AdminSetPublished(ctx context.Context, premadeID uuid.UUID, isPublished bool) error {
	return s.repo.SetPublished(ctx, premadeID, isPublished)
}

func (s *Service) AdminCreateOfficialFromDeck(ctx context.Context, req CreateOfficialFromDeckRequest) (uuid.UUID, error) {
	if strings.TrimSpace(req.Category) == "" {
		return uuid.Nil, ErrInvalidCategory
	}
	deckID, err := uuid.Parse(req.DeckID)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	return s.repo.CreateOfficialFromDeck(ctx, deckID, req.Title, req.Description, req.Category)
}
