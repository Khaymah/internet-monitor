import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { compareVersions } from "./util.mjs";

export function createContent(root) {
  const read = (...parts) => readFileSync(join(root, ...parts), "utf8");
  const json = (...parts) => JSON.parse(read(...parts));
  const appFile = (lang, file) => {
    const local = join(root, "web", "locales", lang, `${file}.json`);
    return JSON.parse(readFileSync(existsSync(local) ? local : join(root, "web", "locales", "en", `${file}.json`), "utf8"));
  };

  function meta(lang) {
    return appFile(lang, "meta");
  }

  function faqLayout() {
    const file = join(root, "web", "src", "content", "faq.ts");
    if (!existsSync(file)) return null;
    const src = readFileSync(file, "utf8");
    const sections = [];
    const re = /id:\s*"(\w+)"[^{}]*?title:\s*"faq\.section\.(\w+)"[^]*?items:\s*\[([^]*?)\]/g;
    for (const m of src.matchAll(re)) {
      const items = [...m[3].matchAll(/id:\s*"(\w+)"/g)].map((x) => x[1]);
      sections.push({ id: m[1], title: m[2], items });
    }
    return sections.length ? sections : null;
  }

  function faq(lang) {
    const data = appFile(lang, "faq");
    const entries = data.faq;
    const layout = faqLayout() ?? [
      { id: "all", title: null, items: Object.keys(entries).filter((k) => k !== "section") },
    ];
    return {
      title: data.faq_title,
      intro: data.faq_intro,
      eyebrow: data.faq_eyebrow,
      sections: layout
        .map((s) => ({
          id: s.id,
          title: s.title ? entries.section?.[s.title] : null,
          items: s.items.filter((id) => entries[id]?.q).map((id) => ({ id, q: entries[id].q, a: entries[id].a })),
        }))
        .filter((s) => s.items.length),
    };
  }

  function privacyUpdated() {
    const ts = join(root, "web", "src", "content", "legal.ts");
    if (existsSync(ts)) {
      const m = readFileSync(ts, "utf8").match(/PRIVACY[^=]*=\s*\{\s*updated:\s*"([\d-]+)"/);
      if (m) return m[1];
    }
    const md = existsSync(join(root, "PRIVACY.md")) ? read("PRIVACY.md") : "";
    const m = md.match(/Last updated:\s*([^\n]+)/);
    return m ? new Date(`${m[1]} UTC`).toISOString().slice(0, 10) : null;
  }

  function privacy(lang) {
    const data = appFile(lang, "legal");
    const { intro, ...rest } = data.privacy;
    const sections = Object.entries(rest).map(([id, s]) => {
      const keys = Object.keys(s);
      return {
        id,
        title: s.title,
        paragraphs: keys.filter((k) => /^p\d/.test(k)).map((k) => s[k]),
        list: keys.filter((k) => /^li\d/.test(k)).map((k) => s[k]),
      };
    });
    return { title: data.privacy_title, updatedLabel: data.legal_updated, toc: data.legal_toc, intro, sections };
  }

  function releaseNotes() {
    const dir = join(root, "docs", "releases");
    if (!existsSync(dir)) return [];
    return readdirSync(dir)
      .filter((f) => /^v\d.*\.md$/.test(f))
      .map((f) => ({ tag: f.replace(/\.md$/, ""), md: read("docs", "releases", f) }))
      .sort((a, b) => compareVersions(b.tag, a.tag));
  }

  function changelogLatest() {
    if (!existsSync(join(root, "CHANGELOG.md"))) return null;
    const text = read("CHANGELOG.md");
    const start = text.search(/^## /m);
    if (start < 0) return null;
    const rest = text.slice(start);
    const next = rest.slice(3).search(/^## /m);
    const section = next < 0 ? rest : rest.slice(0, next + 3);
    const [first, ...body] = section.split("\n");
    return { heading: first.replace(/^##\s+/, "").replace(/\[([^\]]+)\]/, "$1"), md: body.join("\n") };
  }

  function tracks(langs) {
    const dir = join(root, "web", "public", "tutorial");
    return langs.filter((l) => existsSync(join(dir, `${l}.vtt`))).map((l) => ({ lang: l, file: join(dir, `${l}.vtt`) }));
  }

  function screenshots(lang) {
    const pick = existsSync(join(root, "docs", "store", lang)) ? lang : "en";
    const dir = join(root, "docs", "store", pick);
    if (!existsSync(dir)) return { lang: pick, files: [] };
    const files = readdirSync(dir)
      .filter((f) => f.endsWith(".png"))
      .sort();
    return { lang: pick, files };
  }

  return { json, meta, faq, privacy, privacyUpdated, releaseNotes, changelogLatest, tracks, screenshots };
}
