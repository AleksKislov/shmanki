import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import type { QRL } from "@builder.io/qwik";
import { Link, useLocation, type DocumentHead } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t } from "~/lib/i18n";
import type { Card, CardType, InfoObjectDetail, LanguageCode, ReviewCard, ReviewResult, ReviewSubmission } from "~/lib/types";
import { CodeBlock } from "~/components/code-block";
import { CardReview } from "~/components/card-review";
import { getCardTypeLabel, isBlockInteraction } from "~/lib/card-types";

export default component$(() => {
  const loc = useLocation();
  const locale = useSignal<LanguageCode>("en");
  const obj = useSignal<InfoObjectDetail | null>(null);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const showCardForm = useSignal(false);
  const testCard = useSignal<ReviewCard | null>(null);
  const testResult = useSignal<boolean | null>(null);

  const cardForm = useStore({
    front: "",
    cardType: "concept" as CardType,
    step: 0,
    correctAnswersRaw: "",
    distractorsRaw: "",
    submitting: false,
    error: null as string | null,
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    try {
      obj.value = await api.objects.get(loc.params["objectId"]);
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      loading.value = false;
    }
  });

  const handleAddCard$ = $(async () => {
    cardForm.error = null;
    cardForm.submitting = true;
    try {
      const correctAnswers = [
        cardForm.correctAnswersRaw.split(",").map((s) => s.trim()).filter(Boolean),
      ];
      const distractors = cardForm.distractorsRaw.split(",").map((s) => s.trim()).filter(Boolean);

      const card = await api.cards.create(
        loc.params["objectId"],
        cardForm.front,
        cardForm.cardType,
        cardForm.step,
        correctAnswers,
        distractors,
      );
      if (obj.value) {
        obj.value = { ...obj.value, cards: [...obj.value.cards, card] };
      }
      showCardForm.value = false;
      cardForm.front = "";
      cardForm.cardType = "concept";
      cardForm.correctAnswersRaw = "";
      cardForm.distractorsRaw = "";
      cardForm.step = 0;
    } catch (e) {
      cardForm.error = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      cardForm.submitting = false;
    }
  });

  const handleDeleteCard$ = $(async (id: string) => {
    if (!confirm(t(locale.value, "object.card.delete") + "?")) return;
    try {
      await api.cards.delete(id);
      if (obj.value) {
        obj.value = { ...obj.value, cards: obj.value.cards.filter((c) => c.id !== id) };
      }
    } catch {
      // ignore
    }
  });

  const handleDeleteObject$ = $(async () => {
    if (!obj.value) return;
    if (!confirm(t(locale.value, "object.delete.confirm"))) return;
    try {
      await api.objects.delete(obj.value.id);
      history.back();
    } catch {
      // ignore
    }
  });

  const handleTestCard$ = $((card: Card) => {
    if (!obj.value) return;
    const reviewCard: ReviewCard = {
      cardId: card.id,
      front: card.front,
      cardType: card.cardType,
      correctAnswers: card.correctAnswers,
      distractors: card.distractors,
      step: card.step,
      content: obj.value.content,
      contentType: obj.value.contentType,
      languageCode: locale.value,
      infoObjectId: card.infoObjectId,
      state: {
        cardId: card.id,
        stability: 0,
        difficulty: 5,
        effectiveDifficulty: 5,
        hierarchicalSupport: 1,
        retrievability: 0,
        dueDate: null,
        status: "new",
        reps: 0,
        lapses: 0,
        intervalDays: 0,
        learningStep: 0,
        lastReview: null,
      },
    };
    testCard.value = reviewCard;
    testResult.value = null;
  });

  const mockSubmit$ = $(async (submission: ReviewSubmission): Promise<ReviewResult> => {
    const wasCorrect = submission.wrongAttemptsCount === 0 && submission.distractorClicksCount === 0;
    return {
      state: testCard.value!.state,
      rating: wasCorrect ? 4 : 1,
      wasCorrect,
    };
  });

  if (loading.value) {
    return (
      <div class="flex justify-center py-20">
        <span class="loading loading-spinner loading-lg" />
      </div>
    );
  }

  if (error.value || !obj.value) {
    return (
      <div class="alert alert-error">
        <span>{error.value ?? t(locale.value, "common.error")}</span>
      </div>
    );
  }

  const o = obj.value;

  return (
    <main class="flex flex-col gap-6">
      {/* Test card modal */}
      {testCard.value && (
        <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div class="card w-full max-w-2xl border border-base-300 bg-base-100 shadow-xl">
            <div class="card-body gap-5">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-base-content/60">
                  {t(locale.value, "object.card.testMode")}
                </span>
                <button
                  class="btn btn-ghost btn-sm"
                  onClick$={() => { testCard.value = null; testResult.value = null; }}
                  type="button"
                >
                  ✕
                </button>
              </div>
              {testResult.value === null ? (
                <CardReview
                  card={testCard.value}
                  index={0}
                  total={1}
                  locale={locale.value}
                  onAnswer$={$((result: ReviewResult) => { testResult.value = result.wasCorrect; })}
                  onSubmit$={mockSubmit$}
                  showContent={false}
                />
              ) : (
                <div class="flex flex-col items-center gap-5 py-4">
                  <div class={testResult.value ? "alert alert-success text-lg font-semibold" : "alert alert-error text-lg font-semibold"}>
                    {testResult.value
                      ? t(locale.value, "object.card.testCorrect")
                      : t(locale.value, "object.card.testWrong")}
                  </div>
                  <button
                    class="btn btn-primary"
                    onClick$={() => { testCard.value = null; testResult.value = null; }}
                    type="button"
                  >
                    {t(locale.value, "object.form.cancel")}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
      {/* Header */}
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="flex flex-col gap-1">
          <Link
            class="link link-hover text-sm text-base-content/60"
            href={`/decks/${o.deckId}`}
          >
            ← {t(locale.value, "object.back")}
          </Link>
          <div class="flex items-center gap-3">
            <h1 class="text-3xl font-semibold">{o.title}</h1>
            <span class="badge badge-ghost badge-sm">{o.contentType}</span>
          </div>
        </div>
        <div class="flex gap-2">
          <button
            class="btn btn-ghost btn-sm text-error"
            onClick$={handleDeleteObject$}
            type="button"
          >
            {t(locale.value, "object.delete")}
          </button>
        </div>
      </div>

      {/* Reference content */}
      <div class="rounded-box border border-base-300 overflow-hidden">
        <CodeBlock code={o.content} contentType={o.contentType} />
      </div>

      {/* Cards section */}
      <div class="flex items-center justify-between gap-4">
        <h2 class="text-xl font-semibold">{t(locale.value, "object.cards")}</h2>
        <button
          class="btn btn-outline btn-sm"
          onClick$={() => (showCardForm.value = !showCardForm.value)}
          type="button"
        >
          {t(locale.value, "object.addCard")}
        </button>
      </div>

      {showCardForm.value && (
        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            {cardForm.error && (
              <div class="alert alert-error">
                <span>{cardForm.error}</span>
              </div>
            )}
            <fieldset class="fieldset">
              <legend class="fieldset-legend">{t(locale.value, "object.form.front")}</legend>
              <input
                class="input input-bordered w-full"
                type="text"
                value={cardForm.front}
                onInput$={(_, el) => (cardForm.front = el.value)}
              />
            </fieldset>
            <div class="grid gap-4 sm:grid-cols-2">
              <fieldset class="fieldset">
                <legend class="fieldset-legend">{t(locale.value, "object.form.cardType")}</legend>
                <select
                  class="select select-bordered w-full"
                  value={cardForm.cardType}
                  onChange$={(_, el) => (cardForm.cardType = el.value as CardType)}
                >
                  {[
                    "concept",
                    "signature",
                    "trace",
                    "line_order",
                    "choose_snippet",
                    "fix_bug",
                  ].map((cardType) => (
                    <option key={cardType} value={cardType}>
                      {getCardTypeLabel(locale.value, cardType as CardType)}
                    </option>
                  ))}
                </select>
              </fieldset>
              <fieldset class="fieldset">
                <legend class="fieldset-legend">{t(locale.value, "object.form.step")}</legend>
                <input
                  class="input input-bordered w-full"
                  type="number"
                  min={0}
                  value={cardForm.step}
                  onInput$={(_, el) => (cardForm.step = parseInt(el.value, 10) || 0)}
                />
              </fieldset>
            </div>
            <fieldset class="fieldset">
              <legend class="fieldset-legend">{t(locale.value, "object.form.correctAnswers")}</legend>
              <input
                class="input input-bordered w-full"
                type="text"
                value={cardForm.correctAnswersRaw}
                onInput$={(_, el) => (cardForm.correctAnswersRaw = el.value)}
                placeholder="go, worker()"
              />
            </fieldset>
            <fieldset class="fieldset">
              <legend class="fieldset-legend">{t(locale.value, "object.form.distractors")}</legend>
              <input
                class="input input-bordered w-full"
                type="text"
                value={cardForm.distractorsRaw}
                onInput$={(_, el) => (cardForm.distractorsRaw = el.value)}
                placeholder="defer, func, chan"
              />
            </fieldset>
            <div class="flex gap-2">
              <button
                class="btn btn-primary btn-sm"
                onClick$={handleAddCard$}
                disabled={cardForm.submitting || !cardForm.front.trim()}
                type="button"
              >
                {cardForm.submitting ? (
                  <span class="loading loading-spinner loading-xs" />
                ) : (
                  t(locale.value, "object.form.submit")
                )}
              </button>
              <button
                class="btn btn-ghost btn-sm"
                onClick$={() => (showCardForm.value = false)}
                type="button"
              >
                {t(locale.value, "object.form.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {o.cards.length === 0 ? (
        <p class="text-center py-8 text-base-content/60">{t(locale.value, "object.empty")}</p>
      ) : (
        <div class="flex flex-col gap-3">
          {o.cards.map((card) => (
            <CardRow
              key={card.id}
              card={card}
              locale={locale.value}
              onDelete$={handleDeleteCard$}
              onTest$={handleTestCard$}
            />
          ))}
        </div>
      )}
    </main>
  );
});

interface CardRowProps {
  card: Card;
  locale: LanguageCode;
  onDelete$: QRL<(id: string) => void>;
  onTest$: QRL<(card: Card) => void>;
}

export const CardRow = component$<CardRowProps>(({ card, locale, onDelete$, onTest$ }) => {
  return (
    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-3 py-4">
        <div class="flex items-start justify-between gap-2">
          <div class="flex flex-col gap-1 min-w-0">
            <p class="font-medium leading-snug">{card.front}</p>
            <div class="flex flex-wrap gap-2 text-sm text-base-content/60">
              <span>
                {t(locale, "object.card.type")}: {getCardTypeLabel(locale, card.cardType)}
              </span>
              <span>
                {t(locale, "object.card.step")} {card.step}
              </span>
              <span>→ {card.correctAnswers[0]?.join(" ")}</span>
            </div>
            {isBlockInteraction(card.cardType) && (
              <div class="mt-2 flex flex-col gap-2">
                {card.correctAnswers[0]?.map((unit, index) => (
                  <pre
                    key={index}
                    class="rounded-box border border-base-300 bg-base-200/60 p-2 text-xs whitespace-pre-wrap break-words"
                  >
                    {unit}
                  </pre>
                ))}
              </div>
            )}
            {card.distractors.length > 0 && (
              <div class="flex flex-wrap gap-1 mt-1">
                {card.distractors.map((d) => (
                  <span key={d} class="badge badge-ghost badge-xs">
                    {d}
                  </span>
                ))}
              </div>
            )}
          </div>
          <div class="flex shrink-0 gap-1">
            <button
              class="btn btn-ghost btn-xs"
              onClick$={() => onTest$(card)}
              type="button"
            >
              {t(locale, "object.card.test")}
            </button>
            <button
              class="btn btn-ghost btn-xs text-error"
              onClick$={() => onDelete$(card.id)}
              type="button"
            >
              {t(locale, "common.delete")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
});

export const head: DocumentHead = {
  title: "Topic — Shmanki",
};
