import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import { codeToHtml } from "shiki";
import type { ContentType } from "~/lib/types";

interface Props {
  code: string;
  contentType: ContentType;
  highlightLines: number[];
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

export const CodeBlock = component$<Props>(({ code, contentType, highlightLines }) => {
  const html = useSignal("");

  useTask$(async ({ track }) => {
    const trackedCode = track(() => code);
    const trackedContentType = track(() => contentType);
    const trackedHighlightLines = track(() => highlightLines.join(","));
    const highlightLineSet = new Set(
      trackedHighlightLines
        .split(",")
        .filter(Boolean)
        .map((value) => Number(value)),
    );

    html.value = await codeToHtml(trackedCode, {
      lang: normalizeLanguage(trackedContentType),
      theme: "github-dark",
      transformers: [
        {
          line(node, line) {
            if (highlightLineSet.has(line)) {
              (node.properties as Record<string, string>)["data-highlighted"] = "true";
            }
          },
        },
      ],
    });
  });

  return <div class="code-block overflow-x-auto text-sm" dangerouslySetInnerHTML={html.value} />;
});
