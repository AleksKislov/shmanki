package generate

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrGenerationUnavailable = errors.New("generation endpoint is not configured yet")

type Service struct {
	apiKey string
	model  string
}

func NewService(apiKey string, model string) *Service {
	return &Service{apiKey: apiKey, model: model}
}

func (s *Service) Generate(ctx context.Context, userID uuid.UUID, req Request) (*Response, error) {
	_ = ctx
	_ = userID
	_ = req
	_ = s.model
	if s.apiKey == "" {
		return nil, ErrGenerationUnavailable
	}
	return nil, ErrGenerationUnavailable
}
