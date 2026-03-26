# AI Agent Guidelines — Go Backend Developer

## Role

You are an expert Go backend developer working on a spaced repetition learning platform.
You write idiomatic, production-quality Go code following the conventions described in this document.

---

## Go Code Style

### General

- Use Go 1.22+
- Always run `gofmt` and `golangci-lint` mentally before suggesting code
- Prefer standard library over third-party packages when reasonable
- Errors are values — handle them explicitly, never ignore with `_` unless justified
- No `panic()` in business logic — only in `main()` or `init()` for unrecoverable startup failures
- Prefer flat package structure over deep nesting

### Naming

- Packages: short, lowercase, no underscores (`fsrs`, `deck`, `card`, `user`)
- Interfaces: single-method interfaces named by method + `-er` (`Reviewer`, `Scheduler`)
- Exported types: `PascalCase`, unexported: `camelCase`
- Acronyms: `userID` not `userId`, `httpClient` not `httpClient`, `fsrsParams` not `fsrsParams`
- Constructor functions: `New` prefix (`NewScheduler`, `NewCardRepository`)

### Error Handling

```go
// GOOD
user, err := repo.GetByID(ctx, id)
if err != nil {
    return fmt.Errorf("get user %s: %w", id, err)
}

// BAD
user, _ := repo.GetByID(ctx, id)
```

- Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Define sentinel errors in domain packages: `var ErrCardNotFound = errors.New("card not found")`
- Use `errors.Is` and `errors.As` for error checking, never string comparison

### Structs and Interfaces

```go
// Define interfaces at the point of use (consumer), not at definition
// BAD: define interface in the repository package
// GOOD: define interface in the service that uses it

type CardScheduler interface {
    Schedule(ctx context.Context, state CardState, rating Rating) (CardState, error)
}
```

### Context

- Always accept `context.Context` as the first argument in functions that do I/O
- Never store context in structs
- Propagate context through the entire call chain

### Database

- Use `pgx/v5` directly — no heavy ORM
- Use named queries and `pgx` struct scanning
- Always use transactions for multi-step writes
- Close rows explicitly with `defer rows.Close()`

### HTTP

- Use `net/http` standard library with a router (chi)
- Return proper HTTP status codes
- Always validate and sanitize input
- Use middleware for auth, logging, recovery

### Internationalization

- Store canonical language codes using BCP 47 (`en`, `ru`, `es`, `de`, `fr`, `ja`, `zh-CN`)
- Validate language codes in one place and reuse that validation across handlers/services
- Persist `users.preferred_language` and default new deck language from it when request omits `languageCode`
- Treat `decks.language_code` as the source of truth for all nested content in that deck
- Do not assume English-only content in prompts, validation, logs, or examples

---

## Project Conventions

### Repository pattern

All database access goes through repository interfaces:

```go
type CardRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*Card, error)
    GetDueCards(ctx context.Context, userID uuid.UUID, limit int) ([]*Card, error)
    UpdateState(ctx context.Context, state CardState) error
}
```

### Service layer

Business logic lives in services, never in handlers or repositories:

```go
type ReviewService struct {
    cards     CardRepository
    scheduler fsrs.Scheduler
    logs      ReviewLogRepository
}
```

For review submissions, the handler request DTO should carry the full attempt metadata,
not just the final answer tokens. Persist the metadata to `review_logs` and derive the
final FSRS rating in the service layer.

```go
type ReviewRequest struct {
    CardID                uuid.UUID       `json:"cardId"`
    AnsweredTokens        []string        `json:"answeredTokens"`
    Attempts              []ReviewAttempt `json:"attempts"`
    WrongAttemptsCount    int             `json:"wrongAttemptsCount"`
    DistractorClicksCount int             `json:"distractorClicksCount"`
    IncorrectTokensClicked []string       `json:"incorrectTokensClicked"`
}
```

For deck and generation flows, carry language metadata explicitly in request/response DTOs.

```go
type CreateDeckRequest struct {
    Title        string `json:"title"`
    Description  string `json:"description"`
    LanguageCode string `json:"languageCode,omitempty"`
}

type User struct {
    ID                uuid.UUID `json:"id"`
    Email             string    `json:"email"`
    PreferredLanguage string    `json:"preferredLanguage"`
}
```

### Handler layer

Handlers only: parse input → call service → write response. No business logic.

```go
func (h *Handler) ReviewCard(w http.ResponseWriter, r *http.Request) {
    var req ReviewRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    result, err := h.service.SubmitReview(r.Context(), req)
    if err != nil {
        handleServiceError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, result)
}
```

---

## What NOT to do

- Do not use `init()` for anything except registering drivers
- Do not use global variables for dependencies — use dependency injection
- Do not use `interface{}` or `any` in business logic types
- Do not return naked errors from handlers — always map to HTTP status
- Do not put SQL in handlers or domain types
- Do not use `time.Sleep` in business logic
- Do not commit secrets or `.env` files
- Do not duplicate language fields across cards/info objects when deck language already defines the content language
