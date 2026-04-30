import type { LanguageCode, User } from "./types";

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

export function getLocale(): LanguageCode {
  if (typeof localStorage === "undefined") return "en";
  const lang = localStorage.getItem("preferredLanguage");
  if (lang === "ru") return "ru";
  return "en";
}

export function setLocale(locale: LanguageCode): void {
  localStorage.setItem("preferredLanguage", locale);
}
