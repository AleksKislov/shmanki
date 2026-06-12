import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link } from "@builder.io/qwik-city";
import { api } from "~/lib/api";
import { getLocale, getUser, isAdmin } from "~/lib/auth";
import { t } from "~/lib/i18n";
import type { LanguageCode, PremadeDeck } from "~/lib/types";

export default component$(() => {
  const locale = useSignal<LanguageCode>("en");
  const decks = useSignal<PremadeDeck[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const source = useSignal<"all" | "official" | "community">("all");
  const creating = useSignal(false);
  const form = useStore({ deckId: "", category: "", title: "", description: "" });

  const load$ = $(async () => {
    loading.value = true;
    try {
      decks.value = await api.admin.premade.list(source.value);
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      loading.value = false;
    }
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    if (!isAdmin(getUser())) {
      window.location.href = "/premade";
      return;
    }
    await load$();
  });

  const togglePublished$ = $(async (deck: PremadeDeck) => {
    await api.admin.premade.setPublished(deck.id, !deck.isPublished);
    await load$();
  });

  const deleteDeck$ = $(async (deck: PremadeDeck) => {
    if (!confirm(`Delete ${deck.title}?`)) return;
    await api.admin.premade.delete(deck.id);
    await load$();
  });

  const createOfficial$ = $(async () => {
    creating.value = true;
    try {
      await api.admin.premade.createOfficialFromDeck(form.deckId, form.category, form.title, form.description);
      form.deckId = "";
      form.category = "";
      form.title = "";
      form.description = "";
      await load$();
    } catch (e) {
      alert(e instanceof Error ? e.message : t(locale.value, "common.error"));
    } finally {
      creating.value = false;
    }
  });

  return (
    <main class="flex flex-col gap-6">
      <div class="flex items-center justify-between gap-3">
        <h1 class="text-3xl font-semibold">Admin Premade Decks</h1>
        <Link class="btn btn-outline btn-sm" href="/premade">Open public catalog</Link>
      </div>

      <section class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-3">
          <h2 class="text-lg font-semibold">Create official from existing deck</h2>
          <div class="grid gap-3 sm:grid-cols-2">
            <input class="input input-bordered" placeholder="Source deck UUID" value={form.deckId} onInput$={(_, el) => (form.deckId = el.value)} />
            <input class="input input-bordered" placeholder="Category" value={form.category} onInput$={(_, el) => (form.category = el.value)} />
            <input class="input input-bordered" placeholder="Title override (optional)" value={form.title} onInput$={(_, el) => (form.title = el.value)} />
            <input class="input input-bordered" placeholder="Description override (optional)" value={form.description} onInput$={(_, el) => (form.description = el.value)} />
          </div>
          <div>
            <button class="btn btn-primary btn-sm" disabled={creating.value || !form.deckId || !form.category} onClick$={createOfficial$}>
              {creating.value ? "Creating..." : "Create official premade"}
            </button>
          </div>
        </div>
      </section>

      <div class="flex gap-2">
        {(["all", "official", "community"] as const).map((value) => (
          <button key={value} class={`btn btn-sm ${source.value === value ? "btn-primary" : "btn-ghost"}`} onClick$={async () => { source.value = value; await load$(); }}>
            {value}
          </button>
        ))}
      </div>

      {loading.value && <div class="flex justify-center py-10"><span class="loading loading-spinner loading-lg" /></div>}
      {error.value && <div class="alert alert-error"><span>{error.value}</span></div>}

      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {decks.value.map((deck) => (
          <article key={deck.id} class="card border border-base-300 bg-base-100 shadow-sm">
            <div class="card-body gap-3">
              <div class="flex items-start justify-between gap-2">
                <h2 class="card-title text-lg leading-tight">{deck.title}</h2>
                <span class={`badge badge-sm ${deck.isPublished ? "badge-success" : "badge-warning"}`}>{deck.isPublished ? "published" : "hidden"}</span>
              </div>
              <p class="text-sm text-base-content/70">{deck.description}</p>
              <div class="flex flex-wrap gap-2 text-xs">
                <span class="badge badge-ghost">{deck.source}</span>
                <span class="badge badge-ghost">{deck.category}</span>
                <span class="badge badge-ghost">{deck.languageCode}</span>
              </div>
              {deck.source === "community" && <p class="text-xs text-base-content/60">Author: {deck.authorName || "unknown"}</p>}
              <div class="card-actions justify-between mt-2">
                <button class="btn btn-outline btn-sm" onClick$={() => togglePublished$(deck)}>{deck.isPublished ? "Unpublish" : "Publish"}</button>
                <button class="btn btn-error btn-sm" onClick$={() => deleteDeck$(deck)}>Delete</button>
              </div>
            </div>
          </article>
        ))}
      </div>
    </main>
  );
});
