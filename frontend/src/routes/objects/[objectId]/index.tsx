import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useLocation, type DocumentHead } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t } from "~/lib/i18n";
import { getMasteryLevel } from "~/lib/fsrs";
import type { Card, InfoObjectDetail, LanguageCode } from "~/lib/types";
import { MasteryBadge } from "~/components/mastery-badge";
import { CodeBlock } from "~/components/code-block";

export default component$(() => {
  const loc = useLocation();
  const locale = useSignal<LanguageCode>("en");
  const obj = useSignal<InfoObjectDetail | null>(null);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const showCardForm = useSignal(false);

  const cardForm = useStore({
    front: "",
    step: 0,
    correctAnswersRaw: "",
    distractorsRaw: "",
    highlightLinesRaw: "",
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
      const highlightLines = cardForm.highlightLinesRaw
        .split(",")
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => !isNaN(n));

      const card = await api.cards.create(
        loc.params["objectId"],
        cardForm.front,
        cardForm.step,
        correctAnswers,
        distractors,
        highlightLines,
      );
      if (obj.value) {
        obj.value = { ...obj.value, cards: [...obj.value.cards, card] };
      }
      showCardForm.value = false;
      cardForm.front = "";
      cardForm.correctAnswersRaw = "";
      cardForm.distractorsRaw = "";
      cardForm.highlightLinesRaw = "";
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
        <CodeBlock code={o.content} contentType={o.contentType} highlightLines={[]} />
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
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "object.form.front")}</span>
              </div>
              <input
                class="input input-bordered"
                type="text"
                value={cardForm.front}
                onInput$={(_, el) => (cardForm.front = el.value)}
              />
            </label>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "object.form.step")}</span>
                </div>
                <input
                  class="input input-bordered"
                  type="number"
                  min={0}
                  value={cardForm.step}
                  onInput$={(_, el) => (cardForm.step = parseInt(el.value, 10) || 0)}
                />
              </label>
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "object.form.highlightLines")}</span>
                </div>
                <input
                  class="input input-bordered"
                  type="text"
                  value={cardForm.highlightLinesRaw}
                  onInput$={(_, el) => (cardForm.highlightLinesRaw = el.value)}
                  placeholder="1, 5, 6"
                />
              </label>
            </div>
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "object.form.correctAnswers")}</span>
              </div>
              <input
                class="input input-bordered"
                type="text"
                value={cardForm.correctAnswersRaw}
                onInput$={(_, el) => (cardForm.correctAnswersRaw = el.value)}
                placeholder="go, worker()"
              />
            </label>
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "object.form.distractors")}</span>
              </div>
              <input
                class="input input-bordered"
                type="text"
                value={cardForm.distractorsRaw}
                onInput$={(_, el) => (cardForm.distractorsRaw = el.value)}
                placeholder="defer, func, chan"
              />
            </label>
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
  onDelete$: (id: string) => void;
}

export const CardRow = component$<CardRowProps>(({ card, locale, onDelete$ }) => {
  return (
    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-3 py-4">
        <div class="flex items-start justify-between gap-2">
          <div class="flex flex-col gap-1 min-w-0">
            <p class="font-medium leading-snug">{card.front}</p>
            <div class="flex flex-wrap gap-2 text-sm text-base-content/60">
              <span>
                {t(locale, "object.card.step")} {card.step}
              </span>
              <span>→ {card.correctAnswers[0]?.join(" ")}</span>
            </div>
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
          <button
            class="btn btn-ghost btn-xs text-error shrink-0"
            onClick$={() => onDelete$(card.id)}
            type="button"
          >
            {t(locale, "common.delete")}
          </button>
        </div>
      </div>
    </div>
  );
});

export const head: DocumentHead = {
  title: "Topic — Shmanki",
};
