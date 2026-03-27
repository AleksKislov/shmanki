import { component$ } from "@builder.io/qwik";
import { Link, type DocumentHead } from "@builder.io/qwik-city";
import { t } from "~/lib/i18n";

export default component$(() => {
  const locale = "ru" as const;
  const featureKeys = ["home.card1", "home.card2", "home.card3"] as const;
  const listKeys = ["home.list1", "home.list2", "home.list3", "home.list4"] as const;
  const stepKeys = ["home.step1", "home.step2", "home.step3"] as const;

  return (
    <main class="flex flex-col gap-6">
      <section class="hero rounded-box border border-base-300 bg-base-100 shadow-sm">
        <div class="hero-content grid w-full items-start gap-6 p-6 lg:grid-cols-2 lg:p-10">
            <div class="max-w-3xl">
              <div class="badge badge-primary badge-outline badge-lg">
                {t(locale, "home.badge")}
              </div>

              <h1 class="font-display mt-5 text-4xl leading-tight sm:text-5xl lg:text-6xl">
                {t(locale, "home.title")}
              </h1>

              <p class="mt-5 max-w-2xl text-base leading-8 text-base-content/70">
                {t(locale, "home.subtitle")}
              </p>

              <div class="mt-8 flex flex-col gap-3 sm:flex-row">
                <Link class="btn btn-primary btn-wide" href="/decks">
                  {t(locale, "home.primaryCta")}
                </Link>
                <Link class="btn btn-outline btn-wide" href="/review">
                  {t(locale, "home.secondaryCta")}
                </Link>
              </div>
            </div>

            <div class="card border border-base-300 bg-base-200 shadow-sm">
              <div class="card-body gap-5">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-xs font-semibold uppercase tracking-[0.24em] text-primary/80">
                    {t(locale, "home.panelLabel")}
                  </span>
                  <div class="badge badge-secondary badge-soft">{t(locale, "home.panelLocale")}</div>
                </div>

                <div class="stats stats-vertical border border-base-300 bg-base-100">
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
          <article key={featureKey} class="card border border-base-300 bg-base-100 shadow-sm">
            <div class="card-body gap-4">
              <div class="flex items-center justify-between gap-3">
                <span class="badge badge-primary badge-outline">0{index + 1}</span>
                <div class="badge badge-ghost">
                  {t(locale, "home.stepBadge")} {index + 1}
                </div>
              </div>
              <h2 class="font-display card-title text-2xl">
                {t(locale, `${featureKey}.title`)}
              </h2>
              <p class="leading-7 text-base-content/70">
                {t(locale, `${featureKey}.text`)}
              </p>
            </div>
          </article>
        ))}
      </section>

      <section class="grid gap-4 lg:grid-cols-[2fr_1fr]">
        <article class="card border border-base-300 bg-neutral text-neutral-content shadow-sm">
          <div class="card-body gap-5 p-8">
            <p class="badge badge-outline badge-lg">{t(locale, "home.noteTitle")}</p>
            <p class="font-display text-3xl leading-tight sm:text-4xl">
              {t(locale, "home.noteText")}
            </p>
          </div>
        </article>

        <article class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body p-7">
            <h2 class="card-title">{t(locale, "home.listTitle")}</h2>
            <ul class="menu rounded-box bg-base-200">
              {listKeys.map((listKey) => (
                <li key={listKey}>
                  <span>{t(locale, listKey)}</span>
                </li>
              ))}
            </ul>
          </div>
        </article>
      </section>
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
