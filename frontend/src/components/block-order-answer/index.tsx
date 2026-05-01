import { $, component$, useSignal } from "@builder.io/qwik";
import type { ReviewAttempt, ReviewSubmission } from "~/lib/types";

interface Props {
  correctAnswers: string[][];
  distractors: string[];
  cardId: string;
  locale: string;
  onSubmit$: (submission: ReviewSubmission) => void;
}

function shuffle<T>(items: T[]) {
  return [...items].sort(() => Math.random() - 0.5);
}

export const BlockOrderAnswer = component$<Props>(
  ({ correctAnswers, distractors, cardId, locale, onSubmit$ }) => {
    const expected = correctAnswers[0] ?? [];
    const pool = useSignal(shuffle([...expected, ...distractors]));
    const selected = useSignal<string[]>([]);
    const failed = useSignal(false);
    const attempts = useSignal<ReviewAttempt[]>([]);
    const wrongAttemptsCount = useSignal(0);
    const distractorClicksCount = useSignal(0);
    const incorrectTokensClicked = useSignal<string[]>([]);

    const handlePick$ = $((unit: string, index: number) => {
      selected.value = [...selected.value, unit];
      pool.value = pool.value.filter((_, i) => i !== index);
    });

    const resetSelection$ = $(() => {
      pool.value = shuffle([...expected, ...distractors]);
      selected.value = [];
      failed.value = false;
    });

    const handleUndo$ = $(() => {
      if (selected.value.length === 0) return;
      const last = selected.value[selected.value.length - 1];
      selected.value = selected.value.slice(0, -1);
      pool.value = [...pool.value, last];
    });

    const handleCheck$ = $(async () => {
      const answer = selected.value;
      const currentAttemptHasDistractor = answer.some((item) => distractors.includes(item));

      for (const item of answer) {
        if (distractors.includes(item)) {
          distractorClicksCount.value++;
          incorrectTokensClicked.value = [...incorrectTokensClicked.value, item];
        }
      }

      const isCorrect = correctAnswers.some(
        (candidate) =>
          candidate.length === answer.length && candidate.every((item, idx) => item === answer[idx]),
      );

      const nextAttempt: ReviewAttempt = {
        tokens: answer,
        hadDistractor: currentAttemptHasDistractor,
        wasCorrect: isCorrect,
      };
      const completedAttempts = [...attempts.value, nextAttempt];
      attempts.value = completedAttempts;

      if (!isCorrect) {
        failed.value = true;
        wrongAttemptsCount.value++;
        await resetSelection$();
        return;
      }

      await onSubmit$({
        cardId,
        answeredTokens: answer,
        attempts: completedAttempts,
        wrongAttemptsCount: wrongAttemptsCount.value,
        distractorClicksCount: distractorClicksCount.value,
        incorrectTokensClicked: incorrectTokensClicked.value,
      });
    });

    return (
      <div class="flex flex-col gap-4">
        <div class="min-h-24 rounded-box border border-base-300 bg-base-200 p-3 flex flex-col gap-2">
          {selected.value.length === 0 ? (
            <span class="text-base-content/40 text-sm italic">…</span>
          ) : (
            selected.value.map((unit, i) => (
              <div key={i} class="rounded-box border border-primary/30 bg-primary/10 p-3 text-sm font-mono whitespace-pre-wrap break-words">
                {unit}
              </div>
            ))
          )}
        </div>

        {failed.value && (
          <div class="alert alert-error py-2">
            <span class="text-sm">
              {locale === "ru" ? "Неверно — попробуйте ещё раз" : "Incorrect — try again"}
            </span>
            <button class="btn btn-xs btn-ghost ml-auto" onClick$={resetSelection$} type="button">
              {locale === "ru" ? "Сбросить" : "Clear"}
            </button>
          </div>
        )}

        <div class="flex flex-wrap gap-2">
          {pool.value.map((unit, i) => (
            <button
              key={`${i}-${unit}`}
              class="btn btn-outline h-auto min-h-0 whitespace-pre-wrap break-words px-3 py-2 text-left font-mono normal-case"
              onClick$={() => handlePick$(unit, i)}
              type="button"
            >
              {unit}
            </button>
          ))}
        </div>

        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm" onClick$={handleUndo$} type="button">
            {locale === "ru" ? "Назад" : "Undo"}
          </button>
          <button
            class="btn btn-primary btn-sm"
            disabled={selected.value.length === 0}
            onClick$={handleCheck$}
            type="button"
          >
            {locale === "ru" ? "Проверить" : "Check"}
          </button>
        </div>
      </div>
    );
  },
);
