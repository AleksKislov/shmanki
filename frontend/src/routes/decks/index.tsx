import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link, type DocumentHead } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { t, getLocaleLabel, LANGUAGE_OPTIONS } from "~/lib/i18n";
import type { Deck, DeckStats, LanguageCode } from "~/lib/types";

interface DeckWithStats extends Deck {
  stats?: DeckStats;
}

export default component$(() => {
  const locale = useSignal<LanguageCode>("en");
  const decks = useSignal<DeckWithStats[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const showForm = useSignal(false);

  const form = useStore({
    title: "",
    description: "",
    languageCode: "en" as LanguageCode,
    submitting: false,
    error: null as string | null,
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    form.languageCode = locale.value;
    try {
      const data = await api.decks.list();
      decks.value = data;
      // Fetch stats for each deck in parallel
      const statsResults = await Promise.allSettled(data.map((d) => api.decks.stats(d.id)));
      decks.value = data.map((d, i) => {
        const s = statsResults[i];
        return { ...d, stats: s.status === "fulfilled" ? s.value : undefined };
      });
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "decks.error");
    } finally {
      loading.value = false;
    }
  });

  const handleCreate$ = $(async () => {
    form.error = null;
    form.submitting = true;
    try {
      const deck = await api.decks.create(form.title, form.description, form.languageCode);
      decks.value = [...decks.value, deck];
      showForm.value = false;
      form.title = "";
      form.description = "";
    } catch (e) {
      form.error = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      form.submitting = false;
    }
  });

  const handleDelete$ = $(async (id: string) => {
    if (!confirm(t(locale.value, "decks.delete.confirm"))) return;
    try {
      await api.decks.delete(id);
      decks.value = decks.value.filter((d) => d.id !== id);
    } catch {
      // ignore
    }
  });

  return (
    <main class='flex flex-col gap-6'>
      <div class='flex items-center justify-between gap-4'>
        <h1 class='text-3xl font-semibold'>{t(locale.value, "decks.title")}</h1>
        <div class='flex gap-2'>
          <Link class='btn btn-primary btn-sm' href='/review'>
            {t(locale.value, "header.review")}
          </Link>
          <button
            class='btn btn-outline btn-sm'
            onClick$={() => (showForm.value = !showForm.value)}
            type='button'
          >
            {t(locale.value, "decks.create")}
          </button>
        </div>
      </div>

      {showForm.value && (
        <div class='card border border-base-300 bg-base-100 shadow-sm'>
          <div class='card-body gap-4'>
            {form.error && (
              <div class='alert alert-error'>
                <span>{form.error}</span>
              </div>
            )}
            <div class='grid gap-4 sm:grid-cols-2'>
              <fieldset class='fieldset'>
                <legend class='fieldset-legend'>{t(locale.value, "decks.form.title")}</legend>
                <input
                  class='input input-bordered w-full'
                  type='text'
                  value={form.title}
                  onInput$={(_, el) => (form.title = el.value)}
                />
              </fieldset>
              <fieldset class='fieldset'>
                <legend class='fieldset-legend'>{t(locale.value, "decks.form.language")}</legend>
                <select
                  class='select select-bordered w-full'
                  value={form.languageCode}
                  onChange$={(_, el) => (form.languageCode = el.value as LanguageCode)}
                >
                  {LANGUAGE_OPTIONS.map((code) => (
                    <option key={code} value={code}>
                      {getLocaleLabel(code)}
                    </option>
                  ))}
                </select>
              </fieldset>
            </div>
            <fieldset class='fieldset'>
              <legend class='fieldset-legend'>{t(locale.value, "decks.form.description")}</legend>
              <textarea
                class='textarea textarea-bordered w-full'
                value={form.description}
                onInput$={(_, el) => (form.description = el.value)}
                rows={2}
              />
            </fieldset>
            <div class='flex gap-2'>
              <button
                class='btn btn-primary btn-sm'
                onClick$={handleCreate$}
                disabled={form.submitting || !form.title.trim()}
                type='button'
              >
                {form.submitting ? (
                  <span class='loading loading-spinner loading-xs' />
                ) : (
                  t(locale.value, "decks.form.submit")
                )}
              </button>
              <button
                class='btn btn-ghost btn-sm'
                onClick$={() => (showForm.value = false)}
                type='button'
              >
                {t(locale.value, "decks.form.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {loading.value && (
        <div class='flex justify-center py-12'>
          <span class='loading loading-spinner loading-lg' />
        </div>
      )}

      {error.value && (
        <div class='alert alert-error'>
          <span>{error.value}</span>
        </div>
      )}

      {!loading.value && !error.value && decks.value.length === 0 && (
        <div class='text-center py-12 text-base-content/60'>{t(locale.value, "decks.empty")}</div>
      )}

      <div class='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
        {decks.value.map((deck) => (
          <article key={deck.id} class='card border border-base-300 bg-base-100 shadow-sm'>
            <div class='card-body gap-3'>
              <div class='flex items-start justify-between gap-2'>
                <h2 class='card-title text-lg leading-tight'>{deck.title}</h2>
                <span class='badge badge-outline badge-sm shrink-0'>{deck.languageCode}</span>
              </div>

              {deck.description && (
                <p class='text-sm text-base-content/70 leading-relaxed'>{deck.description}</p>
              )}

              {deck.stats && (
                <div class='flex flex-wrap gap-2'>
                  <span class='badge badge-ghost badge-sm'>
                    {t(locale.value, "decks.card.due")} {deck.stats.dueNow}
                  </span>
                  <span class='badge badge-ghost badge-sm'>
                    {t(locale.value, "decks.card.new")} {deck.stats.newNow}
                  </span>
                  <span class='badge badge-ghost badge-sm'>
                    {deck.stats.infoObjects} {t(locale.value, "decks.card.objects")}
                  </span>
                  <span class='badge badge-ghost badge-sm'>
                    {deck.stats.cards} {t(locale.value, "decks.card.cards")}
                  </span>
                </div>
              )}

              <div class='card-actions mt-2 justify-between'>
                <Link class='btn btn-primary btn-sm' href={`/decks/${deck.id}`}>
                  {t(locale.value, "common.edit")}
                </Link>
                <button
                  class='btn btn-ghost btn-sm text-error'
                  onClick$={() => handleDelete$(deck.id)}
                  type='button'
                >
                  {t(locale.value, "common.delete")}
                </button>
              </div>
            </div>
          </article>
        ))}
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Decks — Shmanki",
};
