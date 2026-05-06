import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link, type DocumentHead } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t } from "~/lib/i18n";
import type { LanguageCode, ReviewCard, ReviewResult } from "~/lib/types";
import { CardReview } from "~/components/card-review";

interface SessionState {
  queue: ReviewCard[];
  currentIndex: number;
  results: ReviewResult[];
  status: "loading" | "active" | "complete" | "empty" | "error";
  errorMsg: string | null;
}

export default component$(() => {
  const locale = useSignal<LanguageCode>("en");
  const session = useStore<SessionState>({
    queue: [],
    currentIndex: 0,
    results: [],
    status: "loading",
    errorMsg: null,
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    try {
      const cards = await api.review.getSession(20);
      if (cards.length === 0) {
        session.status = "empty";
      } else {
        session.queue = cards;
        session.status = "active";
      }
    } catch (e) {
      session.errorMsg = e instanceof Error ? e.message : t(locale.value, "review.error");
      session.status = "error";
    }
  });

  const handleAnswer$ = $((result: ReviewResult) => {
    session.results = [...session.results, result];
    const next = session.currentIndex + 1;
    if (next >= session.queue.length) {
      session.status = "complete";
    } else {
      session.currentIndex = next;
    }
  });

  const handleRestart$ = $(async () => {
    session.status = "loading";
    session.currentIndex = 0;
    session.results = [];
    session.errorMsg = null;
    try {
      const cards = await api.review.getSession(20);
      if (cards.length === 0) {
        session.status = "empty";
      } else {
        session.queue = cards;
        session.status = "active";
      }
    } catch (e) {
      session.errorMsg = e instanceof Error ? e.message : t(locale.value, "review.error");
      session.status = "error";
    }
  });

  if (session.status === "loading") {
    return (
      <div class="flex justify-center py-20">
        <span class="loading loading-spinner loading-lg" />
      </div>
    );
  }

  if (session.status === "error") {
    return (
      <div class="flex flex-col gap-4 items-center py-12">
        <div class="alert alert-error max-w-md">
          <span>{session.errorMsg}</span>
        </div>
        <button class="btn btn-primary" onClick$={handleRestart$} type="button">
          {t(locale.value, "common.retry")}
        </button>
      </div>
    );
  }

  if (session.status === "empty") {
    return (
      <div class="flex flex-col gap-6 items-center py-20 text-center">
        <p class="text-xl text-base-content/70">{t(locale.value, "review.empty")}</p>
        <Link class="btn btn-primary" href="/decks">
          {t(locale.value, "review.complete.back")}
        </Link>
      </div>
    );
  }

  if (session.status === "complete") {
    const correct = session.results.filter((r) => r.wasCorrect).length;
    return (
      <div class="flex flex-col gap-6 items-center py-12 text-center max-w-md mx-auto">
        <h1 class="text-3xl font-semibold">{t(locale.value, "review.complete.title")}</h1>
        <div class="stats stats-vertical border border-base-300 bg-base-100 shadow-sm w-full">
          <div class="stat">
            <div class="stat-title">{t(locale.value, "review.complete.reviewed")}</div>
            <div class="stat-value">{session.results.length}</div>
          </div>
          <div class="stat">
            <div class="stat-title">{t(locale.value, "review.complete.correct")}</div>
            <div class="stat-value text-success">{correct}</div>
            <div class="stat-desc">{Math.round((correct / session.results.length) * 100)}%</div>
          </div>
        </div>
        <div class="flex gap-3">
          <Link class="btn btn-outline" href="/decks">
            {t(locale.value, "review.complete.back")}
          </Link>
          <button class="btn btn-primary" onClick$={handleRestart$} type="button">
            {t(locale.value, "review.complete.again")}
          </button>
        </div>
      </div>
    );
  }

  const currentCard = session.queue[session.currentIndex];

  return (
    <main class="max-w-2xl mx-auto flex flex-col gap-6">
      <h1 class="text-2xl font-semibold">{t(locale.value, "review.title")}</h1>
      <CardReview
        key={currentCard.cardId}
        card={currentCard}
        index={session.currentIndex}
        total={session.queue.length}
        locale={locale.value}
        onAnswer$={handleAnswer$}
        showContent={false}
      />
    </main>
  );
});

export const head: DocumentHead = {
  title: "Review — Shmanki",
};
