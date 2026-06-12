import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { LANGUAGE_OPTIONS, getLocaleLabel, t } from "~/lib/i18n";
import type { LanguageCode, PremadeDeck } from "~/lib/types";

export default component$(() => {
  const locale = useSignal<LanguageCode>("en");
  const decks = useSignal<PremadeDeck[]>([]);
  const categories = useSignal<string[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const filters = useStore({ source: "all" as "official" | "community" | "all", category: "", language: "" as LanguageCode | "", minRating: 0, sort: "rating" as "rating" | "newest" | "popular" });

  const load$ = $(async () => {
    loading.value = true;
    try {
      decks.value = await api.premade.list(filters);
      categories.value = await api.premade.categories();
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      loading.value = false;
    }
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    await load$();
  });

  const handleImport$ = $(async (id: string) => {
    const result = await api.premade.clone(id);
    window.location.href = `/decks/${result.deckId}`;
  });

  return <main class="flex flex-col gap-6">
    <div class="flex flex-wrap items-end gap-3">
      <h1 class="text-3xl font-semibold mr-auto">{t(locale.value, "premade.title")}</h1>
      <select class="select select-bordered select-sm" value={filters.source} onChange$={async (_, el) => { filters.source = el.value as any; await load$(); }}>
        <option value="all">{t(locale.value, "premade.source.all")}</option><option value="official">{t(locale.value, "premade.source.official")}</option><option value="community">{t(locale.value, "premade.source.community")}</option>
      </select>
      <select class="select select-bordered select-sm" value={filters.category} onChange$={async (_, el) => { filters.category = el.value; await load$(); }}>
        <option value="">{t(locale.value, "premade.allCategories")}</option>
        {categories.value.map((c) => <option value={c} key={c}>{c}</option>)}
      </select>
      <select class="select select-bordered select-sm" value={filters.language} onChange$={async (_, el) => { filters.language = el.value as any; await load$(); }}>
        <option value="">{t(locale.value, "premade.allLanguages")}</option>
        {LANGUAGE_OPTIONS.map((code) => <option value={code} key={code}>{getLocaleLabel(code)}</option>)}
      </select>
      <select class="select select-bordered select-sm" value={String(filters.minRating)} onChange$={async (_, el) => { filters.minRating = Number(el.value); await load$(); }}>
        <option value="0">{t(locale.value, "premade.minRating")}</option>
        {[1, 2, 3, 4, 5].map((v) => <option value={String(v)} key={String(v)}>{`${v}+`}</option>)}
      </select>
    </div>
    {loading.value && <span class="loading loading-spinner loading-lg mx-auto" />}
    {error.value && <div class="alert alert-error"><span>{error.value}</span></div>}
    {!loading.value && !error.value && decks.value.length === 0 && <div class="text-center py-12 text-base-content/60">{t(locale.value, "premade.empty")}</div>}
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {decks.value.map((deck) => <article key={deck.id} class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-3">
          <div class="flex items-start justify-between gap-2">
            <h2 class="card-title text-lg leading-tight">{deck.title}</h2>
            <span class="badge badge-outline badge-sm">{deck.languageCode}</span>
          </div>
          <p class="text-sm text-base-content/70">{deck.description}</p>
          <div class="flex flex-wrap gap-2 text-xs"><span class="badge badge-ghost">{deck.category}</span><span class="badge badge-ghost">{deck.source}</span><span class="badge badge-ghost">{deck.ratingAvg.toFixed(1)}★ ({deck.ratingCount})</span></div>
          {deck.source === "community" && <p class="text-xs text-base-content/60">{t(locale.value, "premade.author")}: {deck.authorName}</p>}
          <div class="card-actions justify-between mt-2"><Link class="btn btn-outline btn-sm" href={`/premade/${deck.id}`}>{t(locale.value, "premade.preview")}</Link><button class="btn btn-primary btn-sm" onClick$={() => handleImport$(deck.id)} type="button">{t(locale.value, "premade.import")}</button></div>
        </div>
      </article>)}
    </div>
  </main>;
});
