import { createHash } from "node:crypto";
import { copyFileSync, cpSync, existsSync, mkdirSync, readFileSync, rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { appFormat, loadApp } from "./lib/app.mjs";
import { appCss } from "./lib/app-css.mjs";
import { createContent } from "./lib/content.mjs";
import { navLabel, PAGES, pageUrl, shell } from "./lib/layout.mjs";
import { renderAbout, renderContact, renderFaq, renderPrivacy, renderSupport, renderTerms } from "./lib/page-app.mjs";
import { renderChangelog } from "./lib/page-changelog.mjs";
import { renderDownload } from "./lib/page-download.mjs";
import { renderGuide, renderNotFound } from "./lib/page-docs.mjs";
import { renderHome } from "./lib/page-home.mjs";
import { renderTutorial, resolveTours } from "./lib/page-tours.mjs";
import { loadReleases } from "./lib/releases.mjs";
import { pageDates, withIntro } from "./lib/seo.mjs";
import { llmsFull, llmsTxt, robots, rootPage, sitemap } from "./lib/seo-files.mjs";
import { esc, fill, formatDate, formatSize, inline, lookup, shape, write } from "./lib/util.mjs";

const SITE_DIR = dirname(fileURLToPath(import.meta.url));
const ROOT = join(SITE_DIR, "..");
const OUT = join(SITE_DIR, "dist");
const LANGS = ["en", "ar", "fr", "tr", "ur"];
const REPO = process.env.SITE_REPO || "Khaymah/internet-monitor";
const BASE = `/${(process.env.SITE_BASE ?? "/internet-monitor/").replace(/^\/+|\/+$/g, "")}/`.replace(/^\/\/$/, "/");
const ORIGIN = (process.env.SITE_ORIGIN || "https://khaymah.github.io").replace(/\/+$/, "");

const content = createContent(ROOT);

function loadLocales() {
  const dicts = {};
  for (const lang of LANGS) dicts[lang] = JSON.parse(readFileSync(join(SITE_DIR, "locales", `${lang}.json`), "utf8"));
  const reference = shape(dicts.en);
  for (const lang of LANGS) {
    if (shape(dicts[lang]) !== reference) {
      throw new Error(`site/locales/${lang}.json does not have the same keys, order and list lengths as en.json`);
    }
  }
  return dicts;
}

function hash(text) {
  return createHash("sha256").update(text).digest("hex").slice(0, 10);
}

function fontCss() {
  const sources = [join(SITE_DIR, "node_modules", "@fontsource"), join(ROOT, "web", "node_modules", "@fontsource")];
  const dir = sources.find((d) => existsSync(join(d, "poppins")));
  if (!dir) {
    console.warn("fonts: @fontsource packages not installed, run `bun install` in site/. Using system fonts.");
    return "";
  }
  const wanted = [
    ["poppins", ["400", "500", "600"], /-(latin|latin-ext)-\d+-normal \*\//],
    ["ibm-plex-sans-arabic", ["400", "500", "600"], /-arabic-\d+-normal \*\//],
  ];
  let css = "";
  for (const [pkg, weights, subset] of wanted) {
    for (const w of weights) {
      const blocks = readFileSync(join(dir, pkg, `${w}.css`), "utf8").split(/(?=\/\* )/);
      for (const block of blocks.filter((b) => subset.test(b.split("\n")[0]))) {
        const file = block.match(/url\(\.\/files\/([^)]+\.woff2)\)/)[1];
        copyFileSync(join(dir, pkg, "files", file), join(OUT, "assets", "fonts", file));
        css += `${block
          .replace(/src:[^;]+;/, `src: url(fonts/${file}) format("woff2");`)
          .trim()}\n`;
      }
    }
  }
  return css;
}

function copyAssets() {
  mkdirSync(join(OUT, "assets", "fonts"), { recursive: true });
  copyFileSync(join(ROOT, "internal", "assets", "favicon.png"), join(OUT, "assets", "favicon.png"));
  cpSync(join(SITE_DIR, "assets", "og"), join(OUT, "assets", "og"), { recursive: true });
  const tokens = readFileSync(join(ROOT, "web", "src", "styles", "tokens.css"), "utf8");
  const app = appCss(join(ROOT, "web", "src", "styles", "app.css"));
  const own = ["site.css", "app-pages.css", "footer.css", "pages.css", "nav.css", "content.css", "tours.css"].map((f) =>
    readFileSync(join(SITE_DIR, "assets", f), "utf8"),
  );
  const css = [fontCss(), tokens, app, ...own].join("\n");
  write(join(OUT, "assets", "site.css"), css);
  const scripts = {};
  for (const f of ["site.js", "changelog.js"]) {
    const js = readFileSync(join(SITE_DIR, "assets", f), "utf8");
    write(join(OUT, "assets", f), js);
    scripts[f] = hash(js);
  }
  const shotLangs = new Set(LANGS.map((l) => content.screenshots(l).lang));
  for (const l of shotLangs) {
    for (const f of content.screenshots(l).files) {
      mkdirSync(join(OUT, "assets", "shots", l), { recursive: true });
      copyFileSync(join(ROOT, "docs", "store", l, f), join(OUT, "assets", "shots", l, f));
    }
  }
  const tours = content.tours(LANGS);
  for (const tour of tours) {
    const dir = join(OUT, "tutorial", tour.id);
    mkdirSync(dir, { recursive: true });
    for (const tr of tour.tracks) copyFileSync(tr.file, join(dir, `${tr.lang}.vtt`));
    if (tour.poster) copyFileSync(tour.poster, join(dir, "poster.jpg"));
  }
  return { css: hash(css), js: scripts["site.js"], scripts, tours };
}

function makeCtx(lang, dict, site, releases, tours, appData, extra) {
  const intl = site.meta[lang].intl;
  const base = site.base;
  const t = (key, vars) => {
    const value = lookup(dict, key);
    if (typeof value !== "string") throw new Error(`missing string ${key} in ${lang}.json`);
    return fill(value, vars);
  };
  const ctx = {
    lang,
    site,
    releases,
    tours: tours.map((tour) => ({ ...tour, text: content.tourText(lang, tour.id) })),
    t,
    e: (key, vars) => esc(t(key, vars)),
    md: (key, vars) => inline(t(key, vars)),
    inline: (text) => inline(text),
    raw: (key) => lookup(dict, key),
    date: (iso) => formatDate(intl, iso),
    size: (bytes) => formatSize(intl, bytes),
    shots: content.screenshots(lang),
    app: {
      releaseNotes: content.releaseNotes(),
    },
    appx: {
      data: appData,
      lucide: appData.lucide,
      svg: appData.svg,
      repoUrl: appData.repoUrl,
      t: appData.translator(lang),
      en: appData.translator("en"),
      fmt: appFormat(intl),
      href: (id, anchor) => `${pageUrl(base, lang, id)}${anchor ? `#${anchor}` : ""}`,
    },
    url: (id) => pageUrl(base, lang, id),
    ...extra,
  };
  ctx.navLabel = (id) => navLabel(ctx, id);
  return ctx;
}

const RENDER = {
  home: renderHome,
  download: renderDownload,
  guide: renderGuide,
  tutorial: renderTutorial,
  faq: renderFaq,
  privacy: renderPrivacy,
  changelog: renderChangelog,
  about: renderAbout,
  contact: renderContact,
  support: renderSupport,
  terms: renderTerms,
};

function ogImage(ctx) {
  return `${ctx.site.base}assets/og/${ctx.lang}.png`;
}

async function main() {
  const dicts = loadLocales();
  rmSync(OUT, { recursive: true, force: true });
  mkdirSync(OUT, { recursive: true });
  const meta = Object.fromEntries(LANGS.map((l) => [l, content.meta(l)]));
  const releases = await loadReleases(REPO, {
    token: process.env.GITHUB_TOKEN,
    offline: process.env.SITE_OFFLINE === "1",
  });
  const appData = await loadApp(ROOT, SITE_DIR);
  const assets = copyAssets();
  const site = {
    base: BASE,
    origin: ORIGIN,
    langs: LANGS,
    meta,
    repoUrl: `https://github.com/${REPO}`,
    year: new Date().getUTCFullYear(),
    hash: { css: assets.css, js: assets.js, ...assets.scripts },
  };
  const tours = resolveTours(releases, assets.tours);
  const extra = { dates: pageDates(ROOT, releases, appData, tours) };
  let count = 0;
  for (const lang of LANGS) {
    const ctx = makeCtx(lang, dicts[lang], site, releases, tours, appData, extra);
    for (const page of PAGES) {
      const out = RENDER[page.id](ctx);
      out.body = withIntro(ctx, page.id, out.body);
      const html = shell(ctx, { pageId: page.id, image: ogImage(ctx), ...out });
      write(join(OUT, lang, page.path, "index.html"), html);
      count++;
    }
  }
  const enCtx = makeCtx("en", dicts.en, site, releases, tours, appData, extra);
  const nf = renderNotFound(enCtx);
  write(join(OUT, "404.html"), shell(enCtx, { pageId: "notfound", image: ogImage(enCtx), notFound: true, ...nf }));
  write(join(OUT, "index.html"), rootPage(site, dicts));
  write(join(OUT, "sitemap.xml"), sitemap(site, extra.dates));
  write(join(OUT, "robots.txt"), robots(site));
  write(join(OUT, "llms.txt"), llmsTxt(enCtx));
  write(join(OUT, "llms-full.txt"), llmsFull(enCtx));
  write(join(OUT, ".nojekyll"), "");
  const rel = releases.ok
    ? `stable ${releases.stable?.tag ?? "none"}, preview ${releases.preview?.tag ?? "none"}, videos ${tours.filter((t) => t.video).map((t) => `${t.id}@${t.video.release.tag}`).join(" ") || "none"}`
    : "release data unavailable, using fallback links";
  console.log(`site: ${count} pages in ${LANGS.length} languages → ${OUT} (${rel}; tours: ${tours.length}, subtitles: ${tours.reduce((n, t) => n + t.tracks.length, 0)})`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
