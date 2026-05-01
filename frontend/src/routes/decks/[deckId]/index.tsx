import { $, component$, useSignal, useStore, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useNavigate, useLocation, type DocumentHead } from "@builder.io/qwik-city";
import { CodeBlock } from "~/components/code-block";
import { api } from "~/lib/api";
import { getLocale } from "~/lib/auth";
import { getCardTypeLabel, isBlockInteraction } from "~/lib/card-types";
import { t, getLocaleLabel, LANGUAGE_OPTIONS } from "~/lib/i18n";
import type { DeckDetail, DeckStats, GeneratedCard, InfoObject, LanguageCode } from "~/lib/types";

export default component$(() => {
  const loc = useLocation();
  const nav = useNavigate();
  const locale = useSignal<LanguageCode>("en");
  const deck = useSignal<DeckDetail | null>(null);
  const stats = useSignal<DeckStats | null>(null);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const showObjectForm = useSignal(false);
  const showEditForm = useSignal(false);
  const showGenerateForm = useSignal(false);

  const objectForm = useStore({
    title: "",
    content: "",
    discipline: "programming",
    contentType: "code_go",
    submitting: false,
    error: null as string | null,
  });

  const editForm = useStore({
    title: "",
    description: "",
    languageCode: "en" as LanguageCode,
    submitting: false,
  });

  const generateForm = useStore({
    prompt: "",
    discipline: "programming",
    contentType: "code_go",
    submitting: false,
    error: null as string | null,
    result: null as import("~/lib/types").GenerateSuggestResponse | null,
  });

  useVisibleTask$(async () => {
    locale.value = getLocale();
    const deckId = loc.params["deckId"];
    try {
      const [d, s] = await Promise.all([api.decks.get(deckId), api.decks.stats(deckId)]);
      deck.value = d;
      stats.value = s;
      editForm.title = d.title;
      editForm.description = d.description;
      editForm.languageCode = d.languageCode;
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      loading.value = false;
    }
  });

  const handleAddObject$ = $(async () => {
    objectForm.error = null;
    objectForm.submitting = true;
    try {
      const obj = await api.objects.create(
        loc.params["deckId"],
        objectForm.title,
        objectForm.content,
        objectForm.discipline,
        objectForm.contentType,
      );
      if (deck.value) {
        deck.value = {
          ...deck.value,
          infoObjects: [
            ...deck.value.infoObjects,
            { id: obj.id, title: obj.title, discipline: obj.discipline, contentType: obj.contentType },
          ],
        };
      }
      showObjectForm.value = false;
      objectForm.title = "";
      objectForm.content = "";
    } catch (e) {
      objectForm.error = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      objectForm.submitting = false;
    }
  });

  const handleEditDeck$ = $(async () => {
    editForm.submitting = true;
    try {
      const updated = await api.decks.update(
        loc.params["deckId"],
        editForm.title,
        editForm.description,
        editForm.languageCode,
      );
      if (deck.value) {
        deck.value = { ...deck.value, ...updated };
      }
      showEditForm.value = false;
    } catch {
      // ignore
    } finally {
      editForm.submitting = false;
    }
  });

  const handleDeleteDeck$ = $(async () => {
    if (!confirm(t(locale.value, "deck.delete.confirm"))) return;
    try {
      await api.decks.delete(loc.params["deckId"]);
      nav("/decks");
    } catch {
      // ignore
    }
  });

  const handleDeleteObject$ = $(async (id: string) => {
    if (!confirm(t(locale.value, "object.delete.confirm"))) return;
    try {
      await api.objects.delete(id);
      if (deck.value) {
        deck.value = {
          ...deck.value,
          infoObjects: deck.value.infoObjects.filter((o) => o.id !== id),
        };
      }
    } catch {
      // ignore
    }
  });

  const handleGenerate$ = $(async () => {
    if (!deck.value) return;
    generateForm.error = null;
    generateForm.submitting = true;
    generateForm.result = null;
    try {
      const res = await api.generate.suggest({
        deckId: deck.value.id,
        prompt: generateForm.prompt,
        discipline: generateForm.discipline,
        contentType: generateForm.contentType as import("~/lib/types").ContentType,
      });
      generateForm.result = res;
    } catch (e) {
      generateForm.error = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      generateForm.submitting = false;
    }
  });

  const handleSaveGenerated$ = $(async () => {
    if (!deck.value || !generateForm.result) return;
    generateForm.submitting = true;
    try {
      await api.generate.save({
        deckId: deck.value.id,
        prompt: generateForm.prompt,
        model: generateForm.result.model,
        generationId: generateForm.result.generationId,
        infoObjects: generateForm.result.infoObjects,
      });
      // Refresh deck
      const d = await api.decks.get(deck.value.id);
      deck.value = d;
      showGenerateForm.value = false;
      generateForm.result = null;
      generateForm.prompt = "";
    } catch (e) {
      generateForm.error = e instanceof Error ? e.message : t(locale.value, "common.error");
    } finally {
      generateForm.submitting = false;
    }
  });

  if (loading.value) {
    return (
      <div class="flex justify-center py-20">
        <span class="loading loading-spinner loading-lg" />
      </div>
    );
  }

  if (error.value || !deck.value) {
    return (
      <div class="alert alert-error">
        <span>{error.value ?? t(locale.value, "common.error")}</span>
      </div>
    );
  }

  const d = deck.value;

  return (
    <main class="flex flex-col gap-6">
      {/* Header */}
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="flex flex-col gap-1">
          <Link class="link link-hover text-sm text-base-content/60" href="/decks">
            ← {t(locale.value, "deck.back")}
          </Link>
          <div class="flex items-center gap-3">
            <h1 class="text-3xl font-semibold">{d.title}</h1>
            <span class="badge badge-outline">{d.languageCode}</span>
          </div>
          {d.description && (
            <p class="text-base-content/70 text-sm max-w-2xl">{d.description}</p>
          )}
        </div>
        <div class="flex flex-wrap gap-2">
          <Link class="btn btn-primary btn-sm" href="/review">
            {t(locale.value, "deck.review")}
          </Link>
          <button
            class="btn btn-outline btn-sm"
            onClick$={() => (showGenerateForm.value = !showGenerateForm.value)}
            type="button"
          >
            {t(locale.value, "deck.generate")}
          </button>
          <button
            class="btn btn-ghost btn-sm"
            onClick$={() => (showEditForm.value = !showEditForm.value)}
            type="button"
          >
            {t(locale.value, "deck.edit")}
          </button>
          <button
            class="btn btn-ghost btn-sm text-error"
            onClick$={handleDeleteDeck$}
            type="button"
          >
            {t(locale.value, "deck.delete")}
          </button>
        </div>
      </div>

      {/* Stats */}
      {stats.value && (
        <div class="stats stats-horizontal border border-base-300 bg-base-100 shadow-sm overflow-x-auto">
          {(
            [
              ["deck.stats.new", stats.value.levels.new],
              ["deck.stats.learning", stats.value.levels.learning],
              ["deck.stats.learned", stats.value.levels.learned],
              ["deck.stats.mastered", stats.value.levels.mastered],
              ["deck.stats.expert", stats.value.levels.expert],
              ["deck.stats.due", stats.value.dueNow],
            ] as [string, number][]
          ).map(([key, val]) => (
            <div key={key} class="stat px-4 py-3">
              <div class="stat-title text-xs">{t(locale.value, key)}</div>
              <div class="stat-value text-xl">{val}</div>
            </div>
          ))}
        </div>
      )}

      {/* Edit form */}
      {showEditForm.value && (
        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "deck.form.title")}</span>
                </div>
                <input
                  class="input input-bordered"
                  type="text"
                  value={editForm.title}
                  onInput$={(_, el) => (editForm.title = el.value)}
                />
              </label>
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "decks.form.language")}</span>
                </div>
                <select
                  class="select select-bordered"
                  value={editForm.languageCode}
                  onChange$={(_, el) => (editForm.languageCode = el.value as LanguageCode)}
                >
                  {LANGUAGE_OPTIONS.map((code) => (
                    <option key={code} value={code}>
                      {getLocaleLabel(code)}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "decks.form.description")}</span>
              </div>
              <textarea
                class="textarea textarea-bordered"
                value={editForm.description}
                onInput$={(_, el) => (editForm.description = el.value)}
                rows={2}
              />
            </label>
            <div class="flex gap-2">
              <button
                class="btn btn-primary btn-sm"
                onClick$={handleEditDeck$}
                disabled={editForm.submitting}
                type="button"
              >
                {t(locale.value, "common.save")}
              </button>
              <button
                class="btn btn-ghost btn-sm"
                onClick$={() => (showEditForm.value = false)}
                type="button"
              >
                {t(locale.value, "common.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Generate form */}
      {showGenerateForm.value && (
        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            {generateForm.error && (
              <div class="alert alert-error">
                <span>{generateForm.error}</span>
              </div>
            )}
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "deck.generate.prompt")}</span>
              </div>
              <textarea
                class="textarea textarea-bordered"
                value={generateForm.prompt}
                onInput$={(_, el) => (generateForm.prompt = el.value)}
                rows={3}
              />
            </label>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "deck.form.discipline")}</span>
                </div>
                <input
                  class="input input-bordered"
                  type="text"
                  value={generateForm.discipline}
                  onInput$={(_, el) => (generateForm.discipline = el.value)}
                />
              </label>
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "deck.form.contentType")}</span>
                </div>
                <select
                  class="select select-bordered"
                  value={generateForm.contentType}
                  onChange$={(_, el) => (generateForm.contentType = el.value)}
                >
                  {["text", "code_go", "code_python", "code_js", "code_ts", "code_rust"].map(
                    (ct) => (
                      <option key={ct} value={ct}>
                        {ct}
                      </option>
                    ),
                  )}
                </select>
              </label>
            </div>

            {generateForm.result && (
              <div class="rounded-box border border-base-300 bg-base-200 p-4">
                <p class="text-sm font-semibold mb-2">
                  {generateForm.result.infoObjects.length} objects generated
                </p>
                <div class="mb-4 rounded-box border border-base-300 bg-base-100/70 p-3 text-sm text-base-content/70">
                  <p class="font-semibold mb-2">{t(locale.value, "deck.generate.preview.progression")}</p>
                  <div class="flex flex-col gap-1">
                    <p>{t(locale.value, "deck.generate.progression.step0")}</p>
                    <p>{t(locale.value, "deck.generate.progression.step1")}</p>
                    <p>{t(locale.value, "deck.generate.progression.step2")}</p>
                    <p>{t(locale.value, "deck.generate.progression.step3")}</p>
                  </div>
                </div>
                {generateForm.result.infoObjects.map((obj, i) => (
                  <div key={i} class="mb-4 rounded-box border border-base-300 bg-base-100 p-4 last:mb-0">
                    <div class="flex flex-wrap items-start justify-between gap-2">
                      <div>
                        <p class="font-medium">{obj.title}</p>
                        <p class="text-xs text-base-content/60">{obj.cards.length} cards</p>
                      </div>
                      <div class="flex flex-wrap gap-2 text-xs">
                        <span class="badge badge-outline">{obj.discipline}</span>
                        <span class="badge badge-outline">{obj.contentType}</span>
                      </div>
                    </div>

                    <div class="mt-4 flex flex-col gap-4">
                      <div class="flex flex-col gap-2">
                        <p class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
                          {t(locale.value, "deck.generate.preview.content")}
                        </p>
                        <div class="overflow-hidden rounded-box border border-base-300 bg-base-300/40">
                          <CodeBlock code={obj.content} contentType={obj.contentType} highlightLines={[]} />
                        </div>
                      </div>

                      <div class="flex flex-col gap-3">
                        <p class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
                          {t(locale.value, "deck.generate.preview.cards")}
                        </p>
                        {obj.cards.map((card, cardIndex) => (
                          <GeneratedCardPreview key={cardIndex} card={card} locale={locale.value} />
                        ))}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <div class="flex gap-2">
              {!generateForm.result ? (
                <button
                  class="btn btn-primary btn-sm"
                  onClick$={handleGenerate$}
                  disabled={generateForm.submitting || !generateForm.prompt.trim()}
                  type="button"
                >
                  {generateForm.submitting ? (
                    <span class="loading loading-spinner loading-xs" />
                  ) : (
                    t(locale.value, "deck.generate.submit")
                  )}
                </button>
              ) : (
                <button
                  class="btn btn-success btn-sm"
                  onClick$={handleSaveGenerated$}
                  disabled={generateForm.submitting}
                  type="button"
                >
                  {generateForm.submitting ? (
                    <span class="loading loading-spinner loading-xs" />
                  ) : (
                    t(locale.value, "deck.generate.save")
                  )}
                </button>
              )}
              <button
                class="btn btn-ghost btn-sm"
                onClick$={() => {
                  showGenerateForm.value = false;
                  generateForm.result = null;
                }}
                type="button"
              >
                {t(locale.value, "deck.generate.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Info objects */}
      <div class="flex items-center justify-between gap-4">
        <h2 class="text-xl font-semibold">{t(locale.value, "deck.objects")}</h2>
        <button
          class="btn btn-outline btn-sm"
          onClick$={() => (showObjectForm.value = !showObjectForm.value)}
          type="button"
        >
          {t(locale.value, "deck.addObject")}
        </button>
      </div>

      {showObjectForm.value && (
        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            {objectForm.error && (
              <div class="alert alert-error">
                <span>{objectForm.error}</span>
              </div>
            )}
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="form-control">
                <div class="label">
                  <span class="label-text">{t(locale.value, "deck.form.title")}</span>
                </div>
                <input
                  class="input input-bordered"
                  type="text"
                  value={objectForm.title}
                  onInput$={(_, el) => (objectForm.title = el.value)}
                />
              </label>
              <div class="grid gap-4 sm:grid-cols-2">
                <label class="form-control">
                  <div class="label">
                    <span class="label-text">{t(locale.value, "deck.form.discipline")}</span>
                  </div>
                  <input
                    class="input input-bordered"
                    type="text"
                    value={objectForm.discipline}
                    onInput$={(_, el) => (objectForm.discipline = el.value)}
                  />
                </label>
                <label class="form-control">
                  <div class="label">
                    <span class="label-text">{t(locale.value, "deck.form.contentType")}</span>
                  </div>
                  <select
                    class="select select-bordered"
                    value={objectForm.contentType}
                    onChange$={(_, el) => (objectForm.contentType = el.value)}
                  >
                    {["text", "code_go", "code_python", "code_js", "code_ts", "code_rust"].map(
                      (ct) => (
                        <option key={ct} value={ct}>
                          {ct}
                        </option>
                      ),
                    )}
                  </select>
                </label>
              </div>
            </div>
            <label class="form-control">
              <div class="label">
                <span class="label-text">{t(locale.value, "deck.form.content")}</span>
              </div>
              <textarea
                class="textarea textarea-bordered font-mono text-sm"
                value={objectForm.content}
                onInput$={(_, el) => (objectForm.content = el.value)}
                rows={6}
              />
            </label>
            <div class="flex gap-2">
              <button
                class="btn btn-primary btn-sm"
                onClick$={handleAddObject$}
                disabled={objectForm.submitting || !objectForm.title.trim()}
                type="button"
              >
                {objectForm.submitting ? (
                  <span class="loading loading-spinner loading-xs" />
                ) : (
                  t(locale.value, "deck.form.submit")
                )}
              </button>
              <button
                class="btn btn-ghost btn-sm"
                onClick$={() => (showObjectForm.value = false)}
                type="button"
              >
                {t(locale.value, "deck.form.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {d.infoObjects.length === 0 ? (
        <p class="text-center py-8 text-base-content/60">{t(locale.value, "deck.empty")}</p>
      ) : (
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {d.infoObjects.map((obj) => (
            <article key={obj.id} class="card border border-base-300 bg-base-100 shadow-sm">
              <div class="card-body gap-2">
                <div class="flex items-start justify-between gap-2">
                  <h3 class="font-semibold text-base leading-tight">{obj.title}</h3>
                  <span class="badge badge-ghost badge-sm shrink-0">{obj.contentType}</span>
                </div>
                <div class="card-actions mt-2 justify-between">
                  <Link class="btn btn-primary btn-xs" href={`/objects/${obj.id}`}>
                    {t(locale.value, "common.edit")}
                  </Link>
                  <button
                    class="btn btn-ghost btn-xs text-error"
                    onClick$={() => handleDeleteObject$(obj.id)}
                    type="button"
                  >
                    {t(locale.value, "common.delete")}
                  </button>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}
    </main>
  );
});

interface GeneratedCardPreviewProps {
  card: GeneratedCard;
  locale: LanguageCode;
}

const GeneratedCardPreview = component$<GeneratedCardPreviewProps>(({ card, locale }) => {
  return (
    <div class="rounded-box border border-base-300 bg-base-200/60 p-3">
      <div class="flex items-start justify-between gap-3">
        <p class="font-medium leading-snug">{card.front}</p>
        <div class="flex flex-wrap gap-2 justify-end">
          <span class="badge badge-outline badge-sm shrink-0">
            {getCardTypeLabel(locale, card.cardType)}
          </span>
          <span class="badge badge-ghost badge-sm shrink-0">
            {t(locale, "object.card.step")} {card.step}
          </span>
        </div>
      </div>

      {isBlockInteraction(card.cardType) && (
        <div class="mt-3 flex flex-col gap-2">
          <PreviewField
            label={t(locale, "deck.generate.preview.cardType")}
            value={getCardTypeLabel(locale, card.cardType)}
          />
          <div class="flex flex-col gap-2">
            {(card.correctAnswers[0] ?? []).map((unit, index) => (
              <pre
                key={index}
                class="rounded-box border border-base-300 bg-base-100/80 p-2 text-xs whitespace-pre-wrap break-words"
              >
                {unit}
              </pre>
            ))}
          </div>
        </div>
      )}

      <div class="mt-3 grid gap-3 sm:grid-cols-3">
        <PreviewField
          label={t(locale, "deck.generate.preview.cardType")}
          value={getCardTypeLabel(locale, card.cardType)}
        />
        <PreviewField
          label={t(locale, "deck.generate.preview.correctAnswers")}
          value={formatAnswerGroups(card.correctAnswers)}
        />
        <PreviewField
          label={t(locale, "deck.generate.preview.distractors")}
          value={card.distractors.length > 0 ? card.distractors.join(", ") : "-"}
        />
        <PreviewField
          label={t(locale, "deck.generate.preview.highlightLines")}
          value={card.highlightLines.length > 0 ? card.highlightLines.join(", ") : "-"}
        />
      </div>
    </div>
  );
});

interface PreviewFieldProps {
  label: string;
  value: string;
}

const PreviewField = component$<PreviewFieldProps>(({ label, value }) => {
  return (
    <div class="flex flex-col gap-1 min-w-0">
      <p class="text-xs font-semibold uppercase tracking-wide text-base-content/60">{label}</p>
      <p class="text-sm whitespace-pre-wrap break-words">{value}</p>
    </div>
  );
});

function formatAnswerGroups(answerGroups: GeneratedCard["correctAnswers"]) {
  if (answerGroups.length === 0) {
    return "-";
  }

  return answerGroups.map((group) => group.join(" ")).join("\n");
}

export const head: DocumentHead = {
  title: "Deck — Shmanki",
};
