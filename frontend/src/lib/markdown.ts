import MarkdownIt from "markdown-it";
import markdownItShiki from "@shikijs/markdown-it";
import type { BundledLanguage } from "shiki";
import type { ContentType } from "~/lib/types";

// Languages actually used across info object content in this app. Kept explicit
// (rather than loading Shiki's full bundled language set) to keep highlighter
// startup cheap.
const SHIKI_LANGS = [
  "typescript",
  "javascript",
  "tsx",
  "jsx",
  "python",
  "go",
  "json",
  "bash",
  "sql",
  "yaml",
  "html",
  "css",
  "markdown",
] satisfies BundledLanguage[];

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

function normalizeLanguage(contentType: ContentType): string {
  const normalized = contentType.trim().toLowerCase();
  if (!normalized) {
    return "text";
  }
  return langAliases[normalized] ?? normalized;
}

let rendererPromise: Promise<InstanceType<typeof MarkdownIt>> | null = null;

function getRenderer() {
  if (!rendererPromise) {
    rendererPromise = (async () => {
      // html: false is deliberate — info object content can originate from any
      // user's own deck and later reach other users via community-published
      // premade decks, so raw HTML in the source must never be passed through.
      const md = new MarkdownIt({ html: false });
      const shikiPlugin = await markdownItShiki({
        theme: "github-dark",
        langs: SHIKI_LANGS,
        langAlias: langAliases,
      });
      md.use(shikiPlugin);
      return md;
    })();
  }
  return rendererPromise;
}

/**
 * Renders info object `content` as HTML. Content authored as Markdown (headers,
 * lists, tables, fenced code blocks) is parsed as such, with each fenced code
 * block highlighted by its own declared language. Content with no fences at all
 * whose contentType names a known language is treated as a single code block in
 * that language, matching how such content used to be rendered directly by Shiki.
 * Falls back to escaped plain text if rendering fails for any reason (e.g. a
 * community-submitted deck using an unrecognized fence language).
 */
export async function renderContentMarkdown(content: string, contentType: ContentType): Promise<string> {
  const md = await getRenderer();
  const normalized = normalizeLanguage(contentType);
  const hasFence = content.includes("```");
  const isKnownLanguage = (SHIKI_LANGS as readonly string[]).includes(normalized);

  const source = !hasFence && isKnownLanguage ? "```" + normalized + "\n" + content + "\n```" : content;

  try {
    return md.render(source);
  } catch {
    return md.render(content.replace(/```[^\n]*/g, "```"));
  }
}
