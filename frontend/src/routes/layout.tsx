import { $, component$, Slot, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { Link, useNavigate } from "@builder.io/qwik-city";
import { clearAuth, getLocale, getToken, getUser, setLocale, setUser } from "~/lib/auth";
import { api } from "~/lib/api";
import { getLocaleLabel, LANGUAGE_OPTIONS, t } from "~/lib/i18n";
import type { LanguageCode } from "~/lib/types";

const THEME_STORAGE_KEY = "theme";

export default component$(() => {
  const nav = useNavigate();
  const theme = useSignal<"light" | "dark">("light");
  const locale = useSignal<LanguageCode>("en");
  const isAuthed = useSignal(false);

  const applyTheme$ = $((nextTheme: "light" | "dark") => {
    theme.value = nextTheme;
    localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
    document.body.dataset.theme = nextTheme;
  });

  useVisibleTask$(() => {
    const savedTheme = localStorage.getItem(THEME_STORAGE_KEY);
    const initialTheme = savedTheme === "dark" ? "dark" : "light";
    theme.value = initialTheme;
    document.body.dataset.theme = initialTheme;

    locale.value = getLocale();

    const token = getToken();
    const isAuthRoute = window.location.pathname.startsWith("/auth");
    isAuthed.value = !!token;
    if (!token && !isAuthRoute) {
      nav("/auth/login");
    }
  });

  const handleLogout$ = $(() => {
    clearAuth();
    nav("/auth/login");
  });

  const user = useSignal(getUser());
  // Re-read user after hydration
  useVisibleTask$(() => {
    user.value = getUser();
  });

  const handleLanguageChange$ = $(async (next: LanguageCode) => {
    locale.value = next;
    setLocale(next);

    if (!isAuthed.value) {
      return;
    }

    try {
      await api.users.updatePreferredLanguage(next);
      if (user.value) {
        const updatedUser = { ...user.value, preferredLanguage: next };
        user.value = updatedUser;
        setUser(updatedUser);
      }
    } catch {
      // Keep UI language from header selection even if sync fails.
    }
  });

  return (
    <div class="min-h-screen bg-base-200">
      <header class="border-b border-base-300 bg-base-100 shadow-sm">
        <div class="navbar mx-auto max-w-6xl gap-4 px-4 sm:px-6 lg:px-8">
          <div class="navbar-start gap-3">
            <Link class="btn btn-ghost text-lg font-semibold normal-case" href="/">
              {t(locale.value, "header.brand")}
            </Link>
          </div>

          <div class="navbar-center hidden lg:flex">
            <ul class="menu menu-horizontal rounded-box bg-base-200 p-1">
              <li>
                <Link href="/">{t(locale.value, "header.home")}</Link>
              </li>
              <li>
                <Link href="/decks">{t(locale.value, "header.decks")}</Link>
              </li>
              <li>
                <Link href="/review">{t(locale.value, "header.review")}</Link>
              </li>
            </ul>
          </div>

          <div class="navbar-end gap-3">
            <div class="hidden items-center gap-2 sm:flex">
              <select
                class="select select-sm"
                value={locale.value}
                onChange$={(_, el) => handleLanguageChange$(el.value as LanguageCode)}
              >
                {LANGUAGE_OPTIONS.map((code) => (
                  <option key={code} value={code}>
                    {getLocaleLabel(code)}
                  </option>
                ))}
              </select>
            </div>

            <div class="hidden items-center gap-2 sm:flex">
              <label class="label cursor-pointer gap-3">
                <span class="text-sm">{t(locale.value, "header.light")}</span>
                <input
                  aria-label={t(locale.value, "header.theme")}
                  checked={theme.value === "dark"}
                  class="toggle"
                  type="checkbox"
                  onChange$={(_, el) => applyTheme$(el.checked ? "dark" : "light")}
                />
                <span class="text-sm">{t(locale.value, "header.dark")}</span>
              </label>
            </div>

            {isAuthed.value ? (
              <button class="btn btn-ghost btn-sm" onClick$={handleLogout$} type="button">
                {t(locale.value, "header.logout")}
              </button>
            ) : (
              <Link class="btn btn-primary btn-sm" href="/auth/login">
                {t(locale.value, "header.login")}
              </Link>
            )}

            {/* Mobile menu */}
            <div class="dropdown dropdown-end lg:hidden">
              <label class="btn btn-ghost" tabIndex={0}>
                {t(locale.value, "header.menu")}
              </label>
              <ul
                class="menu dropdown-content z-10 mt-3 w-56 rounded-box border border-base-300 bg-base-100 p-2 shadow-lg"
                tabIndex={0}
              >
                <li>
                  <Link href="/">{t(locale.value, "header.home")}</Link>
                </li>
                <li>
                  <Link href="/decks">{t(locale.value, "header.decks")}</Link>
                </li>
                <li>
                  <Link href="/review">{t(locale.value, "header.review")}</Link>
                </li>
                <li class="menu-title mt-2">{t(locale.value, "header.language") || "Language"}</li>
                <li>
                  <select
                    class="select select-sm w-full"
                    value={locale.value}
                    onChange$={(_, el) => handleLanguageChange$(el.value as LanguageCode)}
                  >
                    {LANGUAGE_OPTIONS.map((code) => (
                      <option key={code} value={code}>
                        {getLocaleLabel(code)}
                      </option>
                    ))}
                  </select>
                </li>
                <li class="menu-title mt-2">{t(locale.value, "header.theme")}</li>
                <li>
                  <button
                    class={{ active: theme.value === "light" }}
                    onClick$={() => applyTheme$("light")}
                    type="button"
                  >
                    {t(locale.value, "header.light")}
                  </button>
                </li>
                <li>
                  <button
                    class={{ active: theme.value === "dark" }}
                    onClick$={() => applyTheme$("dark")}
                    type="button"
                  >
                    {t(locale.value, "header.dark")}
                  </button>
                </li>
                {isAuthed.value && (
                  <li>
                    <button onClick$={handleLogout$} type="button">
                      {t(locale.value, "header.logout")}
                    </button>
                  </li>
                )}
              </ul>
            </div>
          </div>
        </div>
      </header>

      <div class="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
        <Slot />
      </div>
    </div>
  );
});
