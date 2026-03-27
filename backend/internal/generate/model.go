package generate

import "github.com/google/uuid"

type Request struct {
	DeckID uuid.UUID `json:"deckId"`
	Prompt string    `json:"prompt"`
}

type Response struct {
	Message string `json:"message"`
}
