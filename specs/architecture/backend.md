# Architecture Specification — Backend

## Related Specs

- Backend architecture: this file
- Frontend architecture: `specs/architecture/frontend.md`
- FSRS algorithm: `specs/scheduler_algorithm.md`
- Database schema: `specs/database.md`
- Internationalization: `specs/i18n.md`

---

## Overview

The backend is a **modular monolith** REST API written in Go.
It is structured as independent modules with clear boundaries,
making it easy to extract into microservices later if needed.
It also owns language metadata for users and decks so UI localization and generated study content stay consistent.

---

## Technology Stack

| Layer            | Technology                              |
| ---------------- | --------------------------------------- |
| Language         | Go 1.26+                                |
| HTTP Router      | `go-chi/chi v5`                         |
| Database         | PostgreSQL 16+ via `pgx/v5`             |
| Migrations       | `golang-migrate/migrate`                |
| Auth             | JWT (`golang-jwt/jwt v5`)               |
| Password hashing | `bcrypt` (stdlib `golang.org/x/crypto`) |
| Config           | `godotenv` + `os.Getenv`                |
| UUID             | `google/uuid`                           |
| LLM integration  | Anthropic API via HTTP client           |
| Testing          | stdlib `testing` + `testify`            |

---

## Project Structure

```
.
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # entry point: wire dependencies, start server
│   │
│   ├── internal/
│   │   ├── user/                    # user module
│   │   │   ├── model.go             # User struct
│   │   │   ├── repository.go        # UserRepository interface + pgx impl
│   │   │   ├── service.go           # UserService (register, login)
│   │   │   └── handler.go           # HTTP handlers
│   │   │
│   │   ├── deck/                    # deck module
│   │   │   ├── model.go
│   │   │   ├── repository.go
│   │   │   ├── service.go
│   │   │   └── handler.go
│   │   │
│   │   ├── card/                    # card + info_object module
│   │   │   ├── model.go             # Card, InfoObject, CardState structs
│   │   │   ├── repository.go        # CardRepository, InfoObjectRepository
│   │   │   ├── service.go           # CardService (CRUD, step unlock logic)
│   │   │   └── handler.go
│   │   │
│   │   ├── review/                  # review session module
│   │   │   ├── model.go             # ReviewRequest, ReviewResult
│   │   │   ├── repository.go        # ReviewLogRepository, CardStateRepository
│   │   │   ├── service.go           # ReviewService (submit answer, schedule next)
│   │   │   └── handler.go
│   │   │
│   │   ├── fsrs/                    # FSRS algorithm — pure functions, no I/O
│   │   │   ├── fsrs.go              # core formulas: R, D, S, NextInterval
│   │   │   ├── scheduler.go         # Scheduler struct wrapping all formulas
│   │   │   └── fsrs_test.go         # unit tests for all formulas
│   │   │
│   │   ├── generate/                # LLM content generation module
│   │   │   ├── model.go
│   │   │   ├── service.go           # GenerateService (call LLM, parse response)
│   │   │   └── handler.go
│   │   │
│   │   └── platform/                # shared infrastructure
│   │       ├── db/
│   │       │   └── postgres.go      # DB connection pool setup
│   │       ├── middleware/
│   │       │   ├── auth.go          # JWT validation middleware
│   │       │   ├── logger.go        # request logging
│   │       │   └── recovery.go      # panic recovery
│   │       ├── response/
│   │       │   └── json.go          # writeJSON, writeError helpers
│   │       └── config/
│   │           └── config.go        # Config struct loaded from env
│   ├── go.mod
│   └── go.sum
│
├── migrations/                  # SQL migration files
│   ├── 000001_create_users.up.sql
│   └── ...
│
├── specs/                       # project documentation
│   ├── database.md
│   ├── scheduler_algorithm.md
│   └── architecture/
│       ├── backend.md
│       └── frontend.md
│
├── .ai/                         # AI agent guidelines
│   ├── README.md
│   ├── project.md
│   └── guidelines/
│       ├── backend.md
│       └── frontend.md
│
├── .env.example
└── Makefile
```

---

## Module Boundaries

Each module owns its domain completely:

```
user     → owns: users table
deck     → owns: decks table
card     → owns: cards, info_objects tables
review   → owns: card_states, review_logs tables
generate → owns: generation_logs table
fsrs     → pure logic, no DB access
```

**Modules may only call each other through interfaces**, never by importing
concrete types from another module's internal files.

Example — ReviewService depends on card module through an interface:

```go
// backend/internal/review/service.go
type CardStateReader interface {
    GetDueCards(ctx context.Context, userID uuid.UUID, limit int) ([]*CardState, error)
    GetByID(ctx context.Context, cardID, userID uuid.UUID) (*CardState, error)
}
```

---

## Request Lifecycle

```
HTTP Request
    │
    ▼
chi Router
    │
    ├── middleware: Recovery (panic → 500)
    ├── middleware: Logger (log method, path, duration, status)
    ├── middleware: Auth (validate JWT → inject userID into context)
    │
    ▼
Handler
    ├── decode + validate request body
    ├── extract path/query params
    │
    ▼
Service (business logic)
    ├── orchestrate repositories
    ├── compute hierarchical support from previous-step card states
    ├── call fsrs.Scheduler if needed
    ├── run DB transaction if multi-step write
    │
    ▼
Repository (DB access)
    ├── execute SQL via pgx
    └── return domain structs
    │
    ▼
Handler
    └── writeJSON(w, status, response)
```

---

## Authentication

- **Registration**: `POST /api/v1/auth/register` → bcrypt password → store user → return JWT
- **Login**: `POST /api/v1/auth/login` → verify bcrypt → return JWT
- **JWT payload**: `{ "sub": "<userID>", "exp": <unix> }`
- **JWT lifetime**: 7 days (configurable via `JWT_TTL_HOURS` env)
- **Transport**: clients send `Authorization: Bearer <token>`
- **Storage**: web frontend stores JWT in `localStorage`; mobile clients store it in platform-local persistent storage and send the same bearer header
- **Middleware** validates token and injects `userID` into `context.Context`

```go
// Extracting userID in handlers:
userID := middleware.UserIDFromContext(r.Context())
```

---

## REST API Endpoints

### Auth

| Method | Path                  | Description                      |
| ------ | --------------------- | -------------------------------- |
| POST   | /api/v1/auth/register | Register new user and return JWT |
| POST   | /api/v1/auth/login    | Login and return JWT             |

### Decks

| Method | Path              | Description                |
| ------ | ----------------- | -------------------------- |
| GET    | /api/v1/decks     | List user's decks          |
| POST   | /api/v1/decks     | Create deck                |
| GET    | /api/v1/decks/:id | Get deck with info objects |
| PUT    | /api/v1/decks/:id | Update deck                |
| DELETE | /api/v1/decks/:id | Delete deck                |

### Info Objects

| Method | Path                          | Description                |
| ------ | ----------------------------- | -------------------------- |
| GET    | /api/v1/decks/:deckID/objects | List info objects in deck  |
| POST   | /api/v1/decks/:deckID/objects | Create info object         |
| GET    | /api/v1/objects/:id           | Get info object with cards |
| PUT    | /api/v1/objects/:id           | Update info object         |
| DELETE | /api/v1/objects/:id           | Delete info object         |

### Cards

| Method | Path                            | Description                |
| ------ | ------------------------------- | -------------------------- |
| POST   | /api/v1/objects/:objectID/cards | Create card in info object |
| PUT    | /api/v1/cards/:id               | Update card                |
| DELETE | /api/v1/cards/:id               | Delete card                |

### Review

| Method | Path                   | Description                                             |
| ------ | ---------------------- | ------------------------------------------------------- |
| GET    | /api/v1/review/session | Get unlocked new cards and due cards for review session |
| POST   | /api/v1/review/submit  | Submit answer metadata and get rating + next FSRS state |

`POST /api/v1/review/submit` request body:

```json
{
  "cardId": "uuid",
  "answeredTokens": ["go", "myFunc()"],
  "attempts": [
    {
      "tokens": ["defer"],
      "hadDistractor": true,
      "wasCorrect": false
    },
    {
      "tokens": ["go", "myFunc()"],
      "hadDistractor": false,
      "wasCorrect": true
    }
  ],
  "wrongAttemptsCount": 1,
  "distractorClicksCount": 1,
  "incorrectTokensClicked": ["defer"]
}
```

Response body:

```json
{
  "state": {
    "cardId": "uuid",
    "stability": 12.4,
    "difficulty": 5.8,
    "effectiveDifficulty": 6.4,
    "hierarchicalSupport": 0.7,
    "retrievability": 1,
    "dueDate": "2026-03-24T12:00:00Z",
    "status": "learning",
    "reps": 4,
    "lapses": 1
  },
  "rating": 2,
  "wasCorrect": true
}
```

`difficulty` remains the persisted base difficulty (`D_base`).
`effectiveDifficulty` and `hierarchicalSupport` are derived per scheduling run and returned by the service layer.

### Stats

| Method | Path                   | Description                               |
| ------ | ---------------------- | ----------------------------------------- |
| GET    | /api/v1/stats/deck/:id | Deck-level stats (cards by mastery level) |
| GET    | /api/v1/stats/card/:id | Single card review history                |

### Generation

| Method | Path             | Description                        |
| ------ | ---------------- | ---------------------------------- |
| POST   | /api/v1/generate | Generate cards for a topic via LLM |

### Language-aware request/response fields

- `POST /api/v1/auth/register` may accept `preferredLanguage`
- Auth responses include `token` and `user.preferredLanguage`
- `POST /api/v1/decks` and `PUT /api/v1/decks/:id` accept `languageCode`
- Deck responses include `languageCode`
- Generation uses the target deck's `languageCode`

Detailed language ownership and behavior live in `specs/i18n.md`.

---

## Error Response Format

All errors return JSON:

```json
{
  "error": "card not found",
  "code": "NOT_FOUND"
}
```

Standard error codes:

| HTTP Status | Code             | When                                 |
| ----------- | ---------------- | ------------------------------------ |
| 400         | INVALID_REQUEST  | Malformed JSON or missing fields     |
| 401         | UNAUTHORIZED     | Missing or invalid JWT               |
| 403         | FORBIDDEN        | User doesn't own the resource        |
| 404         | NOT_FOUND        | Resource doesn't exist               |
| 409         | CONFLICT         | Duplicate (e.g. email already taken) |
| 422         | VALIDATION_ERROR | Input fails business rules           |
| 500         | INTERNAL_ERROR   | Unexpected server error              |

---

## Configuration (Environment Variables)

```bash
# Server
PORT=8080
ENV=development         # development | production

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/spacedrep?sslmode=disable

# Auth
JWT_SECRET=your-secret-key-min-32-chars
JWT_TTL_HOURS=168       # 7 days

# LLM
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_MODEL=claude-sonnet-4-20250514

# I18N
DEFAULT_LANGUAGE=en
```

FSRS algorithm constants are defined in `backend/internal/fsrs/config.go`, not in environment variables.

---

## Language Rules

Backend language ownership, validation rules, and scope limits are defined in `specs/i18n.md`.

---

## Makefile Commands

```makefile
make run          # run the server
make build        # build binary
make test         # run all tests
make lint         # run golangci-lint
make migrate-up   # apply all migrations
make migrate-down # rollback last migration
```

---

## Testing Strategy

### Unit tests

- `backend/internal/fsrs/` — 100% coverage, pure functions, no mocks needed
- `backend/internal/*/service.go` — mock repositories with interfaces

### Integration tests

- `backend/internal/*/repository.go` — test against real PostgreSQL (use Docker)
- Use `testing.T` + `testcontainers-go` for isolated DB per test run

### API tests

- Use `net/http/httptest` to test handlers end-to-end
- Test happy path + error cases for every endpoint
