import { $, component$, useSignal, useStore } from "@builder.io/qwik";
import type { QRL } from "@builder.io/qwik";
import { getCardTypeLabel, isBlockInteraction } from "~/lib/card-types";
import { t } from "~/lib/i18n";
import type { CardType, LanguageCode } from "~/lib/types";

export interface CardDisplayData {
  front: string;
  cardType: CardType;
  step: number;
  correctAnswers: string[][];
  distractors: string[];
}

interface CardRowProps {
  card: CardDisplayData;
  locale: LanguageCode;
  onTest$: QRL<() => void>;
  onDelete$: QRL<() => void>;
  onEdit$?: QRL<(updated: CardDisplayData) => Promise<void>>;
  onAddBefore$?: QRL<(card: CardDisplayData) => Promise<void>>;
  onAddAfter$?: QRL<(card: CardDisplayData) => Promise<void>>;
  /** If provided, a "Generate" prompt field appears in the add/edit form */
  onGenerateCard$?: QRL<(prompt: string, discipline: string, contentType: string) => Promise<CardDisplayData>>;
}

type Mode = "idle" | "editing" | "adding-before" | "adding-after";

interface CardForm {
  front: string;
  cardType: CardType;
  step: number;
  correctAnswersRaw: string;
  distractorsRaw: string;
  generatePrompt: string;
  generating: boolean;
  submitting: boolean;
  error: string | null;
}

function cardToForm(card: CardDisplayData): Partial<CardForm> {
  return {
    front: card.front,
    cardType: card.cardType,
    step: card.step,
    correctAnswersRaw: card.correctAnswers.map((g) => g.join(", ")).join("\n"),
    distractorsRaw: card.distractors.join(", "),
  };
}

function formToCard(form: CardForm): CardDisplayData {
  return {
    front: form.front.trim(),
    cardType: form.cardType,
    step: form.step,
    correctAnswers: [form.correctAnswersRaw.split(",").map((s) => s.trim()).filter(Boolean)],
    distractors: form.distractorsRaw.split(",").map((s) => s.trim()).filter(Boolean),
  };
}

const CARD_TYPES: CardType[] = ["concept", "signature", "trace", "line_order", "choose_snippet", "fix_bug"];

export const CardRow = component$<CardRowProps>(({
  card, locale, onTest$, onDelete$, onEdit$, onAddBefore$, onAddAfter$, onGenerateCard$,
}) => {
  const mode = useSignal<Mode>("idle");

  const form = useStore<CardForm>({
    front: "",
    cardType: "concept",
    step: 0,
    correctAnswersRaw: "",
    distractorsRaw: "",
    generatePrompt: "",
    generating: false,
    submitting: false,
    error: null,
  });

  const openEdit$ = $(() => {
    Object.assign(form, cardToForm(card), { generatePrompt: "", generating: false, submitting: false, error: null });
    mode.value = "editing";
  });

  const openAdd$ = $((m: "adding-before" | "adding-after") => {
    Object.assign(form, { front: "", cardType: "concept", step: card.step, correctAnswersRaw: "", distractorsRaw: "", generatePrompt: "", generating: false, submitting: false, error: null });
    mode.value = m;
  });

  const closeForm$ = $(() => { mode.value = "idle"; form.error = null; });

  const handleGenerate$ = $(async () => {
    if (!onGenerateCard$ || !form.generatePrompt.trim()) return;
    form.generating = true;
    form.error = null;
    try {
      const result = await onGenerateCard$(form.generatePrompt, "", "");
      form.front = result.front;
      form.cardType = result.cardType;
      form.step = result.step;
      form.correctAnswersRaw = result.correctAnswers.map((g) => g.join(", ")).join("\n");
      form.distractorsRaw = result.distractors.join(", ");
    } catch (e) {
      form.error = e instanceof Error ? e.message : t(locale, "common.error");
    } finally {
      form.generating = false;
    }
  });

  const handleSave$ = $(async () => {
    form.error = null;
    form.submitting = true;
    const data = formToCard(form);
    try {
      if (mode.value === "editing" && onEdit$) {
        await onEdit$(data);
      } else if (mode.value === "adding-before" && onAddBefore$) {
        await onAddBefore$(data);
      } else if (mode.value === "adding-after" && onAddAfter$) {
        await onAddAfter$(data);
      }
      mode.value = "idle";
    } catch (e) {
      form.error = e instanceof Error ? e.message : t(locale, "common.error");
    } finally {
      form.submitting = false;
    }
  });

  return (
    <div class="flex flex-col gap-2">
      {/* Add-before form */}
      {mode.value === "adding-before" && (
        <CardForm
          form={form}
          locale={locale}
          hasGenerateCard={!!onGenerateCard$}
          onGenerate$={handleGenerate$}
          onSave$={handleSave$}
          onCancel$={closeForm$}
        />
      )}

      {/* Card display */}
      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-3 py-4">
          {/* Body */}
          <div class="flex flex-col gap-1 min-w-0">
            <p class="font-medium leading-snug">{card.front}</p>
            <div class="flex flex-wrap gap-2 text-sm text-base-content/60">
              <span>{t(locale, "object.card.type")}: {getCardTypeLabel(locale, card.cardType)}</span>
              <span>{t(locale, "object.card.step")} {card.step}</span>
              <span>→ {card.correctAnswers[0]?.join(" ")}</span>
            </div>
            {isBlockInteraction(card.cardType) && (
              <div class="mt-2 flex flex-col gap-2">
                {card.correctAnswers[0]?.map((unit, i) => (
                  <pre key={i} class="rounded-box border border-base-300 bg-base-200/60 p-2 text-xs whitespace-pre-wrap break-words">{unit}</pre>
                ))}
              </div>
            )}
            {card.distractors.length > 0 && (
              <div class="flex flex-wrap gap-1 mt-1">
                {card.distractors.map((d) => (
                  <span key={d} class="badge badge-ghost badge-xs">{d}</span>
                ))}
              </div>
            )}
          </div>

          {/* Inline edit form */}
          {mode.value === "editing" && (
            <CardForm
              form={form}
              locale={locale}
              hasGenerateCard={!!onGenerateCard$}
              onGenerate$={handleGenerate$}
              onSave$={handleSave$}
              onCancel$={closeForm$}
            />
          )}

          {/* Footer buttons */}
          {mode.value === "idle" && (
            <div class="flex flex-wrap items-center justify-between gap-2 pt-1 border-t border-base-300">
              <div class="flex gap-1">
                {onAddBefore$ && (
                  <button class="btn btn-ghost btn-xs" onClick$={() => openAdd$("adding-before")} type="button">
                    + {t(locale, "object.card.addBefore")}
                  </button>
                )}
                {onAddAfter$ && (
                  <button class="btn btn-ghost btn-xs" onClick$={() => openAdd$("adding-after")} type="button">
                    + {t(locale, "object.card.addAfter")}
                  </button>
                )}
              </div>
              <div class="flex gap-1">
                {onEdit$ && (
                  <button class="btn btn-ghost btn-xs" onClick$={openEdit$} type="button">
                    {t(locale, "common.edit")}
                  </button>
                )}
                <button class="btn btn-ghost btn-xs" onClick$={onTest$} type="button">
                  {t(locale, "object.card.test")}
                </button>
                <button class="btn btn-ghost btn-xs text-error" onClick$={onDelete$} type="button">
                  {t(locale, "common.delete")}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Add-after form */}
      {mode.value === "adding-after" && (
        <CardForm
          form={form}
          locale={locale}
          hasGenerateCard={!!onGenerateCard$}
          onGenerate$={handleGenerate$}
          onSave$={handleSave$}
          onCancel$={closeForm$}
        />
      )}
    </div>
  );
});

interface CardFormProps {
  form: CardForm;
  locale: LanguageCode;
  hasGenerateCard: boolean;
  onGenerate$: QRL<() => void>;
  onSave$: QRL<() => void>;
  onCancel$: QRL<() => void>;
}

export const CardForm = component$<CardFormProps>(({ form, locale, hasGenerateCard, onGenerate$, onSave$, onCancel$ }) => {
  return (
    <div class="card border border-primary/30 bg-base-100 shadow-sm">
      <div class="card-body gap-3 py-4">
        {form.error && <div class="alert alert-error py-2"><span class="text-sm">{form.error}</span></div>}

        {hasGenerateCard && (
          <div class="flex gap-2 items-end">
            <fieldset class="fieldset flex-1">
              <legend class="fieldset-legend">{t(locale, "object.card.generatePrompt")}</legend>
              <input
                class="input input-bordered input-sm w-full"
                type="text"
                value={form.generatePrompt}
                onInput$={(_, el) => (form.generatePrompt = el.value)}
                placeholder={t(locale, "object.card.generatePlaceholder")}
              />
            </fieldset>
            <button
              class="btn btn-outline btn-sm"
              onClick$={onGenerate$}
              disabled={form.generating || !form.generatePrompt.trim()}
              type="button"
            >
              {form.generating ? <span class="loading loading-spinner loading-xs" /> : t(locale, "object.card.generate")}
            </button>
          </div>
        )}

        <fieldset class="fieldset">
          <legend class="fieldset-legend">{t(locale, "object.form.front")}</legend>
          <textarea
            class="textarea textarea-bordered w-full text-sm"
            rows={2}
            value={form.front}
            onInput$={(_, el) => (form.front = el.value)}
          />
        </fieldset>

        <div class="grid gap-3 sm:grid-cols-2">
          <fieldset class="fieldset">
            <legend class="fieldset-legend">{t(locale, "object.form.cardType")}</legend>
            <select
              class="select select-bordered select-sm w-full"
              value={form.cardType}
              onChange$={(_, el) => (form.cardType = el.value as CardType)}
            >
              {CARD_TYPES.map((ct) => (
                <option key={ct} value={ct}>{getCardTypeLabel(locale, ct)}</option>
              ))}
            </select>
          </fieldset>
          <fieldset class="fieldset">
            <legend class="fieldset-legend">{t(locale, "object.form.step")}</legend>
            <input
              class="input input-bordered input-sm w-full"
              type="number"
              min={0}
              value={form.step}
              onInput$={(_, el) => (form.step = parseInt(el.value, 10) || 0)}
            />
          </fieldset>
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">{t(locale, "object.form.correctAnswers")}</legend>
          <textarea
            class="textarea textarea-bordered w-full font-mono text-xs"
            rows={2}
            value={form.correctAnswersRaw}
            onInput$={(_, el) => (form.correctAnswersRaw = el.value)}
            placeholder="token1, token2"
          />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">{t(locale, "object.form.distractors")}</legend>
          <input
            class="input input-bordered input-sm w-full"
            type="text"
            value={form.distractorsRaw}
            onInput$={(_, el) => (form.distractorsRaw = el.value)}
            placeholder="wrong1, wrong2"
          />
        </fieldset>

        <div class="flex gap-2">
          <button
            class="btn btn-primary btn-sm"
            onClick$={onSave$}
            disabled={form.submitting || !form.front.trim()}
            type="button"
          >
            {form.submitting ? <span class="loading loading-spinner loading-xs" /> : t(locale, "common.save")}
          </button>
          <button class="btn btn-ghost btn-sm" onClick$={onCancel$} type="button">
            {t(locale, "common.cancel")}
          </button>
        </div>
      </div>
    </div>
  );
});
