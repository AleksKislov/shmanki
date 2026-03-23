# Project Context for AI Agents

## What This Project Is

A spaced repetition learning platform for developers.
Users study programming concepts through flashcards with automatic content generation.

Key differentiators vs Anki:

- Custom FSRS implementation with step-based card unlocking
- Cards use token-selection answers (click tokens in order) instead of self-rating buttons
- Automatic card generation from a topic via LLM
- InfoObjects: cards are grouped under reference content (code/text) with line highlighting

---

## Repository Structure

Current docs assume this target layout:

```
.
├── .ai/
│   ├── README.md
│   ├── project.md
│   └── guidelines/
│       ├── backend.md
│       └── frontend.md
│
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── user/
│   ├── deck/
│   ├── card/
│   ├── review/
│   ├── fsrs/
│   ├── generate/
│   └── platform/
├── migrations/
├── frontend/
│   └── src/
│       ├── routes/
│       ├── components/
│       └── lib/
└── specs/
    ├── scheduler_algorithm.md
    ├── database.md
    └── architecture/
        ├── backend.md
        └── frontend.md
```

---

## Guidelines Index

| File                         | When to read                       |
| ---------------------------- | ---------------------------------- |
| `.ai/guidelines/backend.md`  | Working on anything in `backend/`  |
| `.ai/guidelines/frontend.md` | Working on anything in `frontend/` |

## Specs Index

| File                             | What it covers                                                   |
| -------------------------------- | ---------------------------------------------------------------- |
| `specs/scheduler_algorithm.md`   | FSRS algorithm: all formulas, weights, rating logic, step unlock |
| `specs/database.md`              | Full PostgreSQL schema with all tables, indexes, key queries     |
| `specs/architecture/backend.md`  | Backend structure, modules, API endpoints, error format, config  |
| `specs/architecture/frontend.md` | Frontend structure, components, routes, API integration, config  |

**Always read the relevant guidelines + spec before implementing a feature.**

---

## Domain Model (Quick Reference)

```
User
 └── Deck (collection of info objects)
       └── InfoObject (a concept/topic with full reference content)
              ├── content: string        (full code/text shown as reference)
              ├── content_type: string   (text | code_go | code_python | ...)
             └── Card (a question about the info object)
                   ├── step: int              (unlock order, 0 = always available)
                   ├── correct_answers: JSONB ([[token, token, ...], ...])
                   ├── distractors: JSONB     ([token, token, ...])
                   └── highlight_lines: JSONB ([1, 5, 6, ...])

CardState (per Card per User — FSRS parameters)
 ├── stability: float       (S: days until R drops to 0.9)
 ├── difficulty: float      (D: 1.0–10.0)
 ├── retrievability: float  (R: current recall probability)
 ├── due_date: timestamp
 ├── status: locked | new | learning | review | relearning
 └── reps, lapses: int

ReviewLog (immutable history, one row per review event)
```

---

## FSRS Quick Reference

```
R(t) = (1 + Factor × t / S) ^ Decay
Factor = 19/81 ≈ 0.2346,  Decay = -0.5

Ratings:  1=Again  2=Hard  3=Good  4=Easy
          ↓
          Determined automatically from review attempt metadata:
          incorrect final answer                      → Again (1)
          correct, but with at least one wrong attempt → Hard (2)
          correct, no wrong attempts, some distractors → Good (3)
          correct on first clean attempt               → Easy (4)

Step unlock: all cards at step N need S >= 14 days → step N+1 unlocks
```

---

## Answer Mechanic

The user sees a shuffled pool of `correct_answers[0]` tokens + `distractors`.
They click tokens one by one — order matters.
Frontend tracks the full attempt history for the card, including:

- final `answered_tokens`
- `wrong_attempts_count`
- `distractor_clicks_count`
- `incorrect_tokens_clicked`
- `attempts` history for replay/debugging

Backend checks whether the final sequence matches any entry in `correct_answers`
and derives the FSRS rating from correctness plus the attempt metadata.

---

## Tech Stack

|                     | Technology                |
| ------------------- | ------------------------- |
| Backend language    | Go 1.22+                  |
| HTTP router         | chi v5                    |
| Database            | PostgreSQL 16+ via pgx/v5 |
| Auth                | JWT (golang-jwt/jwt v5)   |
| Frontend framework  | Qwik + Qwik City          |
| Syntax highlighting | Shiki                     |
| LLM                 | Anthropic API             |

---

## Current Status

- [ ] Project scaffolding
- [ ] Database migrations
- [ ] User auth (register / login / JWT)
- [ ] Deck CRUD
- [ ] InfoObject + Card CRUD
- [ ] FSRS scheduler implementation
- [ ] Review session endpoints
- [ ] Step unlock logic
- [ ] Token answer component (frontend)
- [ ] Code block with line highlighting (frontend)
- [ ] LLM generation endpoint
- [ ] Stats endpoints
