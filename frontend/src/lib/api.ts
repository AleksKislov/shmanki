import type {
  AuthResponse,
  Card,
  CardType,
  Deck,
  DeckDetail,
  DeckStats,
  GenerateSaveRequest,
  GenerateEditRequest,
  GenerateSuggestRequest,
  GenerateSuggestResponse,
  InfoObject,
  InfoObjectDetail,
  LanguageCode,
  PremadeDeck,
  PremadeDeckDetail,
  ReviewResult,
  ReviewSubmission,
} from "./types";

const BASE_URL = import.meta.env.PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = typeof localStorage !== "undefined" ? localStorage.getItem("jwt") : null;
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
  return res.json() as Promise<T>;
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      request<AuthResponse>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      }),
    register: (email: string, password: string, preferredLanguage: LanguageCode) =>
      request<AuthResponse>("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({ email, password, preferredLanguage }),
      }),
    me: () => request<import("./types").User>("/api/v1/auth/me"),
  },
  users: {
    updatePreferredLanguage: (preferredLanguage: LanguageCode) =>
      request<{ status: string }>("/api/v1/users/me/language", {
        method: "PATCH",
        body: JSON.stringify({ preferredLanguage }),
      }),
    updateProfile: (displayName: string, preferredLanguage?: LanguageCode) =>
      request<import("./types").User>("/api/v1/users/me", {
        method: "PUT",
        body: JSON.stringify({ displayName, preferredLanguage }),
      }),
  },
  decks: {
    list: () => request<Deck[]>("/api/v1/decks"),
    create: (title: string, description: string, languageCode: LanguageCode) =>
      request<Deck>("/api/v1/decks", {
        method: "POST",
        body: JSON.stringify({ title, description, languageCode }),
      }),
    get: (id: string) => request<DeckDetail>(`/api/v1/decks/${id}`),
    update: (id: string, title: string, description: string, languageCode: LanguageCode) =>
      request<Deck>(`/api/v1/decks/${id}`, {
        method: "PUT",
        body: JSON.stringify({ title, description, languageCode }),
      }),
    delete: (id: string) => request<{ status: string }>(`/api/v1/decks/${id}`, { method: "DELETE" }),
    stats: (id: string) => request<DeckStats>(`/api/v1/stats/deck/${id}`),
  },
  premade: {
    list: (params?: {
      source?: "official" | "community" | "all";
      category?: string;
      language?: LanguageCode | "";
      minRating?: number;
      sort?: "rating" | "newest" | "popular";
    }) => {
      const search = new URLSearchParams();
      if (params?.source && params.source !== "all") search.set("source", params.source);
      if (params?.category) search.set("category", params.category);
      if (params?.language) search.set("language", params.language);
      if (params?.minRating) search.set("min_rating", String(params.minRating));
      if (params?.sort) search.set("sort", params.sort);
      const query = search.toString();
      return request<PremadeDeck[]>(`/api/v1/premade-decks${query ? `?${query}` : ""}`);
    },
    categories: () => request<string[]>("/api/v1/premade-decks/categories"),
    get: (id: string) => request<PremadeDeckDetail>(`/api/v1/premade-decks/${id}`),
    clone: (id: string) => request<{ deckId: string }>(`/api/v1/premade-decks/${id}/clone`, { method: "POST" }),
    rate: (id: string, score: number) =>
      request<{ status: string }>(`/api/v1/premade-decks/${id}/rating`, {
        method: "PUT",
        body: JSON.stringify({ score }),
      }),
    unrate: (id: string) => request<{ status: string }>(`/api/v1/premade-decks/${id}/rating`, { method: "DELETE" }),
    publishDeck: (deckId: string, category: string, title?: string, description?: string) =>
      request<{ premadeDeckId: string }>(`/api/v1/decks/${deckId}/publish`, {
        method: "POST",
        body: JSON.stringify({ category, title, description }),
      }),
    delete: (id: string) => request<{ status: string }>(`/api/v1/premade-decks/${id}`, { method: "DELETE" }),
  },
  admin: {
    premade: {
      list: (source?: "official" | "community" | "all") => {
        const query = source && source !== "all" ? `?source=${source}` : "";
        return request<PremadeDeck[]>(`/api/v1/admin/premade-decks${query}`);
      },
      createOfficialFromDeck: (deckId: string, category: string, title?: string, description?: string) =>
        request<{ premadeDeckId: string }>("/api/v1/admin/premade-decks/from-deck", {
          method: "POST",
          body: JSON.stringify({ deckId, category, title, description }),
        }),
      setPublished: (id: string, isPublished: boolean) =>
        request<{ status: string }>(`/api/v1/admin/premade-decks/${id}/publish`, {
          method: "PATCH",
          body: JSON.stringify({ isPublished }),
        }),
      delete: (id: string) => request<{ status: string }>(`/api/v1/admin/premade-decks/${id}`, { method: "DELETE" }),
    },
  },
  objects: {
    list: (deckId: string) => request<InfoObject[]>(`/api/v1/decks/${deckId}/objects`),
    create: (
      deckId: string,
      title: string,
      content: string,
      discipline: string,
      contentType: string,
    ) =>
      request<InfoObject>(`/api/v1/decks/${deckId}/objects`, {
        method: "POST",
        body: JSON.stringify({ title, content, discipline, contentType }),
      }),
    get: (id: string) => request<InfoObjectDetail>(`/api/v1/objects/${id}`),
    update: (
      id: string,
      title: string,
      content: string,
      discipline: string,
      contentType: string,
    ) =>
      request<InfoObject>(`/api/v1/objects/${id}`, {
        method: "PUT",
        body: JSON.stringify({ title, content, discipline, contentType }),
      }),
    delete: (id: string) => request<{ status: string }>(`/api/v1/objects/${id}`, { method: "DELETE" }),
  },
  cards: {
    create: (
      objectId: string,
      front: string,
      cardType: CardType,
      step: number,
      correctAnswers: string[][],
      distractors: string[],
    ) =>
      request<Card>(`/api/v1/objects/${objectId}/cards`, {
        method: "POST",
        body: JSON.stringify({ front, cardType, step, correctAnswers, distractors }),
      }),
    update: (
      id: string,
      front: string,
      cardType: CardType,
      step: number,
      correctAnswers: string[][],
      distractors: string[],
    ) =>
      request<Card>(`/api/v1/cards/${id}`, {
        method: "PUT",
        body: JSON.stringify({ front, cardType, step, correctAnswers, distractors }),
      }),
    delete: (id: string) => request<{ status: string }>(`/api/v1/cards/${id}`, { method: "DELETE" }),
  },
  review: {
    getSession: (limit = 20) =>
      request<import("./types").ReviewCard[]>(`/api/v1/review/session?limit=${limit}`),
    submit: (input: ReviewSubmission) =>
      request<ReviewResult>("/api/v1/review/submit", {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
  generate: {
    suggest: (input: GenerateSuggestRequest) =>
      request<GenerateSuggestResponse>("/api/v1/generate/suggest", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    edit: (input: GenerateEditRequest) =>
      request<GenerateSuggestResponse>("/api/v1/generate/edit", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    save: (input: GenerateSaveRequest) =>
      request<{ infoObjects: InfoObjectDetail[] }>("/api/v1/generate/save", {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
};
