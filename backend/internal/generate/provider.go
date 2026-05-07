package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type completionClient interface {
	Complete(ctx context.Context, req llmCompletionRequest) (*llmCompletionResult, error)
}

type llmConfig struct {
	APIURL   string
	APIKey   string
	Model    string
	Provider string
	Timeout  time.Duration
}

type chatCompletionsClient struct {
	httpClient *http.Client
	config     llmConfig
}

type chatCompletionsRequest struct {
	Model       string                  `json:"model"`
	Temperature float64                 `json:"temperature"`
	Messages    []chatCompletionMessage `json:"messages"`
	ResponseFmt *chatCompletionRespFmt  `json:"response_format,omitempty"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRespFmt struct {
	Type       string                    `json:"type"`
	JSONSchema *chatCompletionJSONSchema `json:"json_schema,omitempty"`
}

type chatCompletionJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func newCompletionClient(cfg llmConfig) completionClient {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.APIURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &chatCompletionsClient{
		httpClient: &http.Client{Timeout: timeout},
		config:     cfg,
	}
}

func NewClient(apiURL string, apiKey string, model string, provider string, timeoutSeconds int) completionClient {
	return newCompletionClient(llmConfig{
		APIURL:   apiURL,
		APIKey:   apiKey,
		Model:    model,
		Provider: provider,
		Timeout:  time.Duration(timeoutSeconds) * time.Second,
	})
}

func (c *chatCompletionsClient) Complete(ctx context.Context, req llmCompletionRequest) (*llmCompletionResult, error) {
	payload := chatCompletionsRequest{
		Model:       c.config.Model,
		Temperature: 0.4,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: generationSystemPrompt()},
			{Role: "user", Content: generationUserPrompt(req)},
		},
		ResponseFmt: generationResponseFormat(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal llm request: %w", err)
	}

	log.Printf("[LLM] --> POST %s model=%s", c.config.APIURL, c.config.Model)
	log.Printf("[LLM] --> prompt=%q", req.Prompt)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.APIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create llm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call llm api: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[LLM] <-- error status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		return nil, fmt.Errorf("llm api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	log.Printf("[LLM] <-- status=%d body=%s", resp.StatusCode, string(responseBody))

	var completion chatCompletionsResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("llm response contained no choices")
	}

	var structured struct {
		InfoObjects []SuggestedObject `json:"infoObjects"`
	}
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &structured); err != nil {
		return nil, fmt.Errorf("decode llm structured content: %w", err)
	}

	return &llmCompletionResult{
		Provider: c.config.Provider,
		Model:    c.config.Model,
		Objects:  structured.InfoObjects,
	}, nil
}

func generationSystemPrompt() string {
	return strings.TrimSpace(`You generate study content for a spaced repetition system.
The response JSON schema is enforced separately. Follow it exactly.
Generate concise, technically accurate material.
Infer discipline and contentType when they are not explicitly provided.
Use contentType as either "text" or the plain language name of the content, such as "typescript", "go", "python", "english", or "chinese".
For code content, follow this progression:
  - Step 0: concept or signature cards about purpose, methods, signatures, parameter types, and return types.
  - Step 1: trace cards about control flow, invariants, and what key lines do.
  - Step 2: line_order cards for reconstructing a single specific method from its individual lines. Never ask about an entire class or file — scope each card to one method only.
  - Step 3+: choose_snippet or fix_bug cards scoped to a single method: selecting the correct implementation of that method or identifying the bug within it.
  - Every non-trivial code infoObject must include at least one step 0 foundational card, one step 1 trace card, one step 2 line_order card, and one step 3+ reconstruction card.
Every non-trivial code infoObject must include at least one card from each of those progression levels.
If the content is code, contentType must match the actual language of the code.
Do not include explanations outside the schema response.

Distractor and answer quality rules — follow these strictly for every card:
  - Keep each correct answer token short and precise. Test one specific fact per token; do not pack multiple details into a single answer. If the concept has many facets, spread them across multiple cards.
  - Distractors must be the same length and style as the correct answer. A user must not be able to identify the correct answer by its length, formatting, or level of detail alone.
  - Distractors must be plausibly wrong: subtly incorrect in a way that requires genuine knowledge to reject. Use the same vocabulary and phrasing as the correct answer but change a key detail — wrong type, inverted condition, off-by-one, incorrect complexity, swapped terms, etc.
  - Never use obviously nonsensical or unrelated distractors.
  - For trace and signature cards: distractors should be lines or signatures that could plausibly appear in similar code but contain a specific error.`)
}

func generationUserPrompt(req llmCompletionRequest) string {
	discipline := strings.TrimSpace(req.Discipline)
	if discipline == "" {
		discipline = "infer from the user prompt and generated content"
	}

	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "infer from the user prompt and generated content"
	}

	if len(req.ExistingDraft) > 0 {
		draftJSON, err := json.Marshal(req.ExistingDraft)
		if err != nil {
			return fmt.Sprintf("languageCode: %s\ndiscipline: %s\ncontentType: %s\neditInstruction: %s", req.LanguageCode, discipline, contentType, req.Prompt)
		}

		return fmt.Sprintf(
			"languageCode: %s\ndiscipline: %s\ncontentType: %s\neditInstruction: %s\nexistingDraft: %s\nApply the edit instruction to the existing draft. Preserve good unchanged material when possible, but return the full updated draft.",
			req.LanguageCode,
			discipline,
			contentType,
			req.Prompt,
			string(draftJSON),
		)
	}

	return fmt.Sprintf("languageCode: %s\ndiscipline: %s\ncontentType: %s\nuserPrompt: %s", req.LanguageCode, discipline, contentType, req.Prompt)
}

func generationResponseFormat() *chatCompletionRespFmt {
	return &chatCompletionRespFmt{
		Type: "json_schema",
		JSONSchema: &chatCompletionJSONSchema{
			Name:   "study_generation_response",
			Strict: true,
			Schema: generationResponseSchema(),
		},
	}
}

func generationResponseSchema() map[string]any {
	nonBlankString := map[string]any{
		"type":      "string",
		"minLength": 1,
		"pattern":   `.*\S.*`,
	}

	cardSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"front", "cardType", "step", "correctAnswers", "distractors"},
		"properties": map[string]any{
			"front": nonBlankString,
			"cardType": map[string]any{
				"type": "string",
				"enum": []string{"concept", "signature", "trace", "line_order", "choose_snippet", "fix_bug"},
			},
			"step": map[string]any{"type": "integer"},
			"correctAnswers": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items": map[string]any{
					"type":     "array",
					"minItems": 1,
					"items":    nonBlankString,
				},
			},
			"distractors": map[string]any{
				"type": "array",
				"items": nonBlankString,
			},
		},
	}

	infoObjectSchema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"title", "content", "discipline", "contentType", "cards"},
		"properties": map[string]any{
			"title":       nonBlankString,
			"content":     nonBlankString,
			"discipline":  nonBlankString,
			"contentType": nonBlankString,
			"cards": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items":    cardSchema,
			},
		},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"infoObjects"},
		"properties": map[string]any{
			"infoObjects": map[string]any{
				"type":     "array",
				"minItems": 1,
				"items":    infoObjectSchema,
			},
		},
	}
}
