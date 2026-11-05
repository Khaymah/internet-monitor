const { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } = require("node:fs");
const { join } = require("node:path");
const { pathToFileURL } = require("node:url");
const { chromium } = require("playwright");

const SITE = join(__dirname, "..");
const ROOT = join(SITE, "..");
const DIST = join(SITE, "dist", "assets");
const OUT = join(SITE, "assets", "og");
const LANGS = ["en", "ar", "fr", "tr", "ur"];

function esc(s) {
  return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function page(lang) {
  const dict = JSON.parse(readFileSync(join(SITE, "locales", `${lang}.json`), "utf8"));
  const meta = JSON.parse(readFileSync(join(ROOT, "web", "locales", lang, "meta.json"), "utf8"));
  const shotDir = existsSync(join(ROOT, "docs", "store", lang)) ? lang : "en";
  const shot = pathToFileURL(join(ROOT, "docs", "store", shotDir, "1-overview.png")).href;
  const logo = pathToFileURL(join(ROOT, "internal", "assets", "favicon.png")).href;
  const css = pathToFileURL(join(DIST, "site.css")).href;
  return `<!doctype html>
<html lang="${lang}" dir="${meta.dir}" data-theme="dark">
<head>
<meta charset="utf-8">
<link rel="stylesheet" href="${css}">
<style>
html, body { margin: 0; width: 1200px; height: 630px; overflow: hidden; }
body {
  display: flex; align-items: center; gap: 48px; padding: 0 0 0 72px;
  background: var(--color-canvas);
  background-image:
    radial-gradient(700px 420px at 10% -10%, rgba(104, 160, 236, 0.28), transparent 70%),
    radial-gradient(700px 420px at 95% 110%, rgba(254, 159, 193, 0.2), transparent 70%);
}
:root[dir="rtl"] body { padding: 0 72px 0 0; }
.text { flex: 0 0 560px; display: flex; flex-direction: column; gap: 26px; }
.brand { display: flex; align-items: center; gap: 18px; font-size: 34px; font-weight: 500; color: var(--text-primary); }
.brand img { width: 72px; height: 72px; border-radius: 18px; }
h1 { margin: 0; font-size: 58px; line-height: 1.12; font-weight: 600; letter-spacing: -0.01em; }
:root[dir="rtl"] h1 { line-height: 1.4; letter-spacing: 0; font-size: 54px; }
.aurora { background: var(--gradient-aurora); -webkit-background-clip: text; background-clip: text; color: transparent; }
.chip { align-self: flex-start; padding: 10px 20px; border-radius: 999px; font-size: 22px; color: var(--text-secondary);
  background: var(--color-surface-raised); box-shadow: 0 0 0 1px var(--border-default); }
.shot { flex: 1; height: 470px; border-radius: 22px; overflow: hidden; border: 2px solid transparent;
  background: linear-gradient(var(--color-frame), var(--color-frame)) padding-box, var(--gradient-edge) border-box;
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.5); }
:root[dir="ltr"] .shot { border-right: 0; border-radius: 22px 0 0 22px; }
:root[dir="rtl"] .shot { border-left: 0; border-radius: 0 22px 22px 0; }
.shot img { display: block; height: 100%; width: auto; }
:root[dir="rtl"] .shot img { margin-inline-start: auto; }
</style>
</head>
<body>
<div class="text">
<div class="brand"><img src="${logo}" alt=""><span>${esc(dict.site.name)}</span></div>
<h1 class="aurora">${esc(dict.home.hero_title)}</h1>
<span class="chip">${esc(dict.home.eyebrow)}</span>
</div>
<div class="shot"><img src="${shot}" alt=""></div>
</body>
</html>`;
}

(async () => {
  if (!existsSync(join(DIST, "site.css"))) throw new Error("build the site first: bun site/build.mjs");
  mkdirSync(OUT, { recursive: true });
  const browser = await chromium.launch();
  const tab = await browser.newPage({ viewport: { width: 1200, height: 630 } });
  const tmp = join(SITE, "dist", "og-image.html");
  for (const lang of LANGS) {
    writeFileSync(tmp, page(lang));
    await tab.goto(pathToFileURL(tmp).href, { waitUntil: "networkidle" });
    await tab.evaluate(() => document.fonts.ready);
    await tab.screenshot({ path: join(OUT, `${lang}.png`) });
    console.log(`og image: ${join(OUT, `${lang}.png`)}`);
  }
  rmSync(tmp, { force: true });
  await browser.close();
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
