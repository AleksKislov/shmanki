import { $, component$ } from "@builder.io/qwik";
import type { ReviewCard, ReviewResult, ReviewSubmission } from "~/lib/types";
import { CodeBlock } from "~/components/code-block";
import { TokenAnswer } from "~/components/token-answer";
import { MasteryBadge } from "~/components/mastery-badge";
import { ProgressBar } from "~/components/progress-bar";

interface Props {
  card: ReviewCard;
  index: number;
  total: number;
  locale: string;
  onAnswer$: (result: ReviewResult) => void;
}

export const CardReview = component$<Props>(({ card, index, total, locale, onAnswer$ }) => {
  const handleSubmit$ = $(async (submission: ReviewSubmission) => {
    const { api } = await import("~/lib/api");
    const result = await api.review.submit(submission);
    onAnswer$(result);
  });

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
      <div class="rounded-box border border-base-300 overflow-hidden">
        <CodeBlock
          code={card.content}
          contentType={card.contentType}
          highlightLines={card.highlightLines}
        />
      </div>

      {/* Question */}
      <h2 class="text-xl font-semibold">{card.front}</h2>

      {/* Token answer */}
      <TokenAnswer
        correctAnswers={card.correctAnswers}
        distractors={card.distractors}
        cardId={card.cardId}
        locale={locale}
        onSubmit$={handleSubmit$}
      />
    </div>
  );
});
