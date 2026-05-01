package generate

import (
	"time"

	"github.com/google/uuid"

	cardpkg "shmanki/internal/card"
)

type SuggestRequest struct {
	DeckID      uuid.UUID `json:"deckId"`
	Prompt      string    `json:"prompt"`
	Discipline  string    `json:"discipline,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
}

type SaveRequest struct {
	DeckID       uuid.UUID         `json:"deckId"`
	Prompt       string            `json:"prompt"`
	Model        string            `json:"model"`
	InfoObjects  []SuggestedObject `json:"infoObjects"`
	GenerationID *uuid.UUID        `json:"generationId,omitempty"`
}

type SuggestedObject struct {
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	Discipline  string          `json:"discipline"`
	ContentType string          `json:"contentType"`
	Cards       []SuggestedCard `json:"cards"`
}

type SuggestedCard struct {
	Front          string     `json:"front"`
	CardType       cardpkg.CardType `json:"cardType"`
	Step           int        `json:"step"`
	CorrectAnswers [][]string `json:"correctAnswers"`
	Distractors    []string   `json:"distractors"`
	HighlightLines []int      `json:"highlightLines"`
}

type SuggestResponse struct {
	GenerationID *uuid.UUID        `json:"generationId,omitempty"`
	Model        string            `json:"model"`
	InfoObjects  []SuggestedObject `json:"infoObjects"`
}

type SaveResponse struct {
	InfoObjects []SavedObject `json:"infoObjects"`
}

type SavedObject struct {
	ID          uuid.UUID   `json:"id"`
	DeckID      uuid.UUID   `json:"deckId"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Discipline  string      `json:"discipline"`
	ContentType string      `json:"contentType"`
	Cards       []SavedCard `json:"cards"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type SavedCard struct {
	ID             uuid.UUID  `json:"id"`
	InfoObjectID   uuid.UUID  `json:"infoObjectId"`
	Front          string     `json:"front"`
	CardType       cardpkg.CardType `json:"cardType"`
	Step           int        `json:"step"`
	CorrectAnswers [][]string `json:"correctAnswers"`
	Distractors    []string   `json:"distractors"`
	HighlightLines []int      `json:"highlightLines"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type generationLog struct {
	ID         uuid.UUID
	DeckID     uuid.UUID
	UserID     uuid.UUID
	Prompt     string
	Provider   string
	Model      string
	ObjectsRaw []byte
	CardsCount int
	CreatedAt  time.Time
}

type llmCompletionRequest struct {
	Prompt       string
	LanguageCode string
	Discipline   string
	ContentType  string
}

type llmCompletionResult struct {
	Provider string
	Model    string
	Objects  []SuggestedObject
}
