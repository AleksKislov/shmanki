import type { CardType, LanguageCode } from "~/lib/types";
import { t } from "~/lib/i18n";

export function isBlockInteraction(cardType: CardType) {
  return ["line_order", "choose_snippet", "fix_bug"].includes(cardType);
}

export function getCardTypeLabel(locale: LanguageCode, cardType: CardType) {
  return t(locale, `cardType.${cardType}`);
}
