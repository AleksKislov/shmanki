import { createQwikCity } from "@builder.io/qwik-city/middleware/node";
import qwikCityPlan from "@qwik-city-plan";
import render from "./entry.ssr";

/** The default export is the QwikCity adapter for Node.js */
const { router, notFound, staticFile } = createQwikCity({
  render,
  qwikCityPlan,
});

export { router, notFound, staticFile };
