// Command seed-premade-deck imports a hand-reviewed deck JSON file directly into the
// premade_decks / premade_info_objects / premade_cards tables as official content.
//
// Usage (from backend/):
//
//	go run ./cmd/seed-premade-deck -file ../content/premade-decks/algorithms-ds-part1.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	cardpkg "shmanki/internal/card"
	"shmanki/internal/platform/config"
	platformdb "shmanki/internal/platform/db"
)

type deckFile struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	LanguageCode string       `json:"languageCode"`
	Category     string       `json:"category"`
	InfoObjects  []infoObject `json:"infoObjects"`
}

type infoObject struct {
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Discipline  string     `json:"discipline"`
	ContentType string     `json:"contentType"`
	Cards       []cardItem `json:"cards"`
}

type cardItem struct {
	Step           int              `json:"step"`
	CardType       cardpkg.CardType `json:"cardType"`
	Front          string           `json:"front"`
	CorrectAnswers [][]string       `json:"correctAnswers"`
	Distractors    []string         `json:"distractors"`
}

func main() {
	filePath := flag.String("file", "", "path to the deck JSON file to import")
	publish := flag.Bool("publish", true, "publish the deck immediately (is_published = true)")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("usage: seed-premade-deck -file path/to/deck.json")
	}

	raw, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("read deck file: %v", err)
	}

	var deck deckFile
	if err := json.Unmarshal(raw, &deck); err != nil {
		log.Fatalf("parse deck file: %v", err)
	}

	if err := validateDeck(deck); err != nil {
		log.Fatalf("invalid deck file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	dbPool, err := platformdb.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()

	deckID, err := importDeck(ctx, dbPool, deck, *publish)
	if err != nil {
		log.Fatalf("import deck: %v", err)
	}

	fmt.Printf("imported premade deck %q (%s) with %d info objects\n", deck.Title, deckID, len(deck.InfoObjects))
}

func importDeck(ctx context.Context, pool *pgxpool.Pool, deck deckFile, publish bool) (uuid.UUID, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	languageCode := strings.TrimSpace(deck.LanguageCode)
	if languageCode == "" {
		languageCode = "en"
	}

	var deckID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO premade_decks (user_id, source, source_deck_id, title, description, language_code, category, is_published)
VALUES (NULL, 'official', NULL, $1, $2, $3, $4, $5)
RETURNING id`, deck.Title, deck.Description, languageCode, deck.Category, publish).Scan(&deckID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert premade deck: %w", err)
	}

	for _, obj := range deck.InfoObjects {
		var objectID uuid.UUID
		err = tx.QueryRow(ctx, `
INSERT INTO premade_info_objects (premade_deck_id, title, content, discipline, content_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING id`, deckID, obj.Title, obj.Content, obj.Discipline, obj.ContentType).Scan(&objectID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert info object %q: %w", obj.Title, err)
		}

		for _, c := range obj.Cards {
			answersJSON, err := json.Marshal(c.CorrectAnswers)
			if err != nil {
				return uuid.Nil, fmt.Errorf("marshal correct answers (%s, step %d) for %q: %w", c.CardType, c.Step, obj.Title, err)
			}
			distractorsJSON, err := json.Marshal(c.Distractors)
			if err != nil {
				return uuid.Nil, fmt.Errorf("marshal distractors (%s, step %d) for %q: %w", c.CardType, c.Step, obj.Title, err)
			}

			if _, err := tx.Exec(ctx, `
INSERT INTO premade_cards (premade_info_object_id, front, step, correct_answers, distractors, card_type)
VALUES ($1, $2, $3, $4, $5, $6)`, objectID, c.Front, c.Step, answersJSON, distractorsJSON, c.CardType); err != nil {
				return uuid.Nil, fmt.Errorf("insert card (%s, step %d) for %q: %w", c.CardType, c.Step, obj.Title, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit tx: %w", err)
	}

	return deckID, nil
}

func validateDeck(deck deckFile) error {
	if strings.TrimSpace(deck.Title) == "" {
		return fmt.Errorf("deck title is required")
	}
	if strings.TrimSpace(deck.Category) == "" {
		return fmt.Errorf("deck category is required")
	}
	if len(deck.InfoObjects) == 0 {
		return fmt.Errorf("deck must contain at least one info object")
	}

	for _, obj := range deck.InfoObjects {
		if strings.TrimSpace(obj.Title) == "" {
			return fmt.Errorf("info object title is required")
		}
		if len(obj.Cards) == 0 {
			return fmt.Errorf("info object %q must contain at least one card", obj.Title)
		}

		for _, c := range obj.Cards {
			if strings.TrimSpace(c.Front) == "" {
				return fmt.Errorf("card front is required in %q", obj.Title)
			}
			if !c.CardType.Valid() {
				return fmt.Errorf("invalid card type %q in %q", c.CardType, obj.Title)
			}
			if len(c.CorrectAnswers) == 0 {
				return fmt.Errorf("card %q (step %d) in %q must have at least one correct answer variant", c.CardType, c.Step, obj.Title)
			}
			for _, variant := range c.CorrectAnswers {
				if len(variant) == 0 {
					return fmt.Errorf("card %q (step %d) in %q has an empty answer variant", c.CardType, c.Step, obj.Title)
				}
			}
		}
	}

	return nil
}
