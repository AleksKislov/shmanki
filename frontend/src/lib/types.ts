export type CardStatus = "locked" | "new" | "learning" | "review" | "relearning";
export type Rating = 1 | 2 | 3 | 4;
export type LanguageCode = "en" | "ru" | "es" | "de" | "fr" | "ja" | "zh-CN";
export type ContentType = string;
export type MasteryLevel = "new" | "learning" | "learned" | "mastered" | "expert";
export type CardType =
  | "concept"
  | "signature"
  | "trace"
  | "line_order"
  | "choose_snippet"
  | "fix_bug";

export type Messages = Record<string, string>;

export interface User {
  id: string;
  email: string;
  preferredLanguage: LanguageCode;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface Deck {
  id: string;
  title: string;
  description: string;
  languageCode: LanguageCode;
  createdAt: string;
  updatedAt: string;
}

export interface DeckSummary {
  id: string;
  title: string;
  discipline: string;
  contentType: ContentType;
}

export interface DeckDetail extends Deck {
  infoObjects: DeckSummary[];
}

export interface InfoObject {
  id: string;
  deckId: string;
  title: string;
  content: string;
  discipline: string;
  contentType: ContentType;
  createdAt: string;
  updatedAt: string;
}

export interface Card {
  id: string;
  infoObjectId: string;
  front: string;
  cardType: CardType;
  step: number;
  correctAnswers: string[][];
  distractors: string[];
  createdAt: string;
  updatedAt: string;
}

export interface InfoObjectDetail extends InfoObject {
  cards: Card[];
}

export interface CardState {
  cardId: string;
  stability: number;
  difficulty: number;
  effectiveDifficulty: number;
  hierarchicalSupport: number;
  retrievability: number;
  dueDate: string | null;
  status: CardStatus;
  reps: number;
  lapses: number;
  intervalDays: number;
  lastReview?: string | null;
}

export interface ReviewCard {
  cardId: string;
  front: string;
  cardType: CardType;
  correctAnswers: string[][];
  distractors: string[];
  step: number;
  content: string;
  contentType: ContentType;
  languageCode: LanguageCode;
  infoObjectId: string;
  state: CardState;
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

export interface DeckStats {
  deckId: string;
  levels: {
    new: number;
    learning: number;
    learned: number;
    mastered: number;
    expert: number;
  };
  infoObjects: number;
  cards: number;
  dueNow: number;
  newNow: number;
}

export interface GenerateSuggestRequest {
  deckId: string;
  prompt: string;
  discipline?: string;
  contentType?: ContentType;
}

export interface GeneratedCard {
  front: string;
  cardType: CardType;
  step: number;
  correctAnswers: string[][];
  distractors: string[];
}

export interface GeneratedInfoObject {
  title: string;
  content: string;
  discipline: string;
  contentType: ContentType;
  cards: GeneratedCard[];
}

export interface GenerateSuggestResponse {
  generationId: string;
  model: string;
  infoObjects: GeneratedInfoObject[];
}

export interface GenerateEditRequest {
  deckId: string;
  prompt: string;
  generationId: string;
  discipline?: string;
  contentType?: ContentType;
}

export interface GenerateSaveRequest {
  deckId: string;
  generationId: string;
}
