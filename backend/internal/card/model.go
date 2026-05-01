package card

import (
	"time"

	"github.com/google/uuid"
)

type InfoObject struct {
	ID          uuid.UUID `json:"id"`
	DeckID      uuid.UUID `json:"deckId"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Discipline  string    `json:"discipline"`
	ContentType string    `json:"contentType"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Card struct {
	ID             uuid.UUID  `json:"id"`
	InfoObjectID   uuid.UUID  `json:"infoObjectId"`
	Front          string     `json:"front"`
	CardType       CardType   `json:"cardType"`
	Step           int        `json:"step"`
	CorrectAnswers [][]string `json:"correctAnswers"`
	Distractors    []string   `json:"distractors"`
	HighlightLines []int      `json:"highlightLines"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type InfoObjectDetail struct {
	InfoObject
	Cards []Card `json:"cards"`
}

type CreateInfoObjectRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Discipline  string `json:"discipline"`
	ContentType string `json:"contentType"`
}

type UpdateInfoObjectRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Discipline  string `json:"discipline"`
	ContentType string `json:"contentType"`
}

type CreateCardRequest struct {
	Front          string     `json:"front"`
	CardType       CardType   `json:"cardType"`
	Step           int        `json:"step"`
	CorrectAnswers [][]string `json:"correctAnswers"`
	Distractors    []string   `json:"distractors"`
	HighlightLines []int      `json:"highlightLines"`
}

type UpdateCardRequest struct {
	Front          string     `json:"front"`
	CardType       CardType   `json:"cardType"`
	Step           int        `json:"step"`
	CorrectAnswers [][]string `json:"correctAnswers"`
	Distractors    []string   `json:"distractors"`
	HighlightLines []int      `json:"highlightLines"`
}
