import { en } from "./locales/en";
import { ru } from "./locales/ru";
import type { LanguageCode, Messages } from "./types";

const messages: Record<string, Messages> = {
  en,
  ru,
};

export function t(locale: LanguageCode, key: string): string {
  return messages[locale]?.[key] ?? messages["en"][key] ?? key;
}

export function getLocaleLabel(code: LanguageCode): string {
  const labels: Record<LanguageCode, string> = {
    en: "English",
    ru: "Русский"
  };
  return labels[code] ?? code;
}

export const LANGUAGE_OPTIONS: LanguageCode[] = ["en", "ru"];
