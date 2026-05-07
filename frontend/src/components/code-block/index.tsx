import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import { codeToHtml } from "shiki";
import type { ContentType } from "~/lib/types";

interface Props {
  code: string;
  contentType: ContentType;
}

const langAliases: Record<string, string> = {
  plaintext: "text",
  plain: "text",
  txt: "text",
  js: "javascript",
  ts: "typescript",
  py: "python",
  golang: "go",
  csharp: "c#",
  cpp: "c++",
  md: "markdown",
};

function normalizeLanguage(contentType: ContentType) {
  const normalized = contentType.trim().toLowerCase();
  if (!normalized) {
    return "text";
  }
  return langAliases[normalized] ?? normalized;
}

export const CodeBlock = component$<Props>(({ code, contentType }) => {
  const html = useSignal("");

  useTask$(async ({ track }) => {
    const trackedCode = track(() => code);
    const trackedContentType = track(() => contentType);

    html.value = await codeToHtml(trackedCode, {
      lang: normalizeLanguage(trackedContentType),
      theme: "github-dark",
    });
  });

  return <div class="code-block overflow-x-auto text-sm" dangerouslySetInnerHTML={html.value} />;
});
