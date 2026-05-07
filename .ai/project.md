# Project Context for AI Agents

## What This Project Is

A spaced repetition learning platform with multilingual UI and language-aware study content.
Users study programming concepts and other structured topics through flashcards with automatic content generation.

Key differentiators vs Anki:

- Custom FSRS implementation with step-based card unlocking
- Cards use token-selection answers (click tokens in order) instead of self-rating buttons
- Automatic card generation from a topic via LLM
- InfoObjects: cards are grouped under reference content (code/text)
- User preferred language drives both UI localization and the default language for newly created decks
- For AI generation, discipline and content_type are inferred from the user's prompt unless explicitly supplied by an internal caller

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
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── internal/
│   │   ├── user/
│   │   ├── deck/
│   │   ├── card/
│   │   ├── review/
│   │   ├── fsrs/
│   │   ├── generate/
│   │   └── platform/
│   ├── go.mod
│   └── go.sum
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

Backend code lives under `backend/`. Frontend code lives under `frontend/`.

## Specs Index

| File                             | What it covers                                                   |
| -------------------------------- | ---------------------------------------------------------------- |
| `specs/scheduler_algorithm.md`   | FSRS algorithm: all formulas, weights, rating logic, step unlock |
| `specs/database.md`              | Full PostgreSQL schema with all tables, indexes, key queries     |
| `specs/i18n.md`                  | UI localization rules, deck language ownership, language scope   |
| `specs/architecture/backend.md`  | Backend structure, modules, API endpoints, error format, config  |
| `specs/architecture/frontend.md` | Frontend structure, components, routes, API integration, config  |

**Always read the relevant guidelines + spec before implementing a feature.**

---

## Domain Model (Quick Reference)

```
User
 ├── preferred_language: string  (BCP 47 code, e.g. en, ru, es, de, fr, zh-CN)
 └── Deck (collection of info objects)
       ├── language_code: string   (source of truth for all nested study content)
       └── InfoObject (a concept/topic with full reference content)
              ├── content: string        (full code/text shown as reference)
              ├── content_type: string   (text | typescript | go | python | english | chinese | ...)
              ├── language: inherited    (inherits from Deck.language_code)
              └── Card (a question about the info object)
                    ├── card_type: string       (concept | signature | trace | line_order | choose_snippet | fix_bug)
                    ├── step: int              (unlock order, 0 = always available)
                    ├── correct_answers: JSONB ([[token, token, ...], ...])
                    ├── distractors: JSONB     ([token, token, ...])
                    ├── language: inherited    (inherits from Deck.language_code)

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

The user sees a shuffled pool of `correct_answers[0]` answer units plus `distractors`.
They build the answer in order. For simple cards the units are tokens or short phrases. For reconstruction cards the units can be full code lines or larger code blocks.
Frontend tracks the full attempt history for the card, including:

- final `answered_tokens`
- `wrong_attempts_count`
- `distractor_clicks_count`
- `incorrect_tokens_clicked`
- `attempts` history for replay/debugging

Backend checks whether the final sequence matches any entry in `correct_answers`
and derives the FSRS rating from correctness plus the attempt metadata.

The answer mechanic is language-agnostic. Matching uses stored card content and
does not apply language-specific grammar or morphology rules yet.

---

## Code Learning Progression

For code-heavy info objects, generation and review should follow this progression:

1. Step 0: `concept` and `signature` cards for purpose, methods, parameter types, and return types.
2. Step 1: `trace` cards for control flow, invariants, and the role of important lines.
3. Step 2: `line_order` cards for reconstructing one method line-by-line.
4. Step 3+: `choose_snippet` and `fix_bug` cards for larger reconstruction and discrimination tasks.

This progression is preferred over pure fact-recall when the source material is an algorithm, data structure, or non-trivial code implementation.

---

## Language Model

See `specs/i18n.md` for language ownership, localization rules, and scope limits.

---

## Tech Stack

|                     | Technology                |
| ------------------- | ------------------------- |
| Backend language    | Go 1.26+                  |
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
- [ ] LLM generation endpoint
- [ ] Stats endpoints
- [ ] User language preference
- [ ] UI localization infrastructure
- [ ] Deck language defaults and editing
