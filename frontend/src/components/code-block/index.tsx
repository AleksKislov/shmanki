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

  useTask$(async () => {
    html.value = await codeToHtml(code, {
      lang: normalizeLanguage(contentType),
      theme: "github-dark",
      transformers: [
        {
          line(node, line) {
            if (highlightLines.includes(line)) {
              (node.properties as Record<string, string>)["data-highlighted"] = "true";
            }
          },
        },
      ],
    });
  });

  return <div class="code-block overflow-x-auto text-sm" dangerouslySetInnerHTML={html.value} />;
});
