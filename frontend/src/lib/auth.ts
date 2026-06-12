import type { LanguageCode, User } from "./types";
import { api } from "./api";

const VALID_LOCALES: LanguageCode[] = ["en", "ru", "es", "de", "fr", "ja", "zh-CN"];

export function getToken(): string | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage.getItem("jwt");
}

export function setToken(token: string): void {
  localStorage.setItem("jwt", token);
}

export function clearAuth(): void {
  localStorage.removeItem("jwt");
  localStorage.removeItem("user");
}

export function getUser(): User | null {
  if (typeof localStorage === "undefined") return null;
  const raw = localStorage.getItem("user");
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}

export function setUser(user: User): void {
  localStorage.setItem("user", JSON.stringify(user));
}

export async function refreshUser(): Promise<User | null> {
  if (!getToken()) return null;
  const user = await api.auth.me();
  setUser(user);
  if (user.preferredLanguage) {
    setLocale(user.preferredLanguage);
  }
  return user;
}

export function isAdmin(user: User | null): boolean {
  return !!user?.isAdmin;
}

export function getLocale(): LanguageCode {
  if (typeof localStorage === "undefined") return "en";
  const lang = localStorage.getItem("preferredLanguage") as LanguageCode;
  if (lang && VALID_LOCALES.includes(lang)) return lang;
  return "en";
}

export function setLocale(locale: LanguageCode): void {
  localStorage.setItem("preferredLanguage", locale);
}
