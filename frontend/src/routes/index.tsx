import { component$ } from "@builder.io/qwik";
import { Link, type DocumentHead } from "@builder.io/qwik-city";
import { t } from "~/lib/i18n";

export default component$(() => {
  const locale = "ru" as const;
  const featureKeys = ["home.card1", "home.card2", "home.card3"] as const;
  const listKeys = ["home.list1", "home.list2", "home.list3", "home.list4"] as const;
  const stepKeys = ["home.step1", "home.step2", "home.step3"] as const;

  return (
    <main class="relative overflow-hidden">
      <div class="pointer-events-none absolute inset-x-0 top-0 h-96 bg-gradient-to-b from-primary/14 via-secondary/8 to-transparent" />
      <div class="pointer-events-none absolute -left-16 top-20 h-56 w-56 rounded-full bg-secondary/18 blur-3xl" />
      <div class="pointer-events-none absolute right-0 top-40 h-72 w-72 rounded-full bg-primary/14 blur-3xl" />

      <div class="relative mx-auto flex min-h-screen w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8 lg:py-10">
        <section class="hero rounded-[2rem] border border-base-300/70 bg-base-100/85 shadow-2xl shadow-base-content/5 backdrop-blur">
          <div class="hero-content grid w-full items-start gap-8 px-6 py-8 sm:px-8 lg:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)] lg:px-10 lg:py-12">
            <div class="max-w-3xl">
              <div class="badge badge-primary badge-outline badge-lg rounded-full px-4 py-3 font-medium">
                {t(locale, "home.badge")}
              </div>

              <h1 class="font-display mt-5 max-w-[12ch] text-5xl leading-none font-semibold text-balance sm:text-6xl lg:text-7xl">
                {t(locale, "home.title")}
              </h1>

              <p class="mt-5 max-w-2xl text-base leading-8 text-base-content/70 sm:text-lg">
                {t(locale, "home.subtitle")}
              </p>

              <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <Link class="btn btn-primary btn-lg rounded-full px-7" href="/decks">
                  {t(locale, "home.primaryCta")}
                </Link>
                <Link class="btn btn-ghost btn-lg rounded-full border border-base-300 bg-base-100/60 px-7" href="/review">
                  {t(locale, "home.secondaryCta")}
                </Link>
              </div>
            </div>

            <div class="card border border-base-300/70 bg-base-200/80 shadow-xl">
              <div class="card-body gap-5">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-xs font-semibold uppercase tracking-[0.24em] text-primary/80">
                    {t(locale, "home.panelLabel")}
                  </span>
                  <div class="badge badge-secondary badge-soft">{t(locale, "home.panelLocale")}</div>
                </div>

                <div class="stats stats-vertical bg-base-100 shadow-sm">
                  <div class="stat px-5 py-4">
                    <div class="stat-title">{t(locale, "home.stat1.title")}</div>
                    <div class="stat-value text-2xl">{t(locale, "home.stat1.value")}</div>
                    <div class="stat-desc">{t(locale, "home.stat1.desc")}</div>
                  </div>
                  <div class="stat px-5 py-4">
                    <div class="stat-title">{t(locale, "home.stat2.title")}</div>
                    <div class="stat-value text-2xl">{t(locale, "home.stat2.value")}</div>
                    <div class="stat-desc">{t(locale, "home.stat2.desc")}</div>
                  </div>
                </div>

                <ul class="steps steps-vertical text-sm">
                  {stepKeys.map((stepKey, index) => (
                    <li key={stepKey} class={{ step: true, "step-primary": index < 2 }}>
                      {t(locale, stepKey)}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </section>

        <section class="grid gap-4 lg:grid-cols-3">
          {featureKeys.map((featureKey, index) => (
            <article
              key={featureKey}
              class="card border border-base-300/70 bg-base-100/88 shadow-lg shadow-base-content/5 transition-transform duration-200 hover:-translate-y-1"
            >
              <div class="card-body gap-4">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-xs font-semibold uppercase tracking-[0.22em] text-primary/80">
                    0{index + 1}
                  </span>
                  <div class="badge badge-ghost badge-sm">
                    {t(locale, "home.stepBadge")} {index + 1}
                  </div>
                </div>
                <h2 class="font-display text-3xl leading-tight text-balance">
                  {t(locale, `${featureKey}.title`)}
                </h2>
                <p class="leading-7 text-base-content/70">
                  {t(locale, `${featureKey}.text`)}
                </p>
              </div>
            </article>
          ))}
        </section>

        <section class="grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(280px,0.75fr)]">
          <article class="card overflow-hidden border border-base-300/70 bg-neutral text-neutral-content shadow-xl shadow-base-content/10">
            <div class="card-body gap-5 p-8 sm:p-10">
              <p class="text-xs font-semibold uppercase tracking-[0.24em] text-neutral-content/70">
                {t(locale, "home.noteTitle")}
              </p>
              <p class="font-display max-w-3xl text-3xl leading-tight text-balance sm:text-4xl lg:text-5xl">
                {t(locale, "home.noteText")}
              </p>
            </div>
          </article>

          <article class="card border border-base-300/70 bg-base-100/88 shadow-lg shadow-base-content/5">
            <div class="card-body p-7">
              <p class="text-xs font-semibold uppercase tracking-[0.24em] text-primary/80">
                {t(locale, "home.listTitle")}
              </p>
              <div class="divider my-1" />
              <ul class="space-y-3">
                {listKeys.map((listKey) => (
                  <li key={listKey} class="flex items-start gap-3 text-sm leading-6 text-base-content/75 sm:text-base">
                    <span class="badge badge-primary badge-xs mt-2 shrink-0" />
                    <span>{t(locale, listKey)}</span>
                  </li>
                ))}
              </ul>
            </div>
          </article>
        </section>
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Shmanki - обучение с интервальными повторениями",
  meta: [
    {
      name: "description",
      content:
        "Главная страница Shmanki с вводным текстом о колодах, карточках и интервальном повторении.",
    },
  ],
};
