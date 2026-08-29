import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import type { ContentType } from "~/lib/types";
import { renderContentMarkdown } from "~/lib/markdown";

interface Props {
  code: string;
  contentType: ContentType;
}

export const CodeBlock = component$<Props>(({ code, contentType }) => {
  const html = useSignal("");

  useTask$(async ({ track }) => {
    const trackedCode = track(() => code);
    const trackedContentType = track(() => contentType);

    html.value = await renderContentMarkdown(trackedCode, trackedContentType);
  });

  return (
    <div
      class="code-block prose prose-invert prose-sm max-w-none overflow-x-auto px-4 py-3 text-sm"
      dangerouslySetInnerHTML={html.value}
    />
  );
});
