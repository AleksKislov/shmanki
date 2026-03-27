import { en } from "./locales/en";
import { ru } from "./locales/ru";
import type { LanguageCode, Messages } from "./types";

const messages: Record<LanguageCode, Messages> = {
  en,
  ru,
};

export function t(locale: LanguageCode, key: string): string {
  return messages[locale]?.[key] ?? messages.en[key] ?? key;
}
