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
