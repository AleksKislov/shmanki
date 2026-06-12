# Database Schema Specification

## Technology

- **PostgreSQL 16+**
- **Driver**: `pgx/v5` (no ORM)
- **Migrations**: `golang-migrate/migrate`
- **UUID generation**: `gen_random_uuid()` (pgcrypto)

Related specs:

- `specs/architecture/backend.md`
- `specs/architecture/frontend.md`
- `specs/scheduler_algorithm.md`
- `specs/i18n.md`

---

## Tables

### `users`

Stores registered users.

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,         -- bcrypt, cost=12
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'en',
    -- User's chosen UI language and default language for new decks.
    -- BCP 47 code, e.g. 'en', 'ru', 'es', 'de', 'fr', 'ja', 'zh-CN'.
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

### `decks`

A deck is a collection of info objects belonging to a user.

```sql
CREATE TABLE decks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    language_code VARCHAR(20) NOT NULL DEFAULT 'en',
    -- Source of truth for all nested study content in this deck.
    -- New decks default to users.preferred_language unless explicitly overridden.
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_decks_user_id ON decks(user_id);
```

`decks` can be published to the premade catalog exactly once via `premade_decks.source_deck_id`.

---

### `info_objects`

A group of related cards. Stores the full reference content (text or code)
that is displayed to the user alongside the cards.
Info objects inherit their language from the parent deck.

```sql
CREATE TABLE info_objects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id      UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,

    content      TEXT NOT NULL DEFAULT '',
    -- Full text or code to display as reference material.
    -- Example: complete Go implementation of a LinkedList.

    discipline    VARCHAR(50) NOT NULL DEFAULT 'programming',
    -- Broad category of info object, e.g. 'programming', 'language', 'history'.

    content_type VARCHAR(50) NOT NULL DEFAULT 'text',
    -- Either 'text' or the plain language name of the stored content.
    -- Examples: 'typescript', 'go', 'python', 'english', 'chinese'.
    -- Language is stored on the parent deck for deck ownership, but content_type describes
    -- the actual syntax or natural language of the info object content.

    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_info_objects_deck_id ON info_objects(deck_id);
```

---

### `cards`

A single flashcard belonging to an info object.
Cards within an info object are grouped by `step`.
Step 0 cards are always available. Step N cards unlock
only when all step N-1 cards have stability >= 14 days.
Cards inherit language from their parent deck.

For code-oriented info objects, cards should follow this progression:

- Step 0: `concept` and `signature` cards for purpose, method list, parameter types, and return types.
- Step 1: `trace` cards for control flow, invariants, and the meaning of key lines.
- Step 2: `line_order` cards for reconstructing a method from individual lines in the correct order.
- Step 3+: `choose_snippet` and `fix_bug` cards for larger reconstruction or discrimination tasks.

All card types still use the shared ordered-answer model. For simple cards the answer units are short tokens or phrases. For reconstruction cards the answer units are whole code lines or code blocks.

```sql
CREATE TABLE cards (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    info_object_id  UUID NOT NULL REFERENCES info_objects(id) ON DELETE CASCADE,

    front           TEXT NOT NULL,
    -- The question shown to the user.
    -- Example: "Реализация метода AddToTail"

    card_type       VARCHAR(32) NOT NULL DEFAULT 'concept',
    -- Interaction type and intended learning behavior.
    -- Values:
    -- 'concept' | 'signature' | 'trace' | 'line_order' |
    -- 'choose_snippet' | 'fix_bug'

    step            INT NOT NULL DEFAULT 0,
    -- Unlock order within the info object.
    -- Step 0: always available.
    -- Step N: unlocks when all step N-1 cards have S >= 14 days.

    correct_answers JSONB NOT NULL DEFAULT '[]',
    -- Array of valid answer-unit sequences.
    -- Each sequence is an ordered array of strings.
    -- The user must place answer units in the exact order of one sequence.
    -- Example: [["go", "myFunc()"], ["go", "myFunc(arg)"]]
    -- For reconstruction cards, units may be full code lines or larger code blocks.

    distractors     JSONB NOT NULL DEFAULT '[]',
    -- Array of incorrect answer-unit strings shown alongside correct answers.
    -- Example: ["defer", "func myFunc()", "goroutine", "chan struct{}"]
    -- For reconstruction cards, these may be wrong code lines or wrong code blocks.

    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cards_info_object_id ON cards(info_object_id);
CREATE INDEX idx_cards_step ON cards(info_object_id, step);
CREATE INDEX idx_cards_card_type ON cards(card_type);
```

---

### `card_states`

FSRS state per (card, user) pair. One row per card per user.
Created when a user first encounters a card (status = 'new' or 'locked').

```sql
CREATE TABLE card_states (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id         UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- FSRS parameters
    stability       FLOAT NOT NULL DEFAULT 0,
    -- S: number of days until R drops to 0.9.
    -- 0 means card has never been reviewed.

    difficulty      FLOAT NOT NULL DEFAULT 5,
    -- D_base: persisted base FSRS difficulty for this user, range 1.0–10.0.
    -- Effective difficulty D_eff is derived at scheduling time from D_base and H_c,
    -- so no separate column is required in the base schema.

    retrievability  FLOAT NOT NULL DEFAULT 0,
    -- R: recall probability at last_review moment (stored for quick stats queries).
    -- Recomputed after each review; not used for scheduling (computed dynamically).

    -- Scheduling
    due_date        TIMESTAMP,
    -- When to show this card next. NULL for locked/new cards.

    last_review     TIMESTAMP,
    -- Timestamp of the most recent review. NULL if never reviewed.

    interval_days   FLOAT NOT NULL DEFAULT 0,
    -- Current scheduled interval in days.
    -- For cards in learning/relearning step mode this can be fractional,
    -- e.g. 10 minutes = 0.00694 days.

    learning_step   INT NOT NULL DEFAULT 0,
    -- Index into the active step list:
    -- learning  -> Config.LearningSteps
    -- relearning -> Config.RelearningSteps
    -- Ignored when status = 'review'.

    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'new',
    -- 'locked'      step prerequisites not yet met
    -- 'new'         available, never reviewed
    -- 'learning'    currently in learning steps (minutes-based)
    -- 'review'      graduated, scheduled by day-based FSRS
    -- 'relearning'  answered Again after being in review

    -- Counters
    reps            INT NOT NULL DEFAULT 0,
    -- Total number of successful reviews (rating >= 2).

    lapses          INT NOT NULL DEFAULT 0,
    -- Total number of Again ratings after a card has already reached review status.

    UNIQUE (card_id, user_id)
);

CREATE INDEX idx_card_states_due ON card_states(user_id, due_date)
    WHERE status IN ('learning', 'review', 'relearning');

CREATE INDEX idx_card_states_user_status ON card_states(user_id, status);
```

---

### `premade_decks`

Catalog decks visible in the premade page. Two sources are supported:

- `official` (admin-managed)
- `community` (published from user-owned decks)

Community decks are snapshots and become immutable for non-admin users.

Core columns:

- `user_id` nullable owner (`NULL` for official decks)
- `source`
- `source_deck_id` unique nullable link to original user deck (prevents duplicate publishing)
- `category` free text
- `is_published` visibility flag
- `rating_avg`, `rating_count` cached aggregates

Related tables:

- `premade_info_objects`
- `premade_cards`
- `premade_deck_ratings` (`score` 1..5, unique `(premade_deck_id, user_id)`)

---

### `review_logs`

Immutable history of every review event.
Used for analytics, debugging, and future per-user weight optimization.

```sql
CREATE TABLE review_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id         UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- State before this review
    stability_before      FLOAT NOT NULL,
    difficulty_before     FLOAT NOT NULL,
    -- Base difficulty before review.
    retrievability_before FLOAT NOT NULL,
    interval_before       FLOAT NOT NULL,
    status_before         VARCHAR(20) NOT NULL,

    -- State after this review
    stability_after       FLOAT NOT NULL,
    difficulty_after      FLOAT NOT NULL,
    -- Base difficulty after review.
    interval_after        FLOAT NOT NULL,
    status_after          VARCHAR(20) NOT NULL,

    -- User input
    rating          SMALLINT NOT NULL,
    -- 1=Again, 2=Hard, 3=Good, 4=Easy

    answered_tokens JSONB NOT NULL DEFAULT '[]',
    -- The token sequence the user actually clicked.
    -- Example: ["go", "myFunc()"]
    -- Stored for replay and analysis.

    was_correct     BOOLEAN NOT NULL,
    -- Whether the user's token sequence matched a correct_answer.

    wrong_attempts_count   INT NOT NULL DEFAULT 0,
    -- Number of failed attempts before the final submission.

    distractor_clicks_count INT NOT NULL DEFAULT 0,
    -- Total number of distractor token clicks across the whole review interaction.

    incorrect_tokens_clicked JSONB NOT NULL DEFAULT '[]',
    -- Flat list of incorrect tokens clicked during the review interaction.

    attempts JSONB NOT NULL DEFAULT '[]',
    -- Ordered history of attempts.
    -- Example:
    -- [
    --   {"tokens": ["defer"], "had_distractor": true, "was_correct": false},
    --   {"tokens": ["go", "myFunc()"], "had_distractor": false, "was_correct": true}
    -- ]

    reviewed_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_logs_card_user ON review_logs(card_id, user_id, reviewed_at DESC);
CREATE INDEX idx_review_logs_user ON review_logs(user_id, reviewed_at DESC);
```

---

### `generation_logs`

Audit log of LLM-based card generation requests.

```sql
CREATE TABLE generation_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id      UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prompt       TEXT NOT NULL,
    model        VARCHAR(100) NOT NULL,
    cards_count  INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
```

Later migration extends this table with provider metadata and raw suggested objects:

```sql
ALTER TABLE generation_logs
ADD COLUMN provider VARCHAR(100) NOT NULL DEFAULT '',
ADD COLUMN objects_raw JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX idx_generation_logs_deck_user ON generation_logs(deck_id, user_id, created_at DESC);
```

---

### `generation_drafts`

Ephemeral server-side drafts for the generation edit/save flow.

Security role:

- Prevents clients from sending arbitrary `infoObjects` to `/generate/save`.
- `edit` and `save` now resolve draft content by `generation_id` on the server.
- Draft rows expire automatically based on `expires_at`.

```sql
CREATE TABLE generation_drafts (
    generation_id UUID PRIMARY KEY REFERENCES generation_logs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    objects_raw JSONB NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX idx_generation_drafts_user_deck ON generation_drafts(user_id, deck_id);
CREATE INDEX idx_generation_drafts_expires_at ON generation_drafts(expires_at);
```

---

## Key Queries

### Get due cards for a user session

Session loading should include both cards that are due and cards that are available for a first review.
That means:

- `learning` / `review` / `relearning` cards with `due_date <= NOW()`
- `new` cards that are already unlocked

```sql
SELECT
    c.id,
    c.front,
    c.correct_answers,
    c.distractors,
    c.step,
    cs.stability,
    cs.difficulty,
    cs.due_date,
    cs.status,
    cs.reps,
    cs.lapses,
    io.content,
    io.content_type,
    d.language_code
FROM card_states cs
JOIN cards c        ON c.id = cs.card_id
JOIN info_objects io ON io.id = c.info_object_id
JOIN decks d        ON d.id = io.deck_id
WHERE
    cs.user_id = $1
    AND (
        (cs.status IN ('learning', 'review', 'relearning') AND cs.due_date <= NOW())
        OR cs.status = 'new'
    )
ORDER BY cs.due_date ASC
LIMIT $2;
```

The response layer may additionally expose derived fields such as `effectiveDifficulty`
and `hierarchicalSupport`, but those are calculated in the review service and are not read directly from SQL.

### Check if step N should be unlocked for a user

```sql
SELECT COUNT(*) = 0 AS should_unlock
FROM cards c
JOIN card_states cs ON cs.card_id = c.id AND cs.user_id = $1
WHERE
    c.info_object_id = $2
    AND c.step = $3          -- step N-1
    AND cs.stability < 14;
```

### Unlock next step cards

```sql
UPDATE card_states
SET status = 'new', due_date = NOW()
WHERE
    user_id = $1
    AND status = 'locked'
    AND card_id IN (
        SELECT id FROM cards
        WHERE info_object_id = $2
        AND step = $3         -- step N
    );
```

---

## Migrations

All migrations live in `migrations/` directory, numbered sequentially:

```
migrations/
  000001_create_users.up.sql
  000001_create_users.down.sql
  000002_create_decks.up.sql
  000002_create_decks.down.sql
  ...
```

Run with:

```bash
migrate -path migrations -database $DATABASE_URL up
```

## Schema Decision For Hierarchical Support

The hierarchical-support algorithm itself still does not require additional persistence.
The only recent schema addition is `card_states.learning_step`, introduced for
minutes-based learning/relearning step scheduling.

- `card_states.difficulty` already stores the persisted base difficulty `D_base`.
- `H_c` is derived from predecessor cards' current `stability` values.
- `D_eff` is a transient scheduling value derived from `D_base` and `H_c`.
- Because both values are reproducible from current state, storing them in `card_states` would duplicate data and risk drift.

Schema changes would only be justified if the product later needs one of these:

- SQL-only analytics over historical `H_c` / `D_eff`
- offline debugging without recomputing from historical predecessor states
- model training that requires exact persisted derived features per review event

If that becomes necessary, prefer adding optional columns to `review_logs` rather than mutating `card_states` semantics.

---

## Language Rules

Field ownership and behavior are defined in `specs/i18n.md`.
