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
	Type string `json:"type"`
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
		ResponseFmt: &chatCompletionRespFmt{Type: "json_object"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal llm request: %w", err)
	}

	log.Printf("[LLM] --> POST %s model=%s prompt=%q", c.config.APIURL, c.config.Model, req.Prompt)
	log.Printf("[LLM] --> request body: %s", string(body))

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
Return only valid JSON with the shape {"infoObjects": [...] }.
Each infoObject must contain: title, content, discipline, contentType, cards.
Each card must contain: front, cardType, step, correctAnswers, distractors, highlightLines.
Supported cardType values:
- concept
- signature
- trace
- line_order
- block_order
- choose_snippet
- fix_bug
Rules:
- Create concise, technically accurate learning material.
- Group cards under coherent info objects.
- Use integer steps starting from 0.
- For code content, follow this progression:
  - Step 0: concept or signature cards about purpose, methods, signatures, parameter types, and return types.
  - Step 1: trace cards about control flow, invariants, and what key lines do.
  - Step 2: line_order cards for reconstructing a particular method from individual lines.
  - Step 3+: block_order, choose_snippet, or fix_bug cards for reconstructing larger code blocks or selecting the correct implementation.
  - Every non-trivial code infoObject must include at least one step 0 foundational card, one step 1 trace card, one step 2 line_order card, and one step 3+ reconstruction card.
- correctAnswers must be an array of ordered answer-unit arrays.
- For concept, signature, and trace cards, answer units may be short tokens or short phrases.
- For line_order, block_order, choose_snippet, and fix_bug cards, answer units must be full code lines or full code blocks as strings.
- distractors must be a flat array of strings.
- Distractors for reconstruction cards must be plausible code lines or code blocks.
- highlightLines must reference existing 1-indexed lines inside content.
- If the content is code, contentType must match the actual language of the code. Do not use "text" for TypeScript, Go, Python, JavaScript, or Rust code.
- Do not include explanations outside JSON.`)
}

func generationUserPrompt(req llmCompletionRequest) string {
	return fmt.Sprintf("languageCode: %s\ndiscipline: %s\ncontentType: %s\nuserPrompt: %s", req.LanguageCode, req.Discipline, req.ContentType, req.Prompt)
}
