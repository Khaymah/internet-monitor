import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { appFormat, loadApp } from "./lib/app.mjs";
import { appCss } from "./lib/app-css.mjs";
import { createContent } from "./lib/content.mjs";
import { PAGES, THEME_BOOT, pageUrl, shell } from "./lib/layout.mjs";
import { renderAbout, renderContact, renderFaq, renderPrivacy, renderSupport, renderTerms } from "./lib/page-app.mjs";
import { renderChangelog } from "./lib/page-changelog.mjs";
import { renderDownload } from "./lib/page-download.mjs";
import { renderGuide, renderNotFound, renderTutorial } from "./lib/page-docs.mjs";
import { renderHome } from "./lib/page-home.mjs";
import { loadReleases } from "./lib/releases.mjs";
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
  const tokens = readFileSync(join(ROOT, "web", "src", "styles", "tokens.css"), "utf8");
  const app = appCss(join(ROOT, "web", "src", "styles", "app.css"));
  const own = ["site.css", "app-pages.css", "footer.css", "pages.css", "nav.css", "content.css"].map((f) =>
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
  const tracks = content.tracks(LANGS);
  for (const tr of tracks) {
    mkdirSync(join(OUT, "tutorial"), { recursive: true });
    copyFileSync(tr.file, join(OUT, "tutorial", `${tr.lang}.vtt`));
  }
  return { css: hash(css), js: scripts["site.js"], scripts, tracks };
}

function makeCtx(lang, dict, site, releases, tracks, appData) {
  const intl = site.meta[lang].intl;
  const base = site.base;
  const t = (key, vars) => {
    const value = lookup(dict, key);
    if (typeof value !== "string") throw new Error(`missing string ${key} in ${lang}.json`);
    return fill(value, vars);
  };
  return {
    lang,
    site,
    releases,
    tracks,
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
  };
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
  const f = ctx.shots.files[0];
  return f ? `${ctx.site.base}assets/shots/${ctx.shots.lang}/${f}` : `${ctx.site.base}assets/favicon.png`;
}

function rootPage(site, dicts) {
  const links = site.langs
    .map(
      (l) =>
        `<li><a class="card" href="${site.base}${l}/" hreflang="${l}" lang="${l}" dir="${site.meta[l].dir}" data-lang="${l}">${esc(site.meta[l].name)}<span>${esc(dicts[l].root.pick)}</span></a></li>`,
    )
    .join("");
  const alternates = site.langs
    .map((l) => `<link rel="alternate" hreflang="${l}" href="${site.origin}${site.base}${l}/">`)
    .join("\n");
  const redirect = `(function(){var langs="${site.langs.join(",")}".split(",");var pick=null;try{var s=localStorage.getItem("im.site.lang");if(langs.indexOf(s)>=0)pick=s;}catch(e){}if(!pick){var list=navigator.languages&&navigator.languages.length?navigator.languages:[navigator.language||"en"];for(var i=0;i<list.length&&!pick;i++){var c=String(list[i]).toLowerCase().split("-")[0];if(langs.indexOf(c)>=0)pick=c;}}location.replace("${site.base}"+(pick||"en")+"/"+location.search+location.hash);})();`;
  return `<!doctype html>
<html lang="en" dir="ltr" class="no-js" data-theme="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(dicts.en.root.title)}</title>
<meta name="description" content="${esc(dicts.en.home.description)}">
<link rel="canonical" href="${site.origin}${site.base}">
${alternates}
<link rel="alternate" hreflang="x-default" href="${site.origin}${site.base}">
<meta property="og:title" content="${esc(dicts.en.home.title)}">
<meta property="og:description" content="${esc(dicts.en.home.description)}">
<meta property="og:url" content="${site.origin}${site.base}">
<meta property="og:image" content="${site.origin}${site.base}assets/shots/en/1-overview.png">
<meta name="color-scheme" content="dark light">
<link rel="icon" type="image/png" href="${site.base}assets/favicon.png">
<link rel="stylesheet" href="${site.base}assets/site.css?v=${site.hash.css}">
<script>${THEME_BOOT}${redirect}</script>
</head>
<body class="root-page">
<main class="wrap root-main">
<img src="${site.base}assets/favicon.png" alt="${esc(dicts.en.site.logo_alt)}" width="72" height="72">
<h1 class="aurora-text">${esc(dicts.en.site.name)}</h1>
<p class="lead">${esc(dicts.en.home.hero_title)}</p>
<h2 class="sr-only">${esc(dicts.en.root.pick)}</h2>
<ul class="root-langs">${links}</ul>
</main>
</body>
</html>
`;
}

function sitemap(site) {
  const urls = PAGES.map((p) =>
    site.langs
      .map((l) => {
        const alts = site.langs
          .map((a) => `<xhtml:link rel="alternate" hreflang="${a}" href="${site.origin}${pageUrl(site.base, a, p.id)}"/>`)
          .join("");
        return `<url><loc>${site.origin}${pageUrl(site.base, l, p.id)}</loc>${alts}</url>`;
      })
      .join("\n"),
  ).join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${urls}
</urlset>
`;
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
  let count = 0;
  for (const lang of LANGS) {
    const ctx = makeCtx(lang, dicts[lang], site, releases, assets.tracks, appData);
    for (const page of PAGES) {
      const out = RENDER[page.id](ctx);
      const html = shell(ctx, { pageId: page.id, image: ogImage(ctx), ...out });
      write(join(OUT, lang, page.path, "index.html"), html);
      count++;
    }
  }
  const enCtx = makeCtx("en", dicts.en, site, releases, assets.tracks, appData);
  const nf = renderNotFound(enCtx);
  write(join(OUT, "404.html"), shell(enCtx, { pageId: "notfound", image: ogImage(enCtx), notFound: true, ...nf }));
  write(join(OUT, "index.html"), rootPage(site, dicts));
  write(join(OUT, "sitemap.xml"), sitemap(site));
  write(join(OUT, "robots.txt"), `User-agent: *\nAllow: /\n\nSitemap: ${ORIGIN}${BASE}sitemap.xml\n`);
  write(join(OUT, ".nojekyll"), "");
  const rel = releases.ok
    ? `stable ${releases.stable?.tag ?? "none"}, preview ${releases.preview?.tag ?? "none"}, video ${releases.tutorial ? releases.tutorial.release.tag : "none"}`
    : "release data unavailable, using fallback links";
  console.log(`site: ${count} pages in ${LANGS.length} languages → ${OUT} (${rel}; subtitles: ${assets.tracks.map((x) => x.lang).join(", ") || "none"})`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
