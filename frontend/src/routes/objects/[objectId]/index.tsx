import { $, component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useLocation, type DocumentHead } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t } from "~/lib/i18n";
import type { Card, InfoObjectDetail, LanguageCode, ReviewCard, ReviewResult, ReviewSubmission } from "~/lib/types";
import { CodeBlock } from "~/components/code-block";
import { CardReview } from "~/components/card-review";
import { CardRow } from "~/components/card-row";
import type { CardDisplayData } from "~/components/card-row";

export default component$(() => {
  const loc = useLocation();
  const locale = useSignal<LanguageCode>("en");
  const obj = useSignal<InfoObjectDetail | null>(null);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const testCard = useSignal<ReviewCard | null>(null);
  const testResult = useSignal<boolean | null>(null);

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

  const mockSubmit$ = $(async (submission: ReviewSubmission): Promise<ReviewResult> => {
    const easy = submission.wrongAttemptsCount === 0 && submission.distractorClicksCount === 0;
    return {
      state: testCard.value!.state,
      rating: easy ? 4 : 1,
      wasCorrect: true,
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
          <div class="card w-full max-w-2xl max-h-[90vh] overflow-y-auto border border-base-300 bg-base-100 shadow-xl">
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
          <Link class="link link-hover text-sm text-base-content/60" href={`/decks/${o.deckId}`}>
            ← {t(locale.value, "object.back")}
          </Link>
          <div class="flex items-center gap-3">
            <h1 class="text-3xl font-semibold">{o.title}</h1>
            <span class="badge badge-ghost badge-sm">{o.contentType}</span>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-ghost btn-sm text-error" onClick$={handleDeleteObject$} type="button">
            {t(locale.value, "object.delete")}
          </button>
        </div>
      </div>

      {/* Reference content */}
      <div class="rounded-box border border-base-300 overflow-hidden">
        <CodeBlock code={o.content} contentType={o.contentType} />
      </div>

      {/* Cards section */}
      <h2 class="text-xl font-semibold">{t(locale.value, "object.cards")}</h2>

      {o.cards.length === 0 ? (
        <p class="text-center py-8 text-base-content/60">{t(locale.value, "object.empty")}</p>
      ) : (
        <div class="flex flex-col gap-3">
          {o.cards.map((card, cardIndex) => (
            <CardRow
              key={card.id}
              card={card}
              locale={locale.value}
              onTest$={$(() => {
                if (!obj.value) return;
                const rc: ReviewCard = {
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
                testCard.value = rc;
                testResult.value = null;
              })}
              onDelete$={$(async () => {
                if (!confirm(t(locale.value, "object.card.delete") + "?")) return;
                await api.cards.delete(card.id);
                if (obj.value) {
                  obj.value = { ...obj.value, cards: obj.value.cards.filter((c) => c.id !== card.id) };
                }
              })}
              onEdit$={$(async (updated: CardDisplayData) => {
                const saved = await api.cards.update(
                  card.id, updated.front, updated.cardType, updated.step,
                  updated.correctAnswers, updated.distractors,
                );
                if (obj.value) {
                  obj.value = {
                    ...obj.value,
                    cards: obj.value.cards.map((c) => c.id === card.id ? { ...c, ...saved } : c),
                  };
                }
              })}
              onAddBefore$={$(async (data: CardDisplayData) => {
                const newCard = await api.cards.create(
                  loc.params["objectId"], data.front, data.cardType, data.step,
                  data.correctAnswers, data.distractors,
                );
                if (obj.value) {
                  const cards = [...obj.value.cards];
                  cards.splice(cardIndex, 0, newCard as Card);
                  obj.value = { ...obj.value, cards };
                }
              })}
              onAddAfter$={$(async (data: CardDisplayData) => {
                const newCard = await api.cards.create(
                  loc.params["objectId"], data.front, data.cardType, data.step,
                  data.correctAnswers, data.distractors,
                );
                if (obj.value) {
                  const cards = [...obj.value.cards];
                  cards.splice(cardIndex + 1, 0, newCard as Card);
                  obj.value = { ...obj.value, cards };
                }
              })}
              onGenerateCard$={$(async (prompt: string) => {
                const result = await api.generate.suggestCard({
                  objectId: loc.params["objectId"],
                  prompt,
                });
                return result;
              })}
            />
          ))}
        </div>
      )}
    </main>
  );
});

export const head: DocumentHead = {
  title: "Topic — Shmanki",
};
