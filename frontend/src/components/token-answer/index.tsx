import { $, component$, useSignal } from "@builder.io/qwik";
import type { QRL } from "@builder.io/qwik";
import type { ReviewAttempt, ReviewSubmission } from "~/lib/types";

interface Props {
  correctAnswers: string[][];
  distractors: string[];
  cardId: string;
  locale: string;
  onSubmit$: QRL<(submission: ReviewSubmission) => void>;
}

export const TokenAnswer = component$<Props>(
  ({ correctAnswers, distractors, cardId, locale, onSubmit$ }) => {
    const allTokens = [...(correctAnswers[0] ?? []), ...distractors].sort(() => Math.random() - 0.5);

    const clicked = useSignal<string[]>([]);
    const failed = useSignal(false);
    const attempts = useSignal<ReviewAttempt[]>([]);
    const wrongAttemptsCount = useSignal(0);
    const distractorClicksCount = useSignal(0);
    const incorrectTokensClicked = useSignal<string[]>([]);

    const handleToken$ = $((token: string) => {
      const next = [...clicked.value, token];
      clicked.value = next;

      const expected = correctAnswers[0];
      const currentAttemptHasDistractor = next.some((t) => distractors.includes(t));

      if (distractors.includes(token)) {
        distractorClicksCount.value++;
        incorrectTokensClicked.value = [...incorrectTokensClicked.value, token];
      }

      // Check if still on a valid path
      for (let i = 0; i < next.length; i++) {
        if (next[i] !== expected[i]) {
          failed.value = true;
          wrongAttemptsCount.value++;
          attempts.value = [
            ...attempts.value,
            { tokens: next, hadDistractor: currentAttemptHasDistractor, wasCorrect: false },
          ];
          clicked.value = [];
          return;
        }
      }

      // Check if complete
      if (next.length === expected.length) {
        const completedAttempts = [
          ...attempts.value,
          { tokens: next, hadDistractor: currentAttemptHasDistractor, wasCorrect: true },
        ];
        attempts.value = completedAttempts;
        onSubmit$({
          cardId,
          answeredTokens: next,
          attempts: completedAttempts,
          wrongAttemptsCount: wrongAttemptsCount.value,
          distractorClicksCount: distractorClicksCount.value,
          incorrectTokensClicked: incorrectTokensClicked.value,
        });
      }
    });

    const handleClear$ = $(() => {
      clicked.value = [];
      failed.value = false;
    });

    return (
      <div class="flex flex-col gap-4">
        {/* Answer so far */}
        <div class="min-h-10 rounded-box border border-base-300 bg-base-200 px-4 py-2 flex items-center gap-2 flex-wrap">
          {clicked.value.length === 0 ? (
            <span class="text-base-content/40 text-sm italic">…</span>
          ) : (
            clicked.value.map((tok, i) => (
              <span key={i} class="badge badge-primary badge-lg">
                {tok}
              </span>
            ))
          )}
        </div>

        {failed.value && (
          <div class="alert alert-error py-2">
            <span class="text-sm">
              {locale === "ru" ? "Неверно — попробуйте ещё раз" : "Incorrect — try again"}
            </span>
            <button class="btn btn-xs btn-ghost ml-auto" onClick$={handleClear$} type="button">
              {locale === "ru" ? "Сбросить" : "Clear"}
            </button>
          </div>
        )}

        {/* Token pool */}
        <div class="flex flex-wrap gap-2">
          {allTokens.map((token, i) => (
            <button
              key={i}
              class={{
                "btn btn-sm": true,
                "btn-primary": clicked.value.includes(token),
                "btn-outline": !clicked.value.includes(token),
              }}
              onClick$={() => handleToken$(token)}
              type="button"
            >
              {token}
            </button>
          ))}
        </div>
      </div>
    );
  },
);
