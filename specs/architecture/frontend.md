# Frontend Architecture Specification

## Related Specs

- Frontend architecture: this file
- Backend architecture: `specs/architecture/backend.md`
- FSRS algorithm: `specs/scheduler_algorithm.md`
- Database schema: `specs/database.md`

---

## Technology Stack

|                     | Technology                         |
| ------------------- | ---------------------------------- |
| Framework           | Qwik + Qwik City (meta-framework)  |
| Language            | TypeScript 5+ (strict mode)        |
| Styling             | Per-component CSS files + global CSS variables |
| Syntax highlighting | Shiki                              |
| Build tool          | Vite                               |
| Package manager     | npm                                |

---

## Project Structure

```
frontend/
├── src/
│   ├── routes/                      # file-based routing (Qwik City)
│   │   ├── layout.tsx               # root layout: nav, auth guard
│   │   ├── index.tsx                # / → redirect to /decks if authed
│   │   ├── auth/
│   │   │   ├── login/
│   │   │   │   └── index.tsx        # /auth/login
│   │   │   └── register/
│   │   │       └── index.tsx        # /auth/register
│   │   ├── decks/
│   │   │   ├── index.tsx            # /decks — deck list
│   │   │   └── [deckId]/
│   │   │       └── index.tsx        # /decks/:deckId — deck detail + info objects
│   │   ├── objects/
│   │   │   └── [objectId]/
│   │   │       └── index.tsx        # /objects/:objectId — info object detail + cards
│   │   └── review/
│   │       └── index.tsx            # /review — active review session
│   │
│   ├── components/
│   │   ├── token-answer/
│   │   │   ├── index.tsx            # token click mechanic
│   │   │   └── token-answer.css
│   │   ├── code-block/
│   │   │   ├── index.tsx            # Shiki syntax highlight + line highlight
│   │   │   └── code-block.css
│   │   ├── card-review/
│   │   │   ├── index.tsx            # full review screen: code + question + tokens
│   │   │   └── card-review.css
│   │   ├── mastery-badge/
│   │   │   └── index.tsx            # visual badge: new/learning/learned/mastered/expert
│   │   ├── progress-bar/
│   │   │   └── index.tsx            # stability progress indicator
│   │   └── nav/
│   │       └── index.tsx            # top navigation bar
│   │
│   ├── lib/
│   │   ├── types.ts                 # TypeScript types (mirror of Go models)
│   │   ├── api.ts                   # typed fetch wrapper for backend API
│   │   ├── auth.ts                  # JWT helpers: save, read, clear
│   │   ├── i18n.ts                  # current locale, dictionary lookup, fallback logic
│   │   └── locales/
│   │       ├── en.ts
│   │       ├── ru.ts
│   │       └── ...
│   │   └── fsrs.ts                  # client-side FSRS helpers (mastery level, R display)
│   │
│   ├── global.css                   # CSS variables, resets, base typography
│   └── entry.ssr.tsx                # SSR entry point (do not edit)
│
├── public/
├── package.json
└── vite.config.ts
```

---

## Routing

Qwik City uses **file-based routing** — each `index.tsx` in `routes/` maps to a URL.

### Route overview

| Route file                            | URL                  | What it shows                         |
| ------------------------------------- | -------------------- | ------------------------------------- |
| `routes/index.tsx`                    | `/`                  | Redirect to `/decks` if authenticated |
| `routes/auth/login/index.tsx`         | `/auth/login`        | Login form                            |
| `routes/auth/register/index.tsx`      | `/auth/register`     | Register form                         |
| `routes/decks/index.tsx`              | `/decks`             | List of user's decks                  |
| `routes/decks/[deckId]/index.tsx`     | `/decks/:deckId`     | Deck detail with info objects         |
| `routes/objects/[objectId]/index.tsx` | `/objects/:objectId` | Info object with cards per step       |
| `routes/review/index.tsx`             | `/review`            | Active review session                 |

### Auth guard

Defined once in `routes/layout.tsx` — applies to all routes.
Because the token is stored in `localStorage`, route protection is enforced on the client after hydration:

```tsx
export default component$(() => {
  const nav = useNavigate();

  useVisibleTask$(() => {
    const token = localStorage.getItem("jwt");
    const isPublic = window.location.pathname.startsWith("/auth");
    if (!token && !isPublic) {
      nav("/auth/login");
    }
  });

  return <Slot />;
});
```

### Localization bootstrap

- The app reads the current locale from the authenticated user profile
- Before profile data is available, the frontend may fall back to `localStorage` or `en`
- All route-level UI text must render through translation keys, not inline literals

---

## Data Fetching

Qwik City provides two patterns for route-driven data fetching.
Because auth uses `localStorage`, authenticated requests should generally use the shared `api.*` client from client-visible code.

### `routeLoader$` — fetch public or non-authenticated page data

Runs before the page renders. Use it when the request does not depend on the browser-held JWT.

```tsx
// routes/index.tsx
export const useLandingContent = routeLoader$(async () => {
  return { headline: "Learn faster with spaced repetition" };
});

export default component$(() => {
  const content = useLandingContent();
  return <Hero headline={content.value.headline} />;
});
```

### `routeAction$` — handle form submissions / mutations

Use for forms that do not require direct access to the browser-held JWT, such as login.

```tsx
export const useLogin = routeAction$(
  async (data, { fail }) => {
    const res = await fetch(`${process.env.API_URL}/api/v1/auth/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });
    if (!res.ok) return fail(400, { message: "Failed to log in" });
    return res.json() as Promise<{ token: string }>;
  },
  zod$({ email: z.string().email(), password: z.string().min(8) }),
);
```

### Client-side fetching (`api.*`)

Use for all authenticated requests, because the JWT lives in `localStorage`:

```
routeLoader$  → public or non-authenticated page data
routeAction$  → login/register forms and similar route actions
api.*         → authenticated deck/object/card/review calls
```

The user profile response should include `preferredLanguage`, and deck payloads should include `languageCode`.

---

## Auth Flow

```
1. User submits login form
        ↓
2. routeAction$ calls POST /api/v1/auth/login
        ↓
3. Backend returns { token: "eyJ..." }
        ↓
4. Frontend stores token in `localStorage`
        ↓
5. Redirect to /decks
        ↓
6. Route guards and pages read auth state from `localStorage`
7. All subsequent API calls send `Authorization: Bearer <token>`
```

**JWT is stored in `localStorage`** for consistency with future mobile clients.
Mobile apps should store the same JWT in the platform's local persistent storage and send the same bearer header.

```tsx
// lib/auth.ts
export function getToken(): string | null {
  return localStorage.getItem("jwt");
}

export function setToken(token: string) {
  localStorage.setItem("jwt", token);
}

export function clearAuth() {
  localStorage.removeItem("jwt");
}
```

```tsx
// lib/i18n.ts
export type LanguageCode = "en" | "ru" | "es" | "de" | "fr" | "ja" | "zh-CN";

export function getLocale(): LanguageCode {
  return (localStorage.getItem("preferredLanguage") as LanguageCode | null) ?? "en";
}

export function setLocale(locale: LanguageCode) {
  localStorage.setItem("preferredLanguage", locale);
}
```

---

## Internationalization

The frontend supports multilingual UI driven by the user's preferred language.
Study content language is defined by the deck.

### Source of truth

- UI locale: `user.preferredLanguage`
- Deck content language: `deck.languageCode`
- Info objects and cards inherit deck language

### Translation resources

- Keep dictionaries in `src/lib/locales/`
- Use stable translation keys such as `nav.decks`, `auth.login`, `deck.language`
- Fallback order: current locale -> `en` -> key itself

### UX rules

- Registration should let the user choose `preferredLanguage`
- New deck forms default `languageCode` from `preferredLanguage`
- Deck edit forms allow changing `languageCode`
- Layouts and components must handle longer translations and non-Latin scripts gracefully
- Do not rely on English-only copy length when designing buttons, forms, or navigation

### Scope limits for this version

- No RTL-specific layout rules yet
- No language-specific token matching or morphology rules yet
- No server-side localized error messages yet

---

## Review Session Flow

This is the most complex UI flow in the app.

```
1. Page load → routeLoader$ fetches GET /api/v1/review/session
   Returns: ReviewCard[] (up to 20 unlocked new cards + due cards)
        ↓
2. Store cards in useStore({ queue: ReviewCard[], current: 0 })
        ↓
3. Show CardReview component for queue[current]:
   ├── CodeBlock: full info object content with highlight_lines for this card's step
   ├── Card front: the question text
   └── TokenAnswer: shuffled pool of correct tokens + distractors
        ↓
4. User clicks tokens in order
        ↓
5. TokenAnswer checks sequence client-side and records interaction metadata
   ├── final answered tokens
   ├── wrong attempts count
   ├── distractor clicks count
   ├── incorrect tokens clicked
   └── ordered attempt history
        ↓
6. api.review.submit({ cardId, answeredTokens, attempts, wrongAttemptsCount,
   distractorClicksCount, incorrectTokensClicked }) → POST /api/v1/review/submit
   Backend derives Rating (Again/Hard/Good/Easy), stores analytics metadata,
   runs FSRS, and returns `ReviewResult`
        ↓
7. current++ → show next card
    If current >= queue.length → show session complete screen
```

### Component hierarchy for review screen

```
routes/review/index.tsx
  └── CardReview                     ← orchestrates the full review UI
        ├── CodeBlock                ← info object content + highlighted lines
        │     (highlight_lines from current card)
        ├── <h2> card.front </h2>    ← the question
        ├── TokenAnswer              ← token click mechanic
        │     ├── token pool (correct + distractors, shuffled)
        │     └── answer-so-far display
        └── SessionProgress          ← X of N cards done
```

### State shape for review session

```tsx
interface ReviewSessionState {
  queue: ReviewCard[]; // all due cards for this session
  currentIndex: number; // which card we're on
  results: ReviewResult[]; // history of this session's answers
  status: "loading" | "active" | "complete";
}

interface ReviewSubmission {
  cardId: string;
  answeredTokens: string[];
  attempts: Array<{
    tokens: string[];
    hadDistractor: boolean;
    wasCorrect: boolean;
  }>;
  wrongAttemptsCount: number;
  distractorClicksCount: number;
  incorrectTokensClicked: string[];
}

interface ReviewResult {
  state: CardState;
  rating: Rating;
  wasCorrect: boolean;
}

interface User {
  id: string;
  email: string;
  preferredLanguage: LanguageCode;
}

interface Deck {
  id: string;
  title: string;
  description: string;
  languageCode: LanguageCode;
}

const session = useStore<ReviewSessionState>({
  queue: [],
  currentIndex: 0,
  results: [],
  status: "loading",
});
```

---

## State Management

Qwik has no global store like Redux. Use this hierarchy:

| Scope                          | Tool                        | When to use                 |
| ------------------------------ | --------------------------- | --------------------------- |
| Component-local primitive      | `useSignal`                 | toggle, count, string input |
| Component-local object         | `useStore`                  | form state, session state   |
| Page-level server data         | `routeLoader$`              | initial data for a route    |
| Cross-component (parent→child) | props                       | pass data down              |
| Cross-component (child→parent) | `QRL` callbacks (`onDone$`) | events up                   |
| Global (auth token)            | `localStorage`              | JWT only                    |

There is **no global client-side store**. If two sibling components need shared state,
lift it to their common parent route.

---

## CSS Architecture

### Global variables (`global.css`)

All colors, spacing, and typography defined as CSS variables:

```css
:root {
  /* Colors */
  --color-bg: #0f1117;
  --color-surface: #1a1d27;
  --color-border: #2a2d3a;
  --color-text: #e2e8f0;
  --color-text-muted: #64748b;
  --color-accent: #6366f1;
  --color-success: #22c55e;
  --color-error: #ef4444;
  --color-highlight: rgba(255, 215, 0, 0.12); /* code line highlight */

  /* Spacing */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;

  /* Typography */
  --font-sans: "IBM Plex Sans", system-ui, sans-serif;
  --font-mono: "IBM Plex Mono", "Fira Code", monospace;
  --font-size-sm: 0.875rem;
  --font-size-base: 1rem;
  --font-size-lg: 1.125rem;
  --font-size-xl: 1.25rem;

  /* Borders */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
}
```

### Component styles

Each component has its own `.css` file using standard CSS classes.
No CSS-in-JS, no Tailwind.

```
components/token-answer/token-answer.css   → .token-answer, .token, .token--selected
components/code-block/code-block.css       → .code-block, [data-highlighted]
```

---

## Environment Variables

Qwik/Vite convention: variables prefixed with `PUBLIC_` are exposed to the client.

```bash
# .env
PUBLIC_API_URL=http://localhost:8080   # backend base URL, available client-side

# .env.production
PUBLIC_API_URL=https://api.yourdomain.com
```

Access in code:

```tsx
const apiUrl = import.meta.env.PUBLIC_API_URL;
```

---

## Error Handling

### Route-level errors

```tsx
// Any routeLoader$ or routeAction$ can return an error state
export const useDecks = routeLoader$(async ({ fail }) => {
    const res = await fetch(...);
    if (!res.ok) return fail(500, { message: 'Failed to load decks' });
    return res.json();
});

// In component — always check for failure
export default component$(() => {
    const decks = useDecks();
    if (decks.value.failed) {
        return <ErrorMessage message={decks.value.message} />;
    }
    return <DeckList decks={decks.value} />;
});
```

### Client-side errors (review session)

```tsx
const error = useSignal<string | null>(null);

const submitAnswer = $(async (submission: ReviewSubmission) => {
  try {
    const result = await api.review.submit(submission);
    session.results.push(result);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Something went wrong";
  }
});
```

---

## Key Conventions

- **No business logic in components** — components render and emit events only
- **No raw `fetch` in components** — always use `routeLoader$`, `routeAction$`, or `api.*`
- **Review submissions include interaction metadata** — final answer, wrong attempts, distractor clicks, incorrect tokens, and attempt history
- **No hardcoded user-facing text** — use translation dictionaries and locale helpers
- **Decks are language-aware** — forms and views must display and preserve `languageCode`
- **No `any` in TypeScript** — use types from `lib/types.ts`
- **No hardcoded colors or sizes** — always use CSS variables from `global.css`
- **No `window`/`document` outside `useVisibleTask$`**
- File names: `kebab-case`
- Component exports: named, `PascalCase`
- One component per file
