package premade

import (
	"time"

	"github.com/google/uuid"

	"shmanki/internal/card"
)

type Source string

const (
	SourceOfficial  Source = "official"
	SourceCommunity Source = "community"
)

type Deck struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"userId,omitempty"`
	Source       Source     `json:"source"`
	SourceDeckID *uuid.UUID `json:"sourceDeckId,omitempty"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	LanguageCode string     `json:"languageCode"`
	Category     string     `json:"category"`
	IsPublished  bool       `json:"isPublished"`
	RatingAvg    float64    `json:"ratingAvg"`
	RatingCount  int        `json:"ratingCount"`
	AuthorName   string     `json:"authorName"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type DeckDetail struct {
	Deck
	InfoObjects []InfoObject `json:"infoObjects"`
	MyRating    *int         `json:"myRating,omitempty"`
}

type InfoObject struct {
	ID          uuid.UUID   `json:"id"`
	DeckID      uuid.UUID   `json:"premadeDeckId"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Discipline  string      `json:"discipline"`
	ContentType string      `json:"contentType"`
	Cards       []card.Card `json:"cards"`
}

type ListFilters struct {
	Source    string
	Category  string
	Language  string
	MinRating int
	Sort      string
	UserID    uuid.UUID
}

type PublishDeckRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category"`
}

type RateDeckRequest struct {
	Score int `json:"score"`
}

type SetPublishedRequest struct {
	IsPublished bool `json:"isPublished"`
}

type CreateOfficialFromDeckRequest struct {
	DeckID      string `json:"deckId"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category"`
}
