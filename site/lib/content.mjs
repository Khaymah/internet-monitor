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

  function releaseNotes() {
    const dir = join(root, "docs", "releases");
    if (!existsSync(dir)) return [];
    return readdirSync(dir)
      .filter((f) => /^v\d.*\.md$/.test(f))
      .map((f) => ({ tag: f.replace(/\.md$/, ""), md: read("docs", "releases", f) }))
      .sort((a, b) => compareVersions(b.tag, a.tag));
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

  return { json, meta, releaseNotes, tracks, screenshots };
}
