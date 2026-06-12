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
	ErrDraftNotFound         = errors.New("generation draft not found")
)

const (
	maxPromptLen      = 4000
	maxDisciplineLen  = 100
	maxContentTypeLen = 50
	maxTitleLen       = 200
	maxContentLen     = 20000
	maxFrontLen       = 2000
	maxDistractorLen  = 2000
	maxAnswerTokenLen = 200
	maxAnswerTokens   = 50
	maxDistractors    = 20
	maxCardsPerObject = 100
	maxObjectsPerResp = 30
)

// hasDisallowedControlChars reports whether s contains control characters
// other than common whitespace (tab, LF, CR).
func hasDisallowedControlChars(s string) bool {
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func validateUserInput(prompt, discipline, contentType string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("%w: prompt is required", ErrInvalidRequest)
	}
	if len(prompt) > maxPromptLen {
		return fmt.Errorf("%w: prompt exceeds %d characters", ErrInvalidRequest, maxPromptLen)
	}
	if len(discipline) > maxDisciplineLen {
		return fmt.Errorf("%w: discipline exceeds %d characters", ErrInvalidRequest, maxDisciplineLen)
	}
	if len(contentType) > maxContentTypeLen {
		return fmt.Errorf("%w: contentType exceeds %d characters", ErrInvalidRequest, maxContentTypeLen)
	}
	for label, val := range map[string]string{"prompt": prompt, "discipline": discipline, "contentType": contentType} {
		if hasDisallowedControlChars(val) {
			return fmt.Errorf("%w: %s contains disallowed control characters", ErrInvalidRequest, label)
		}
	}
	return nil
}

type store interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	GetDeckLanguage(ctx context.Context, tx pgx.Tx, userID uuid.UUID, deckID uuid.UUID) (string, error)
	InsertGenerationLog(ctx context.Context, tx pgx.Tx, log generationLog) (*generationLog, error)
	UpsertGenerationDraft(ctx context.Context, tx pgx.Tx, draft generationDraft) error
	GetGenerationDraft(ctx context.Context, tx pgx.Tx, userID uuid.UUID, generationID uuid.UUID) (*generationDraft, error)
	DeleteGenerationDraft(ctx context.Context, tx pgx.Tx, userID uuid.UUID, generationID uuid.UUID) error
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
	if err := validateUserInput(req.Prompt, req.Discipline, req.ContentType); err != nil {
		return nil, err
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
	deduplicateDistractors(result.Objects)
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

	if err := s.store.UpsertGenerationDraft(ctx, tx, generationDraft{
		GenerationID: logRecord.ID,
		UserID:       userID,
		DeckID:       req.DeckID,
		ObjectsRaw:   rawObjects,
		Model:        result.Model,
	}); err != nil {
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
	if req.DeckID == uuid.Nil || strings.TrimSpace(req.Prompt) == "" || req.GenerationID == nil {
		return nil, ErrInvalidRequest
	}
	if err := validateUserInput(req.Prompt, req.Discipline, req.ContentType); err != nil {
		return nil, err
	}
	if s.client == nil {
		return nil, ErrGenerationUnavailable
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	draft, err := s.store.GetGenerationDraft(ctx, tx, userID, *req.GenerationID)
	if err != nil {
		return nil, err
	}
	if draft.DeckID != req.DeckID {
		return nil, ErrDraftNotFound
	}

	var existingObjects []SuggestedObject
	if err := json.Unmarshal(draft.ObjectsRaw, &existingObjects); err != nil {
		return nil, fmt.Errorf("decode stored draft: %w", err)
	}

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
		ExistingDraft: existingObjects,
	})
	if err != nil {
		return nil, err
	}
	if err := validateSuggestedObjects(result.Objects); err != nil {
		return nil, err
	}

	rawObjects, err := json.Marshal(result.Objects)
	if err != nil {
		return nil, fmt.Errorf("marshal edited objects: %w", err)
	}

	if err := s.store.UpsertGenerationDraft(ctx, tx, generationDraft{
		GenerationID: *req.GenerationID,
		UserID:       userID,
		DeckID:       req.DeckID,
		ObjectsRaw:   rawObjects,
		Model:        result.Model,
	}); err != nil {
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
	if req.DeckID == uuid.Nil || req.GenerationID == nil {
		return nil, ErrInvalidRequest
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	draft, err := s.store.GetGenerationDraft(ctx, tx, userID, *req.GenerationID)
	if err != nil {
		return nil, err
	}
	if draft.DeckID != req.DeckID {
		return nil, ErrDraftNotFound
	}

	var objects []SuggestedObject
	if err := json.Unmarshal(draft.ObjectsRaw, &objects); err != nil {
		return nil, fmt.Errorf("decode stored draft: %w", err)
	}
	deduplicateDistractors(objects)
	if err := validateSuggestedObjects(objects); err != nil {
		return nil, err
	}

	if _, err := s.store.GetDeckLanguage(ctx, tx, userID, req.DeckID); err != nil {
		return nil, err
	}

	savedObjects := make([]SavedObject, 0, len(objects))
	for _, object := range objects {
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

	if err := s.store.DeleteGenerationDraft(ctx, tx, userID, *req.GenerationID); err != nil {
		return nil, err
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
	if len(items) > maxObjectsPerResp {
		return fmt.Errorf("%w: too many info objects (max %d)", ErrInvalidSuggestion, maxObjectsPerResp)
	}

	for _, object := range items {
		if strings.TrimSpace(object.Title) == "" || strings.TrimSpace(object.Content) == "" {
			return fmt.Errorf("%w: info object title and content are required", ErrInvalidSuggestion)
		}
		if strings.TrimSpace(object.Discipline) == "" || strings.TrimSpace(object.ContentType) == "" {
			return fmt.Errorf("%w: info object discipline and content type are required", ErrInvalidSuggestion)
		}
		if len(object.Title) > maxTitleLen {
			return fmt.Errorf("%w: info object title exceeds %d characters", ErrInvalidSuggestion, maxTitleLen)
		}
		if len(object.Content) > maxContentLen {
			return fmt.Errorf("%w: info object content exceeds %d characters", ErrInvalidSuggestion, maxContentLen)
		}
		if len(object.Discipline) > maxDisciplineLen {
			return fmt.Errorf("%w: discipline exceeds %d characters", ErrInvalidSuggestion, maxDisciplineLen)
		}
		if len(object.ContentType) > maxContentTypeLen {
			return fmt.Errorf("%w: contentType exceeds %d characters", ErrInvalidSuggestion, maxContentTypeLen)
		}
		if len(object.Cards) == 0 {
			return fmt.Errorf("%w: each info object must contain at least one card", ErrInvalidSuggestion)
		}
		if len(object.Cards) > maxCardsPerObject {
			return fmt.Errorf("%w: too many cards per info object (max %d)", ErrInvalidSuggestion, maxCardsPerObject)
		}

		hasFoundationalCard := false
		hasTraceCard := false
		hasLineOrderCard := false
		hasReconstructionCard := false
		for _, card := range object.Cards {
			if strings.TrimSpace(card.Front) == "" || len(card.CorrectAnswers) == 0 || !card.CardType.Valid() {
				return fmt.Errorf("%w: card front and correct answers are required", ErrInvalidSuggestion)
			}
			if len(card.Front) > maxFrontLen {
				return fmt.Errorf("%w: card front exceeds %d characters", ErrInvalidSuggestion, maxFrontLen)
			}
			if len(card.Distractors) > maxDistractors {
				return fmt.Errorf("%w: too many distractors (max %d)", ErrInvalidSuggestion, maxDistractors)
			}
			for _, d := range card.Distractors {
				if len(d) > maxDistractorLen {
					return fmt.Errorf("%w: distractor exceeds %d characters", ErrInvalidSuggestion, maxDistractorLen)
				}
			}
			for _, answer := range card.CorrectAnswers {
				if len(answer) == 0 {
					return fmt.Errorf("%w: correct answers cannot contain empty token sequences", ErrInvalidSuggestion)
				}
				if len(answer) > maxAnswerTokens {
					return fmt.Errorf("%w: too many tokens in correct answer (max %d)", ErrInvalidSuggestion, maxAnswerTokens)
				}
				for _, tok := range answer {
					if len(tok) > maxAnswerTokenLen {
						return fmt.Errorf("%w: answer token exceeds %d characters", ErrInvalidSuggestion, maxAnswerTokenLen)
					}
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
			case cardpkg.CardTypeChooseSnippet, cardpkg.CardTypeFixBug:
				if card.Step >= 3 {
					hasReconstructionCard = true
				}
			}
		}

		if isCodeDiscipline(object.Discipline) {
			if !hasFoundationalCard || !hasTraceCard || !hasLineOrderCard || !hasReconstructionCard {
				return fmt.Errorf("%w: code info objects must follow the progression with foundational, trace, line-order, and reconstruction cards", ErrInvalidSuggestion)
			}
		}
	}

	return nil
}

func isCodeDiscipline(discipline string) bool {
	d := strings.ToLower(strings.TrimSpace(discipline))
	return d == "programming" || d == "computer science"
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

func deduplicateDistractors(objects []SuggestedObject) {
	for i := range objects {
		for j := range objects[i].Cards {
			card := &objects[i].Cards[j]
			correctSet := make(map[string]struct{})
			for _, ans := range card.CorrectAnswers {
				for _, tok := range ans {
					correctSet[tok] = struct{}{}
				}
			}
			filtered := card.Distractors[:0]
			for _, d := range card.Distractors {
				if _, dup := correctSet[d]; !dup {
					filtered = append(filtered, d)
				}
			}
			card.Distractors = filtered
		}
	}
}
