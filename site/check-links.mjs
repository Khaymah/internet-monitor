import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { checkSeo } from "./lib/seo-check.mjs";
import { resolve } from "./serve.mjs";

const DIST = fileURLToPath(new URL("./dist/", import.meta.url));
const BASE = `/${(process.env.SITE_BASE ?? "/internet-monitor/").replace(/^\/+|\/+$/g, "")}/`.replace(/^\/\/$/, "/");

function htmlFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
    e.isDirectory() ? htmlFiles(join(dir, e.name)) : e.name.endsWith(".html") ? [join(dir, e.name)] : [],
  );
}

const files = htmlFiles(DIST);
const ids = new Map();
for (const f of files) {
  const html = readFileSync(f, "utf8");
  ids.set(f, new Set([...html.matchAll(/\sid="([^"]+)"/g)].map((m) => m[1])));
}

const problems = [];
let checked = 0;
for (const f of files) {
  const html = readFileSync(f, "utf8");
  const where = relative(DIST, f);
  if (!/<html lang="[a-z]{2}" dir="(ltr|rtl)"/.test(html)) problems.push(`${where}: <html> has no lang/dir`);
  if (!f.endsWith("404.html") && !/<link rel="canonical"/.test(html)) problems.push(`${where}: no canonical link`);
  for (const m of html.matchAll(/<img\b[^>]*>/g)) {
    if (!/\balt="/.test(m[0])) problems.push(`${where}: image without alt: ${m[0].slice(0, 80)}`);
  }
  const refs = [...html.matchAll(/\s(?:href|src)="([^"]+)"/g)].map((m) => m[1].replace(/&amp;/g, "&"));
  for (const ref of refs) {
    if (/^(https?:|mailto:|data:)/.test(ref)) continue;
    checked++;
    if (ref.startsWith("#")) {
      if (ref.length > 1 && !ids.get(f).has(ref.slice(1))) problems.push(`${where}: missing anchor ${ref}`);
      continue;
    }
    if (!ref.startsWith(BASE)) {
      problems.push(`${where}: link outside the site base: ${ref}`);
      continue;
    }
    const [path, hash] = ref.split("#");
    const target = resolve(path);
    if (!target) {
      problems.push(`${where}: broken link ${ref}`);
      continue;
    }
    if (hash && target.endsWith(".html") && !ids.get(target)?.has(hash)) problems.push(`${where}: missing anchor ${ref}`);
  }
}

const seo = checkSeo(DIST, files, BASE);
problems.push(...seo.problems);

if (problems.length) {
  console.error(problems.join("\n"));
  console.error(`\n${problems.length} problem(s) in ${files.length} pages`);
  process.exit(1);
}
console.log(`links ok: ${checked} internal links in ${files.length} pages`);
const types = Object.entries(seo.stats.types)
  .map(([t, n]) => `${t} ${n}`)
  .join(", ");
console.log(`seo ok: meta tags on ${seo.stats.pages} pages, ${seo.stats.ldBlocks} JSON-LD blocks (${types}), ${seo.stats.sitemapUrls} sitemap urls`);
