import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import { codeToHtml } from "shiki";
import type { ContentType } from "~/lib/types";

interface Props {
  code: string;
  contentType: ContentType;
  highlightLines: number[];
}

const langMap: Record<ContentType, string> = {
  text: "text",
  code_go: "go",
  code_python: "python",
  code_js: "javascript",
  code_ts: "typescript",
  code_rust: "rust",
};

export const CodeBlock = component$<Props>(({ code, contentType, highlightLines }) => {
  const html = useSignal("");

  useTask$(async () => {
    html.value = await codeToHtml(code, {
      lang: langMap[contentType] ?? "text",
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
