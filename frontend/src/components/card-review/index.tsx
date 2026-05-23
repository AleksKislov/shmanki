import { $, component$ } from "@builder.io/qwik";
import type { QRL } from "@builder.io/qwik";
import type { ReviewCard, ReviewResult, ReviewSubmission, LanguageCode } from "~/lib/types";
import { CodeBlock } from "~/components/code-block";
import { BlockOrderAnswer } from "~/components/block-order-answer";
import { TokenAnswer } from "~/components/token-answer";
import { MasteryBadge } from "~/components/mastery-badge";
import { ProgressBar } from "~/components/progress-bar";
import { isBlockInteraction } from "~/lib/card-types";
import { t } from "~/lib/i18n";

interface Props {
  card: ReviewCard;
  index: number;
  total: number;
  locale: LanguageCode;
  onAnswer$: QRL<(result: ReviewResult) => void>;
  onSubmit$?: QRL<(submission: ReviewSubmission) => Promise<ReviewResult>>;
  showContent?: boolean;
}

export const CardReview = component$<Props>(({ card, index, total, locale, onAnswer$, onSubmit$, showContent = true }) => {
  const handleSubmit$ = $(async (submission: ReviewSubmission) => {
    let result: ReviewResult;
    if (onSubmit$) {
      result = await onSubmit$(submission);
    } else {
      const { api } = await import("~/lib/api");
      result = await api.review.submit(submission);
    }
    onAnswer$(result);
  });

  const useBlockInteraction = isBlockInteraction(card.cardType);

  return (
    <div class="flex flex-col gap-5">
      {/* Session progress */}
      <div class="flex items-center justify-between gap-4">
        <span class="text-sm text-base-content/60">
          {index + 1} {locale === "ru" ? "из" : "of"} {total}
        </span>
        <div class="flex items-center gap-2">
          <MasteryBadge status={card.state.status} stability={card.state.stability} />
          <span class="text-xs text-base-content/50">
            S={card.state.stability.toFixed(1)}d R={Math.round(card.state.retrievability * 100)}%
          </span>
        </div>
      </div>

      <ProgressBar value={(index) / total} />

      {/* Code block */}
      {showContent && (
        <div class="rounded-box border border-base-300 overflow-hidden">
          <CodeBlock
            code={card.content}
            contentType={card.contentType}
          />
        </div>
      )}

      {/* Question */}
      <div class="flex flex-col gap-2">
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <span class="badge badge-outline">{t(locale, `cardType.${card.cardType}`)}</span>
          <span class="badge badge-ghost">
            {t(locale, "object.card.step")} {card.step}
          </span>
        </div>
        <h2 class="text-xl font-semibold">{card.front}</h2>
        <p class="text-sm text-base-content/60">
          {useBlockInteraction
            ? t(locale, "review.answer.instructions.blocks")
            : t(locale, "review.answer.instructions.tokens")}
        </p>
      </div>

      {useBlockInteraction ? (
        <BlockOrderAnswer
          correctAnswers={card.correctAnswers}
          distractors={card.distractors}
          cardId={card.cardId}
          locale={locale}
          onSubmit$={handleSubmit$}
        />
      ) : (
        <TokenAnswer
          correctAnswers={card.correctAnswers}
          distractors={card.distractors}
          cardId={card.cardId}
          locale={locale}
          onSubmit$={handleSubmit$}
        />
      )}
    </div>
  );
});
