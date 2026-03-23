# AI Agent Guidelines — Qwik Frontend Developer

## Role

You are an expert Qwik frontend developer working on a spaced repetition learning platform.
You write idiomatic, production-quality Qwik + TypeScript code following the conventions below.

---

## Qwik Fundamentals — What Makes Qwik Different

Qwik is NOT React. The core concept is **resumability** — the app serializes state to HTML
and resumes on the client without re-running components. This has critical implications:

### Lazy execution — `$` suffix

Any function passed to an event handler or lifecycle hook must be serializable.
Qwik uses the `$` suffix to mark lazy-loaded boundaries:

```tsx
// GOOD
const handleClick = $(() => {
  count.value++;
});

// BAD — will not serialize correctly
const handleClick = () => {
  count.value++;
};
```

### Signals — reactive state

```tsx
import { useSignal, useStore } from "@builder.io/qwik";

// primitive value
const count = useSignal(0);
count.value++; // read/write via .value

// object state
const state = useStore({ name: "", loading: false });
state.name = "hello"; // direct mutation is fine
```

### No `useEffect` — use `useTask$`

```tsx
import { useTask$ } from "@builder.io/qwik";

// Runs on server + client, re-runs when tracked signals change
useTask$(({ track }) => {
  const val = track(() => mySignal.value);
  // do something with val
});
```

### Server vs Client

```tsx
import { useVisibleTask$ } from "@builder.io/qwik";

// Runs ONLY on client, after component is visible
// Use sparingly — only for browser-only APIs (canvas, localStorage, etc.)
useVisibleTask$(() => {
  // safe to use window, document here
});
```

---

## Project Structure (`frontend/`)

```
frontend/
├── src/
│   ├── routes/                  # file-based routing (like Next.js pages)
│   │   ├── index.tsx            # home page /
│   │   ├── layout.tsx           # root layout (nav, auth wrapper)
│   │   ├── auth/
│   │   │   ├── login/index.tsx
│   │   │   └── register/index.tsx
│   │   ├── decks/
│   │   │   ├── index.tsx        # /decks — list
│   │   │   └── [deckId]/
│   │   │       └── index.tsx    # /decks/:deckId — deck detail
│   │   ├── objects/
│   │   │   └── [objectId]/
│   │   │       └── index.tsx    # /objects/:objectId — info object detail
│   │   └── review/
│   │       └── index.tsx        # /review — review session
│   │
│   ├── components/              # shared reusable components
│   │   ├── card-viewer/         # shows info object content with highlights
│   │   ├── token-answer/        # token click mechanic for answering
│   │   ├── progress-bar/        # stability/mastery indicator
│   │   └── code-block/          # syntax highlighted code via Shiki
│   │
│   ├── lib/
│   │   ├── api.ts               # typed API client (fetch wrapper)
│   │   ├── auth.ts              # JWT storage and helpers
│   │   └── types.ts             # shared TypeScript types matching backend models
│   │
│   └── global.css               # CSS variables, resets, base styles
│
├── public/
├── package.json
└── vite.config.ts
```

---

## TypeScript Types (`lib/types.ts`)

Keep types in sync with the backend Go models:

```typescript
export type CardStatus = "locked" | "new" | "learning" | "review" | "relearning";
export type Rating = 1 | 2 | 3 | 4;
export type ContentType = "text" | "code_go" | "code_python" | "code_js" | "code_ts" | "code_rust";

export interface User {
  id: string;
  email: string;
}

export interface Deck {
  id: string;
  title: string;
  description: string;
  createdAt: string;
}

export interface InfoObject {
  id: string;
  deckId: string;
  title: string;
  content: string;
  contentType: ContentType;
}

export interface Card {
  id: string;
  infoObjectId: string;
  front: string;
  step: number;
  correctAnswers: string[][];
  distractors: string[];
  highlightLines: number[];
}

export interface CardState {
  cardId: string;
  stability: number;
  difficulty: number;
  retrievability: number;
  dueDate: string;
  status: CardStatus;
  reps: number;
  lapses: number;
}

export interface ReviewCard extends Card {
  state: CardState;
  infoObject: InfoObject;
}

export interface ReviewAttempt {
  tokens: string[];
  hadDistractor: boolean;
  wasCorrect: boolean;
}

export interface ReviewSubmission {
  cardId: string;
  answeredTokens: string[];
  attempts: ReviewAttempt[];
  wrongAttemptsCount: number;
  distractorClicksCount: number;
  incorrectTokensClicked: string[];
}

export interface ReviewResult {
  state: CardState;
  rating: Rating;
  wasCorrect: boolean;
}
```

---

## API Client (`lib/api.ts`)

All client-side backend calls go through a single typed client.
With bearer auth stored in `localStorage`, authenticated data fetching and mutations should go through this client.
`routeLoader$` / `routeAction$` are still fine for public or non-authenticated pages.

```typescript
const BASE_URL = import.meta.env.PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem("jwt");
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "unknown error" }));
    throw new Error(err.error ?? `HTTP ${res.status}`);
  }
  return res.json();
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      request<{ token: string }>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      }),
    register: (email: string, password: string) =>
      request<{ token: string }>("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      }),
  },
  decks: {
    list: () => request<Deck[]>("/api/v1/decks"),
    create: (title: string, description: string) =>
      request<Deck>("/api/v1/decks", {
        method: "POST",
        body: JSON.stringify({ title, description }),
      }),
  },
  review: {
    getSession: (limit = 20) => request<ReviewCard[]>(`/api/v1/review/session?limit=${limit}`),
    submit: (input: ReviewSubmission) =>
      request<ReviewResult>("/api/v1/review/submit", {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
};
```

## Token Answer Component

The core mechanic: user clicks tokens in correct order.

```tsx
// components/token-answer/index.tsx
import { component$, useSignal, $ } from "@builder.io/qwik";

interface Props {
  correctAnswers: string[][]; // [[token, token], ...]
  distractors: string[];
  cardId: string;
  onSubmit$: (submission: ReviewSubmission) => void;
}

export const TokenAnswer = component$<Props>(
  ({ correctAnswers, distractors, cardId, onSubmit$ }) => {
    // Shuffle correct tokens from first answer + distractors
    const allTokens = [...correctAnswers[0], ...distractors].sort(() => Math.random() - 0.5);

    const clicked = useSignal<string[]>([]);
    const failed = useSignal(false);
    const attempts = useSignal<ReviewAttempt[]>([]);
    const wrongAttemptsCount = useSignal(0);
    const distractorClicksCount = useSignal(0);
    const incorrectTokensClicked = useSignal<string[]>([]);

    const handleToken = $((token: string) => {
      const next = [...clicked.value, token];
      clicked.value = next;

      const expected = correctAnswers[0];
      const currentAttemptHasDistractor = next.some((part) => distractors.includes(part));

      if (distractors.includes(token)) {
        distractorClicksCount.value++;
        incorrectTokensClicked.value = [...incorrectTokensClicked.value, token];
      }

      // Check if still on a valid path
      for (let i = 0; i < next.length; i++) {
        if (next[i] !== expected[i]) {
          failed.value = true;
          wrongAttemptsCount.value++;
          attempts.value = [
            ...attempts.value,
            { tokens: next, hadDistractor: currentAttemptHasDistractor, wasCorrect: false },
          ];
          clicked.value = [];
          return;
        }
      }

      // Check if complete
      if (next.length === expected.length) {
        const completedAttempts = [
          ...attempts.value,
          { tokens: next, hadDistractor: currentAttemptHasDistractor, wasCorrect: true },
        ];
        attempts.value = completedAttempts;
        onSubmit$({
          cardId,
          answeredTokens: next,
          attempts: completedAttempts,
          wrongAttemptsCount: wrongAttemptsCount.value,
          distractorClicksCount: distractorClicksCount.value,
          incorrectTokensClicked: incorrectTokensClicked.value,
        });
      }
    });

    return (
      <div class='token-answer'>
        <div class='token-pool'>
          {allTokens.map((token) => (
            <button
              key={token}
              class={{
                token: true,
                selected: clicked.value.includes(token),
              }}
              onClick$={() => handleToken(token)}
            >
              {token}
            </button>
          ))}
        </div>
        <div class='answer-so-far'>{clicked.value.join(" ")}</div>
        {failed.value && <p class='error'>Incorrect — try again</p>}
      </div>
    );
  },
);
```

---

## Code Block with Line Highlighting

Uses **Shiki** for syntax highlighting with line-level highlighting:

```bash
npm install shiki
```

```tsx
// components/code-block/index.tsx
import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import { codeToHtml } from "shiki";
import type { ContentType } from "~/lib/types";

interface Props {
  code: string;
  contentType: ContentType;
  highlightLines: number[]; // 1-indexed line numbers to highlight
}

const langMap: Record<ContentType, string> = {
  text: "text",
  code_go: "go",
  code_python: "python",
  code_js: "javascript",
  code_ts: "typescript",
  code_rust: "rust",
};

export const CodeBlock = component$<Props>(({ code, contentType, highlightLines }) => {
  const html = useSignal("");

  useTask$(async () => {
    html.value = await codeToHtml(code, {
      lang: langMap[contentType] ?? "text",
      theme: "github-dark",
      transformers: [
        {
          line(node, line) {
            if (highlightLines.includes(line)) {
              node.properties["data-highlighted"] = "true";
            }
          },
        },
      ],
    });
  });

  return <div class='code-block' dangerouslySetInnerHTML={html.value} />;
});
```

```css
/* global.css */
.code-block [data-highlighted] {
  background-color: rgba(255, 215, 0, 0.12);
  border-left: 3px solid #ffd700;
  display: block;
  width: 100%;
}
```

---

## Routing & Auth Guard

```tsx
// routes/layout.tsx
import { component$, Slot, useVisibleTask$ } from "@builder.io/qwik";
import { useNavigate } from "@builder.io/qwik-city";

export default component$(() => {
  const nav = useNavigate();

  useVisibleTask$(() => {
    const token = localStorage.getItem("jwt");
    const isAuthRoute = window.location.pathname.startsWith("/auth");
    if (!token && !isAuthRoute) {
      nav("/auth/login");
    }
  });

  return <Slot />;
});
```

---

## Conventions

- All components in `components/` are **presentational** — no direct API calls
- Authenticated requests use the shared `api` client with `Authorization: Bearer <token>`
- Store JWT in `localStorage` on web; mobile clients should store the same token in platform-local persistent storage
- Review submissions must send final answer plus attempt metadata for backend rating and analytics
- Never use `any` in TypeScript
- CSS: use CSS variables from `global.css` for colors and spacing — no hardcoded hex values in components
- File names: `kebab-case` for files and folders
- Component names: `PascalCase`
- Prefer `useSignal` for primitives, `useStore` for objects with multiple fields
- Never access `window` or `document` outside `useVisibleTask$`
