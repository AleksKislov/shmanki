import { $, component$, Slot, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { Link } from "@builder.io/qwik-city";
import { t } from "~/lib/i18n";

const THEME_STORAGE_KEY = "theme";

export default component$(() => {
  const locale = "ru" as const;
  const theme = useSignal<"light" | "dark">("light");

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
  });

  return (
    <div class='min-h-screen bg-base-200'>
      <header class='border-b border-base-300 bg-base-100 shadow-sm'>
        <div class='navbar mx-auto max-w-6xl gap-4 px-4 sm:px-6 lg:px-8'>
          <div class='navbar-start gap-3'>
            <Link class='btn btn-ghost text-lg font-semibold normal-case' href='/'>
              {t(locale, "header.brand")}
            </Link>
          </div>

          <div class='navbar-center hidden lg:flex'>
            <ul class='menu menu-horizontal rounded-box bg-base-200 p-1'>
              <li>
                <Link href='/'>{t(locale, "header.home")}</Link>
              </li>
              <li>
                <Link href='/decks'>{t(locale, "header.decks")}</Link>
              </li>
              <li>
                <Link href='/review'>{t(locale, "header.review")}</Link>
              </li>
            </ul>
          </div>

          <div class='navbar-end gap-3'>
            <div class='hidden items-center gap-2 sm:flex'>
              <span class='text-sm text-base-content/70'>{t(locale, "header.theme")}</span>
              <label class='label cursor-pointer gap-3'>
                <span class='text-sm'>{t(locale, "header.light")}</span>
                <input
                  aria-label={t(locale, "header.theme")}
                  checked={theme.value === "dark"}
                  class='toggle'
                  type='checkbox'
                  onChange$={(_, currentTarget) =>
                    applyTheme$(currentTarget.checked ? "dark" : "light")
                  }
                />
                <span class='text-sm'>{t(locale, "header.dark")}</span>
              </label>
            </div>

            <div class='dropdown dropdown-end lg:hidden'>
              <label class='btn btn-ghost' tabIndex={0}>
                {t(locale, "header.menu")}
              </label>
              <ul
                class='menu dropdown-content z-10 mt-3 w-56 rounded-box border border-base-300 bg-base-100 p-2 shadow-lg'
                tabIndex={0}
              >
                <li>
                  <Link href='/'>{t(locale, "header.home")}</Link>
                </li>
                <li>
                  <Link href='/decks'>{t(locale, "header.decks")}</Link>
                </li>
                <li>
                  <Link href='/review'>{t(locale, "header.review")}</Link>
                </li>
                <li class='menu-title mt-2'>{t(locale, "header.theme")}</li>
                <li>
                  <button
                    class={{ active: theme.value === "light" }}
                    onClick$={() => applyTheme$("light")}
                    type='button'
                  >
                    {t(locale, "header.light")}
                  </button>
                </li>
                <li>
                  <button
                    class={{ active: theme.value === "dark" }}
                    onClick$={() => applyTheme$("dark")}
                    type='button'
                  >
                    {t(locale, "header.dark")}
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </header>

      <div class='mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8'>
        <Slot />
      </div>
    </div>
  );
});
