# Backend API Usage

## Base URLs

- Development: `http://localhost:8080`
- Production: `https://api.yourdomain.com`

All application endpoints are rooted at `/api/v1`.

## Authentication

- Public endpoints:
  - `GET /healthz`
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
- Every other `/api/v1` endpoint requires:

```http
Authorization: Bearer <jwt>
```

Auth responses return:

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "email": "demo@example.com",
    "preferredLanguage": "en"
  }
}
```

## Error Format

All errors follow this shape:

```json
{
  "error": "card not found",
  "code": "NOT_FOUND"
}
```

Common codes:

- `INVALID_REQUEST`
- `UNAUTHORIZED`
- `NOT_FOUND`
- `CONFLICT`
- `VALIDATION_ERROR`
- `INTERNAL_ERROR`

## Endpoints

### Health

#### `GET /healthz`

Returns a simple liveness response.

Response:

```json
{
  "status": "ok"
}
```

### Auth

#### `POST /api/v1/auth/register`

Create a new user and immediately return a JWT.

Request:

```json
{
  "email": "demo@example.com",
  "password": "demo12345",
  "preferredLanguage": "en"
}
```

Response: `201 Created`

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "email": "demo@example.com",
    "preferredLanguage": "en"
  }
}
```

#### `POST /api/v1/auth/login`

Authenticate an existing user.

Request:

```json
{
  "email": "demo@example.com",
  "password": "demo12345"
}
```

Response: `200 OK`

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "email": "demo@example.com",
    "preferredLanguage": "en"
  }
}
```

### Decks

#### `GET /api/v1/decks`

List all decks owned by the authenticated user.

Response:

```json
[
  {
    "id": "uuid",
    "title": "Go Concurrency Basics",
    "description": "Starter deck",
    "languageCode": "en",
    "createdAt": "2026-03-27T12:00:00Z",
    "updatedAt": "2026-03-27T12:00:00Z"
  }
]
```

#### `POST /api/v1/decks`

Create a deck. If `languageCode` is omitted, the backend defaults it from the user's preferred language.

Request:

```json
{
  "title": "Go Concurrency Basics",
  "description": "Starter deck for local development.",
  "languageCode": "en"
}
```

#### `GET /api/v1/decks/:id`

Get one deck plus its info object summaries.

Response:

```json
{
  "id": "uuid",
  "title": "Go Concurrency Basics",
  "description": "Starter deck",
  "languageCode": "en",
  "createdAt": "2026-03-27T12:00:00Z",
  "updatedAt": "2026-03-27T12:00:00Z",
  "infoObjects": [
    {
      "id": "uuid",
      "title": "Launching a goroutine",
      "discipline": "programming",
      "contentType": "code_go"
    }
  ]
}
```

#### `PUT /api/v1/decks/:id`

Update title, description, or language.

Request:

```json
{
  "title": "Advanced Go Concurrency",
  "description": "Updated deck description.",
  "languageCode": "en"
}
```

#### `DELETE /api/v1/decks/:id`

Delete a deck.

Response:

```json
{
  "status": "deleted"
}
```

### Info Objects

#### `GET /api/v1/decks/:deckID/objects`

List all info objects inside a deck.

Response:

```json
[
  {
    "id": "uuid",
    "deckId": "uuid",
    "title": "Launching a goroutine",
    "content": "func worker() {}",
    "discipline": "programming",
    "contentType": "code_go",
    "createdAt": "2026-03-27T12:00:00Z",
    "updatedAt": "2026-03-27T12:00:00Z"
  }
]
```

#### `POST /api/v1/decks/:deckID/objects`

Create an info object in a deck.

Request:

```json
{
  "title": "Launching a goroutine",
  "content": "func worker() {\n    fmt.Println(\"working\")\n}\n\ngo worker()",
  "discipline": "programming",
  "contentType": "code_go"
}
```

#### `GET /api/v1/objects/:id`

Get one info object plus its cards.

Response:

```json
{
  "id": "uuid",
  "deckId": "uuid",
  "title": "Launching a goroutine",
  "content": "func worker() {}",
  "discipline": "programming",
  "contentType": "code_go",
  "createdAt": "2026-03-27T12:00:00Z",
  "updatedAt": "2026-03-27T12:00:00Z",
  "cards": [
    {
      "id": "uuid",
      "infoObjectId": "uuid",
      "front": "Which expression starts the goroutine?",
      "step": 0,
      "correctAnswers": [["go", "worker()"]],
      "distractors": ["defer", "func", "chan"],
      "highlightLines": [5],
      "createdAt": "2026-03-27T12:00:00Z",
      "updatedAt": "2026-03-27T12:00:00Z"
    }
  ]
}
```

#### `PUT /api/v1/objects/:id`

Update an info object.

Request:

```json
{
  "title": "Launching multiple goroutines",
  "content": "updated content",
  "discipline": "programming",
  "contentType": "code_go"
}
```

#### `DELETE /api/v1/objects/:id`

Delete an info object.

Response:

```json
{
  "status": "deleted"
}
```

### Cards

#### `POST /api/v1/objects/:objectID/cards`

Create a card inside an info object.

Request:

```json
{
  "front": "Which expression starts the goroutine?",
  "step": 0,
  "correctAnswers": [["go", "worker()"]],
  "distractors": ["defer", "func", "chan"],
  "highlightLines": [5]
}
```

#### `PUT /api/v1/cards/:id`

Update a card.

Request:

```json
{
  "front": "Which function name is called by the goroutine?",
  "step": 1,
  "correctAnswers": [["worker()"]],
  "distractors": ["main()", "run()", "job()"],
  "highlightLines": [5]
}
```

#### `DELETE /api/v1/cards/:id`

Delete a card.

Response:

```json
{
  "status": "deleted"
}
```

### Review

#### `GET /api/v1/review/session?limit=20`

Fetch due cards and unlocked new cards for the current session.

Response:

```json
[
  {
    "cardId": "uuid",
    "front": "Which expression starts the goroutine?",
    "correctAnswers": [["go", "worker()"]],
    "distractors": ["defer", "func", "chan"],
    "highlightLines": [5],
    "step": 0,
    "content": "func worker() {}",
    "contentType": "code_go",
    "languageCode": "en",
    "infoObjectId": "uuid",
    "state": {
      "cardId": "uuid",
      "stability": 0,
      "difficulty": 5,
      "effectiveDifficulty": 5,
      "hierarchicalSupport": 1,
      "retrievability": 0,
      "dueDate": "2026-03-27T12:00:00Z",
      "status": "new",
      "reps": 0,
      "lapses": 0,
      "intervalDays": 0
    }
  }
]
```

`difficulty` in the payload is stored base difficulty. `effectiveDifficulty` and `hierarchicalSupport`
are derived by the backend for the current scheduling context.

#### `POST /api/v1/review/submit`

Submit the full answer interaction for one card. The backend derives the FSRS rating from this payload.

Request:

```json
{
  "cardId": "uuid",
  "answeredTokens": ["go", "worker()"],
  "attempts": [
    {
      "tokens": ["defer"],
      "hadDistractor": true,
      "wasCorrect": false
    },
    {
      "tokens": ["go", "worker()"],
      "hadDistractor": false,
      "wasCorrect": true
    }
  ],
  "wrongAttemptsCount": 1,
  "distractorClicksCount": 1,
  "incorrectTokensClicked": ["defer"]
}
```

Response:

```json
{
    "state": {
      "cardId": "uuid",
      "stability": 12.4,
      "difficulty": 5.8,
      "effectiveDifficulty": 6.4,
      "hierarchicalSupport": 0.7,
      "retrievability": 1,
      "dueDate": "2026-03-28T12:00:00Z",
      "status": "learning",
    "reps": 1,
    "lapses": 0,
    "intervalDays": 1,
    "lastReview": "2026-03-27T12:00:00Z"
  },
  "rating": 2,
  "wasCorrect": true
}
```

### Stats

#### `GET /api/v1/stats/deck/:id`

Get aggregate deck-level mastery counters.

Response:

```json
{
  "deckId": "uuid",
  "levels": {
    "new": 3,
    "learning": 4,
    "learned": 2,
    "mastered": 1,
    "expert": 0
  },
  "total": 10,
  "dueNow": 4,
  "newNow": 3
}
```

#### `GET /api/v1/stats/card/:id`

Get review history for one card.

Response:

```json
[
  {
    "reviewedAt": "2026-03-27T12:00:00Z",
    "rating": 2,
    "wasCorrect": true,
    "stabilityBefore": 0,
    "stabilityAfter": 12.4,
    "difficultyBefore": 5,
    "difficultyAfter": 5.8,
    "wrongAttemptsCount": 1,
    "distractorClicksCount": 1,
    "answeredTokens": ["go", "worker()"],
    "incorrectTokensClicked": ["defer"]
  }
]
```

### Generation

#### `POST /api/v1/generate`

Current status: wired, but not implemented yet. Without real generation support, this endpoint returns a not-implemented style error.

Request:

```json
{
  "deckId": "uuid",
  "prompt": "Generate beginner cards for goroutines"
}
```

Current response:

```json
{
  "error": "generation endpoint is not configured yet",
  "code": "INTERNAL_ERROR"
}
```

## Suggested Call Order For Manual Testing

1. `POST /api/v1/auth/register`
2. Save the returned `token`
3. `POST /api/v1/decks`
4. `POST /api/v1/decks/:deckID/objects`
5. `POST /api/v1/objects/:objectID/cards`
6. `GET /api/v1/review/session`
7. `POST /api/v1/review/submit`
8. `GET /api/v1/stats/deck/:id`
9. `GET /api/v1/stats/card/:id`
