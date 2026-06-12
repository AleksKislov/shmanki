import { $, component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useLocation } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t } from "~/lib/i18n";
import type { LanguageCode, PremadeDeckDetail } from "~/lib/types";

export default component$(() => {
  const loc = useLocation();
  const locale = useSignal<LanguageCode>("en");
  const deck = useSignal<PremadeDeckDetail | null>(null);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);

  useVisibleTask$(async () => {
    locale.value = getLocale();
    try {
      deck.value = await api.premade.get(loc.params["premadeId"]);
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      loading.value = false;
    }
  });

  const rate$ = $(async (score: number) => {
    if (!deck.value) return;
    await api.premade.rate(deck.value.id, score);
    deck.value = await api.premade.get(deck.value.id);
  });

  const import$ = $(async () => {
    if (!deck.value) return;
    const result = await api.premade.clone(deck.value.id);
    window.location.href = `/decks/${result.deckId}`;
  });

  if (loading.value) return <div class="flex justify-center py-12"><span class="loading loading-spinner loading-lg" /></div>;
  if (error.value || !deck.value) return <div class="alert alert-error"><span>{error.value ?? t(locale.value, "common.error")}</span></div>;

  return <main class="flex flex-col gap-4">
    <Link class="link link-hover text-sm text-base-content/60" href="/premade">← {t(locale.value, "premade.title")}</Link>
    <div class="flex items-start justify-between gap-3"><div><h1 class="text-3xl font-semibold">{deck.value.title}</h1><p class="text-sm text-base-content/70">{deck.value.description}</p></div><button class="btn btn-primary btn-sm" onClick$={import$}>{t(locale.value, "premade.import")}</button></div>
    <div class="flex items-center gap-2">{[1,2,3,4,5].map((s)=><button key={s} class={`btn btn-xs ${deck.value?.myRating===s?"btn-warning":"btn-ghost"}`} onClick$={() => rate$(s)}>{s}★</button>)}<span class="text-sm text-base-content/70">{deck.value.ratingAvg.toFixed(1)} ({deck.value.ratingCount})</span></div>
    {deck.value.infoObjects.map((obj) => <section key={obj.id} class="card border border-base-300 bg-base-100"><div class="card-body"><h2 class="font-semibold">{obj.title}</h2><p class="text-sm whitespace-pre-wrap">{obj.content}</p><p class="text-xs text-base-content/60">{obj.cards.length} cards</p></div></section>)}
  </main>;
});
