import { $, component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useNavigate, type DocumentHead } from "@builder.io/qwik-city";
import { setLocale, setToken, setUser, getToken, getLocale } from "~/lib/auth";
import { api } from "~/lib/api";
import { t } from "~/lib/i18n";
import type { LanguageCode } from "~/lib/types";

export default component$(() => {
  const locale = useSignal<LanguageCode>("en");
  const nav = useNavigate();
  const email = useSignal("");
  const password = useSignal("");
  const error = useSignal<string | null>(null);
  const loading = useSignal(false);

  useVisibleTask$(() => {
    locale.value = getLocale();
    if (getToken()) {
      nav("/decks");
    }
  });

  const handleSubmit$ = $(async () => {
    error.value = null;
    loading.value = true;
    try {
      const res = await api.auth.register(email.value, password.value, locale.value);
      setToken(res.token);
      setUser(res.user);
      setLocale(res.user.preferredLanguage);
      nav("/decks");
    } catch (e) {
      error.value = e instanceof Error ? e.message : t(locale.value, "auth.register.error");
    } finally {
      loading.value = false;
    }
  });

  return (
    <div class="flex min-h-[70vh] items-center justify-center">
      <div class="card w-full max-w-md border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-5">
          <h1 class="card-title text-2xl">{t(locale.value, "auth.register.title")}</h1>

          {error.value && (
            <div class="alert alert-error">
              <span>{error.value}</span>
            </div>
          )}

          <fieldset class="fieldset">
            <legend class="fieldset-legend">{t(locale.value, "auth.register.email")}</legend>
            <input
              class="input input-bordered w-full"
              type="email"
              value={email.value}
              onInput$={(_, el) => (email.value = el.value)}
              disabled={loading.value}
            />
          </fieldset>

          <fieldset class="fieldset">
            <legend class="fieldset-legend">{t(locale.value, "auth.register.password")}</legend>
            <input
              class="input input-bordered w-full"
              type="password"
              value={password.value}
              onInput$={(_, el) => (password.value = el.value)}
              disabled={loading.value}
            />
          </fieldset>

          <button
            class="btn btn-primary"
            onClick$={handleSubmit$}
            disabled={loading.value}
            type="button"
          >
            {loading.value ? (
              <span class="loading loading-spinner loading-sm" />
            ) : (
              t(locale.value, "auth.register.submit")
            )}
          </button>

          <p class="text-center text-sm text-base-content/70">
            {t(locale.value, "auth.register.hasAccount")}{" "}
            <Link class="link link-primary" href="/auth/login">
              {t(locale.value, "auth.register.login")}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
});

export const head: DocumentHead = {
  title: "Register — Shmanki",
};
