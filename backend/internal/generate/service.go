package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	cardpkg "shmanki/internal/card"
	"shmanki/internal/platform/language"
)

var (
	ErrGenerationUnavailable = errors.New("generation provider is not configured")
	ErrInvalidRequest        = errors.New("generation prompt and deck id are required")
	ErrInvalidSuggestion     = errors.New("generated content is invalid")
	ErrDeckNotFound          = errors.New("deck not found")
)

type store interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	GetDeckLanguage(ctx context.Context, tx pgx.Tx, userID uuid.UUID, deckID uuid.UUID) (string, error)
	InsertGenerationLog(ctx context.Context, tx pgx.Tx, log generationLog) (*generationLog, error)
	CreateInfoObject(ctx context.Context, tx pgx.Tx, userID uuid.UUID, deckID uuid.UUID, object SuggestedObject) (*SavedObject, error)
	CreateCard(ctx context.Context, tx pgx.Tx, userID uuid.UUID, infoObjectID uuid.UUID, card SuggestedCard) (*SavedCard, error)
}

type Service struct {
	store           store
	client          completionClient
	defaultLanguage string
}

func NewService(store store, client completionClient, defaultLanguage string) *Service {
	return &Service{store: store, client: client, defaultLanguage: defaultLanguage}
}

func (s *Service) Suggest(ctx context.Context, userID uuid.UUID, req SuggestRequest) (*SuggestResponse, error) {
	if req.DeckID == uuid.Nil || strings.TrimSpace(req.Prompt) == "" {
		return nil, ErrInvalidRequest
	}
	if s.client == nil {
		return nil, ErrGenerationUnavailable
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	languageCode, err := s.store.GetDeckLanguage(ctx, tx, userID, req.DeckID)
	if err != nil {
		return nil, err
	}
	languageCode, err = language.Normalize(languageCode, s.defaultLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalize deck language: %w", err)
	}

	result, err := s.client.Complete(ctx, llmCompletionRequest{
		Prompt:       strings.TrimSpace(req.Prompt),
		LanguageCode: languageCode,
		Discipline:   strings.TrimSpace(req.Discipline),
		ContentType:  strings.TrimSpace(req.ContentType),
	})
	if err != nil {
		return nil, err
	}
	if err := validateSuggestedObjects(result.Objects); err != nil {
		return nil, err
	}

	rawObjects, err := json.Marshal(result.Objects)
	if err != nil {
		return nil, fmt.Errorf("marshal generated objects: %w", err)
	}

	logRecord, err := s.store.InsertGenerationLog(ctx, tx, generationLog{
		DeckID:     req.DeckID,
		UserID:     userID,
		Prompt:     strings.TrimSpace(req.Prompt),
		Provider:   result.Provider,
		Model:      result.Model,
		ObjectsRaw: rawObjects,
		CardsCount: totalCards(result.Objects),
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit generation suggestion tx: %w", err)
	}

	return &SuggestResponse{
		GenerationID: &logRecord.ID,
		Model:        result.Model,
		InfoObjects:  result.Objects,
	}, nil
}

func (s *Service) Edit(ctx context.Context, userID uuid.UUID, req EditRequest) (*SuggestResponse, error) {
	if req.DeckID == uuid.Nil || strings.TrimSpace(req.Prompt) == "" || len(req.InfoObjects) == 0 {
		return nil, ErrInvalidRequest
	}
	if s.client == nil {
		return nil, ErrGenerationUnavailable
	}
	if err := validateSuggestedObjects(req.InfoObjects); err != nil {
		return nil, err
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	languageCode, err := s.store.GetDeckLanguage(ctx, tx, userID, req.DeckID)
	if err != nil {
		return nil, err
	}
	languageCode, err = language.Normalize(languageCode, s.defaultLanguage)
	if err != nil {
		return nil, fmt.Errorf("normalize deck language: %w", err)
	}

	result, err := s.client.Complete(ctx, llmCompletionRequest{
		Prompt:        strings.TrimSpace(req.Prompt),
		LanguageCode:  languageCode,
		Discipline:    strings.TrimSpace(req.Discipline),
		ContentType:   strings.TrimSpace(req.ContentType),
		ExistingDraft: req.InfoObjects,
	})
	if err != nil {
		return nil, err
	}
	if err := validateSuggestedObjects(result.Objects); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit generation edit tx: %w", err)
	}

	return &SuggestResponse{
		GenerationID: req.GenerationID,
		Model:        result.Model,
		InfoObjects:  result.Objects,
	}, nil
}

func (s *Service) Save(ctx context.Context, userID uuid.UUID, req SaveRequest) (*SaveResponse, error) {
	if req.DeckID == uuid.Nil || len(req.InfoObjects) == 0 || strings.TrimSpace(req.Prompt) == "" {
		return nil, ErrInvalidRequest
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, ErrInvalidRequest
	}
	if err := validateSuggestedObjects(req.InfoObjects); err != nil {
		return nil, err
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := s.store.GetDeckLanguage(ctx, tx, userID, req.DeckID); err != nil {
		return nil, err
	}

	savedObjects := make([]SavedObject, 0, len(req.InfoObjects))
	for _, object := range req.InfoObjects {
		savedObject, err := s.store.CreateInfoObject(ctx, tx, userID, req.DeckID, object)
		if err != nil {
			return nil, err
		}

		savedObject.Cards = make([]SavedCard, 0, len(object.Cards))
		for _, item := range object.Cards {
			savedCard, err := s.store.CreateCard(ctx, tx, userID, savedObject.ID, item)
			if err != nil {
				return nil, err
			}
			savedObject.Cards = append(savedObject.Cards, *savedCard)
		}

		savedObjects = append(savedObjects, *savedObject)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit generation save tx: %w", err)
	}

	return &SaveResponse{InfoObjects: savedObjects}, nil
}

func validateSuggestedObjects(items []SuggestedObject) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: at least one info object is required", ErrInvalidSuggestion)
	}

	for _, object := range items {
		if strings.TrimSpace(object.Title) == "" || strings.TrimSpace(object.Content) == "" {
			return fmt.Errorf("%w: info object title and content are required", ErrInvalidSuggestion)
		}
		if strings.TrimSpace(object.Discipline) == "" || strings.TrimSpace(object.ContentType) == "" {
			return fmt.Errorf("%w: info object discipline and content type are required", ErrInvalidSuggestion)
		}
		if len(object.Cards) == 0 {
			return fmt.Errorf("%w: each info object must contain at least one card", ErrInvalidSuggestion)
		}

		hasFoundationalCard := false
		hasTraceCard := false
		hasLineOrderCard := false
		hasReconstructionCard := false
		for _, card := range object.Cards {
			if strings.TrimSpace(card.Front) == "" || len(card.CorrectAnswers) == 0 || !card.CardType.Valid() {
				return fmt.Errorf("%w: card front and correct answers are required", ErrInvalidSuggestion)
			}
			for _, answer := range card.CorrectAnswers {
				if len(answer) == 0 {
					return fmt.Errorf("%w: correct answers cannot contain empty token sequences", ErrInvalidSuggestion)
				}
			}
			for _, line := range card.HighlightLines {
				if line <= 0 || line > len(strings.Split(object.Content, "\n")) {
					return fmt.Errorf("%w: highlight lines must point to existing content lines", ErrInvalidSuggestion)
				}
			}

			switch card.CardType {
			case cardpkg.CardTypeConcept, cardpkg.CardTypeSignature:
				if card.Step == 0 {
					hasFoundationalCard = true
				}
			case cardpkg.CardTypeTrace:
				if card.Step >= 1 {
					hasTraceCard = true
				}
			case cardpkg.CardTypeLineOrder:
				if card.Step >= 2 {
					hasLineOrderCard = true
				}
			case cardpkg.CardTypeBlockOrder, cardpkg.CardTypeChooseSnippet, cardpkg.CardTypeFixBug:
				if card.Step >= 3 {
					hasReconstructionCard = true
				}
			}
		}

		if strings.HasPrefix(object.ContentType, "code_") {
			if !hasFoundationalCard || !hasTraceCard || !hasLineOrderCard || !hasReconstructionCard {
				return fmt.Errorf("%w: code info objects must follow the progression with foundational, trace, line-order, and reconstruction cards", ErrInvalidSuggestion)
			}
		}
	}

	return nil
}

func totalCards(items []SuggestedObject) int {
	total := 0
	for _, item := range items {
		total += len(item.Cards)
	}
	return total
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
