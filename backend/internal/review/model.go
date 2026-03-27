package review

import (
	"time"

	"github.com/google/uuid"

	"shmanki/internal/fsrs"
)

type ReviewAttempt struct {
	Tokens        []string `json:"tokens"`
	HadDistractor bool     `json:"hadDistractor"`
	WasCorrect    bool     `json:"wasCorrect"`
}

type ReviewRequest struct {
	CardID                 uuid.UUID       `json:"cardId"`
	AnsweredTokens         []string        `json:"answeredTokens"`
	Attempts               []ReviewAttempt `json:"attempts"`
	WrongAttemptsCount     int             `json:"wrongAttemptsCount"`
	DistractorClicksCount  int             `json:"distractorClicksCount"`
	IncorrectTokensClicked []string        `json:"incorrectTokensClicked"`
}

type ReviewCard struct {
	CardID         uuid.UUID     `json:"cardId"`
	Front          string        `json:"front"`
	CorrectAnswers [][]string    `json:"correctAnswers"`
	Distractors    []string      `json:"distractors"`
	HighlightLines []int         `json:"highlightLines"`
	Step           int           `json:"step"`
	Content        string        `json:"content"`
	ContentType    string        `json:"contentType"`
	LanguageCode   string        `json:"languageCode"`
	State          CardStateView `json:"state"`
	InfoObjectID   uuid.UUID     `json:"infoObjectId"`
}

type CardStateView struct {
	CardID         uuid.UUID  `json:"cardId"`
	Stability      float64    `json:"stability"`
	Difficulty     float64    `json:"difficulty"`
	Retrievability float64    `json:"retrievability"`
	DueDate        *time.Time `json:"dueDate"`
	Status         string     `json:"status"`
	Reps           int        `json:"reps"`
	Lapses         int        `json:"lapses"`
	IntervalDays   float64    `json:"intervalDays"`
	LastReview     *time.Time `json:"lastReview,omitempty"`
}

type ReviewResult struct {
	State      CardStateView `json:"state"`
	Rating     fsrs.Rating   `json:"rating"`
	WasCorrect bool          `json:"wasCorrect"`
}

type DeckStats struct {
	DeckID uuid.UUID        `json:"deckId"`
	Levels map[string]int64 `json:"levels"`
	Total  int64            `json:"total"`
	DueNow int64            `json:"dueNow"`
	NewNow int64            `json:"newNow"`
}

type CardHistoryEntry struct {
	ReviewedAt            time.Time   `json:"reviewedAt"`
	Rating                fsrs.Rating `json:"rating"`
	WasCorrect            bool        `json:"wasCorrect"`
	StabilityBefore       float64     `json:"stabilityBefore"`
	StabilityAfter        float64     `json:"stabilityAfter"`
	DifficultyBefore      float64     `json:"difficultyBefore"`
	DifficultyAfter       float64     `json:"difficultyAfter"`
	WrongAttemptsCount    int         `json:"wrongAttemptsCount"`
	DistractorClicksCount int         `json:"distractorClicksCount"`
	AnsweredTokens        []string    `json:"answeredTokens"`
	IncorrectTokens       []string    `json:"incorrectTokensClicked"`
}
