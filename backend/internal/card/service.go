package card

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInfoObjectNotFound = errors.New("info object not found")
	ErrCardNotFound       = errors.New("card not found")
	ErrInvalidInfoObject  = errors.New("info object title is required")
	ErrInvalidCard        = errors.New("card front and correct answers are required")
)

type store interface {
	ListInfoObjectsByDeckID(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) ([]InfoObject, error)
	CreateInfoObject(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req CreateInfoObjectRequest) (*InfoObject, error)
	GetInfoObjectByID(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) (*InfoObjectDetail, error)
	UpdateInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req UpdateInfoObjectRequest) (*InfoObject, error)
	DeleteInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) error
	CreateCard(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req CreateCardRequest) (*Card, error)
	UpdateCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, req UpdateCardRequest) (*Card, error)
	DeleteCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) error
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
}

func (s *Service) ListInfoObjects(ctx context.Context, userID uuid.UUID, deckID uuid.UUID) ([]InfoObject, error) {
	return s.store.ListInfoObjectsByDeckID(ctx, userID, deckID)
}

func (s *Service) CreateInfoObject(ctx context.Context, userID uuid.UUID, deckID uuid.UUID, req CreateInfoObjectRequest) (*InfoObject, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrInvalidInfoObject
	}
	if req.Discipline == "" {
		req.Discipline = "programming"
	}
	if req.ContentType == "" {
		req.ContentType = "text"
	}

	return s.store.CreateInfoObject(ctx, userID, deckID, req)
}

func (s *Service) GetInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) (*InfoObjectDetail, error) {
	return s.store.GetInfoObjectByID(ctx, userID, objectID)
}

func (s *Service) UpdateInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req UpdateInfoObjectRequest) (*InfoObject, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrInvalidInfoObject
	}
	if req.Discipline == "" {
		req.Discipline = "programming"
	}
	if req.ContentType == "" {
		req.ContentType = "text"
	}

	return s.store.UpdateInfoObject(ctx, userID, objectID, req)
}

func (s *Service) DeleteInfoObject(ctx context.Context, userID uuid.UUID, objectID uuid.UUID) error {
	return s.store.DeleteInfoObject(ctx, userID, objectID)
}

func (s *Service) CreateCard(ctx context.Context, userID uuid.UUID, objectID uuid.UUID, req CreateCardRequest) (*Card, error) {
	if strings.TrimSpace(req.Front) == "" || len(req.CorrectAnswers) == 0 {
		return nil, ErrInvalidCard
	}
	return s.store.CreateCard(ctx, userID, objectID, req)
}

func (s *Service) UpdateCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID, req UpdateCardRequest) (*Card, error) {
	if strings.TrimSpace(req.Front) == "" || len(req.CorrectAnswers) == 0 {
		return nil, ErrInvalidCard
	}
	return s.store.UpdateCard(ctx, userID, cardID, req)
}

func (s *Service) DeleteCard(ctx context.Context, userID uuid.UUID, cardID uuid.UUID) error {
	return s.store.DeleteCard(ctx, userID, cardID)
}
