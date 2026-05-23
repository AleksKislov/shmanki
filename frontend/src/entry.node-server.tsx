import { createServer } from "node:http";
import { createQwikCity } from "@builder.io/qwik-city/middleware/node";
import qwikCityPlan from "@qwik-city-plan";
import render from "./entry.ssr";

const { router, notFound, staticFile } = createQwikCity({
  render,
  qwikCityPlan,
  static: {
    root: "./dist",
    cacheControl: "public, max-age=600",
  },
});

const PORT = parseInt(process.env.PORT ?? "3000", 10);

const server = createServer((req, res) => {
  staticFile(req, res, () => {
    router(req, res, () => {
      notFound(req, res, () => {});
    });
  });
});

server.listen(PORT, () => {
  console.log(`Frontend server listening on http://0.0.0.0:${PORT}`);
});
