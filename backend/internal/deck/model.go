package deck

import (
	"time"

	"github.com/google/uuid"
)

type Deck struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	LanguageCode string    `json:"languageCode"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type InfoObjectSummary struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Discipline  string    `json:"discipline"`
	ContentType string    `json:"contentType"`
}

type DeckDetail struct {
	Deck
	InfoObjects []InfoObjectSummary `json:"infoObjects"`
}

type CreateDeckRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	LanguageCode string `json:"languageCode,omitempty"`
}

type UpdateDeckRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	LanguageCode string `json:"languageCode,omitempty"`
}
