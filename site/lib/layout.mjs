import { icon } from "./icons.mjs";
import { esc } from "./util.mjs";

export const PAGES = [
  { id: "home", path: "" },
  { id: "download", path: "download/" },
  { id: "guide", path: "guide/" },
  { id: "tutorial", path: "tutorial/" },
  { id: "faq", path: "faq/" },
  { id: "privacy", path: "privacy/" },
  { id: "changelog", path: "whats-new/" },
];

const NAV = ["guide", "tutorial", "faq", "changelog", "privacy"];

const OG_LOCALE = { en: "en_GB", ar: "ar_EG", fr: "fr_FR", tr: "tr_TR", ur: "ur_PK" };

export function pageUrl(base, lang, id) {
  const page = PAGES.find((p) => p.id === id) ?? PAGES[0];
  return `${base}${lang}/${page.path}`;
}

export const THEME_BOOT = `(function(){var d=document.documentElement;d.className=d.className.replace("no-js","js");try{var t=localStorage.getItem("im.site.theme");if(t!=="light"&&t!=="dark")t=window.matchMedia&&matchMedia("(prefers-color-scheme: light)").matches?"light":"dark";d.setAttribute("data-theme",t);}catch(e){}})();`;

export function head({ ctx, title, description, canonical, alternates, image, extra = "" }) {
  const { site } = ctx;
  const alt = alternates
    .map((a) => `<link rel="alternate" hreflang="${a.lang}" href="${esc(site.origin + a.href)}">`)
    .join("\n");
  const ogAlt = alternates
    .filter((a) => a.lang !== ctx.lang && OG_LOCALE[a.lang])
    .map((a) => `<meta property="og:locale:alternate" content="${OG_LOCALE[a.lang]}">`)
    .join("\n");
  return `<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)}</title>
<meta name="description" content="${esc(description)}">
<link rel="canonical" href="${esc(site.origin + canonical)}">
${alt}
<meta property="og:type" content="website">
<meta property="og:site_name" content="${esc(ctx.t("site.name"))}">
<meta property="og:title" content="${esc(title)}">
<meta property="og:description" content="${esc(description)}">
<meta property="og:url" content="${esc(site.origin + canonical)}">
<meta property="og:image" content="${esc(site.origin + image)}">
<meta property="og:image:width" content="1920">
<meta property="og:image:height" content="1080">
<meta property="og:locale" content="${OG_LOCALE[ctx.lang] ?? "en_GB"}">
${ogAlt}
<meta name="twitter:card" content="summary_large_image">
<meta name="color-scheme" content="dark light">
<meta name="theme-color" content="#0c1015">
<link rel="icon" type="image/png" href="${site.base}assets/favicon.png">
<link rel="apple-touch-icon" href="${site.base}assets/favicon.png">
<link rel="stylesheet" href="${site.base}assets/site.css?v=${site.hash.css}">
<script>${THEME_BOOT}</script>
<script src="${site.base}assets/site.js?v=${site.hash.js}" defer></script>
${extra}`;
}

function langMenu(ctx, pageId) {
  const { site, lang } = ctx;
  const current = site.meta[lang];
  const items = site.langs
    .map((l) => {
      const href = pageId ? pageUrl(site.base, l, pageId) : `${site.base}${l}/`;
      const cur = l === lang ? ' aria-current="true"' : "";
      return `<li><a href="${href}" hreflang="${l}" lang="${l}" dir="${site.meta[l].dir}" data-lang="${l}"${cur}>${esc(site.meta[l].name)}</a></li>`;
    })
    .join("");
  return `<details class="lang-menu">
<summary aria-label="${esc(ctx.t("site.language"))}: ${esc(current.name)}">${icon("globe", { size: 18 })}<span class="lang-current">${esc(current.name)}</span>${icon("chevron", { size: 16 })}</summary>
<ul class="lang-list">${items}</ul>
</details>`;
}

function header(ctx, pageId) {
  const { site, lang } = ctx;
  const links = NAV.map((id) => {
    const cur = id === pageId ? ' aria-current="page"' : "";
    return `<li><a href="${pageUrl(site.base, lang, id)}"${cur}>${esc(ctx.t(`nav.${id}`))}</a></li>`;
  }).join("");
  const dlCur = pageId === "download" ? ' aria-current="page"' : "";
  return `<header class="site-header">
<div class="wrap header-row">
<a class="brand" href="${pageUrl(site.base, lang, "home")}"${pageId === "home" ? ' aria-current="page"' : ""}><img src="${site.base}assets/favicon.png" alt="" width="32" height="32"><span>${esc(ctx.t("site.name"))}</span></a>
<nav class="main-nav" id="site-nav" aria-label="${esc(ctx.t("site.main_nav"))}"><ul>${links}</ul></nav>
<div class="header-tools">
${langMenu(ctx, pageId)}
<button type="button" class="btn btn-icon btn-ghost theme-toggle" aria-label="${esc(ctx.t("site.theme"))}" title="${esc(ctx.t("site.theme"))}">${icon("sun", { size: 18, cls: "i-sun" })}${icon("moon", { size: 18, cls: "i-moon" })}</button>
<a class="btn btn-aurora btn-sm header-cta" href="${pageUrl(site.base, lang, "download")}"${dlCur}>${icon("download", { size: 16 })}<span>${esc(ctx.t("nav.download"))}</span></a>
<button type="button" class="btn btn-icon btn-ghost nav-toggle" aria-expanded="false" aria-controls="site-nav" aria-label="${esc(ctx.t("site.menu"))}">${icon("menu", { size: 20 })}</button>
</div>
</div>
</header>`;
}

function footer(ctx) {
  const { site, lang } = ctx;
  const links = ["download", "guide", "tutorial", "faq", "changelog", "privacy"]
    .map((id) => `<li><a href="${pageUrl(site.base, lang, id)}">${esc(ctx.t(`nav.${id}`))}</a></li>`)
    .join("");
  const langs = site.langs
    .map((l) => `<li><a href="${site.base}${l}/" hreflang="${l}" lang="${l}" data-lang="${l}">${esc(site.meta[l].name)}</a></li>`)
    .join("");
  return `<footer class="site-footer">
<div class="wrap footer-grid">
<div class="footer-brand">
<a class="brand" href="${pageUrl(site.base, lang, "home")}"><img src="${site.base}assets/favicon.png" alt="" width="28" height="28" loading="lazy"><span>${esc(ctx.t("site.name"))}</span></a>
<p>${esc(ctx.t("footer.tagline"))}</p>
</div>
<nav aria-label="${esc(ctx.t("site.menu"))}"><ul class="footer-links">${links}</ul></nav>
<div>
<ul class="footer-links">
<li><a href="${site.repoUrl}" rel="noopener">${icon("github", { size: 16 })} ${esc(ctx.t("footer.source"))}</a></li>
<li><a href="${site.repoUrl}/issues" rel="noopener">${esc(ctx.t("footer.issues"))}</a></li>
<li><a href="${pageUrl(site.base, lang, "privacy")}">${esc(ctx.t("nav.privacy"))}</a></li>
</ul>
</div>
<nav aria-label="${esc(ctx.t("footer.languages"))}"><ul class="footer-links">${langs}</ul></nav>
</div>
<div class="wrap footer-bottom"><p><a href="${site.repoUrl}/blob/master/LICENSE" rel="noopener">${esc(ctx.t("footer.license"))}</a></p><p>© ${site.year} <a href="https://github.com/Khaymah" rel="noopener">Khaymah</a></p></div>
</footer>`;
}

export function shell(ctx, { pageId, title, description, body, image, notFound = false }) {
  const { site, lang } = ctx;
  const m = site.meta[lang];
  const alternates = notFound
    ? []
    : [
        ...site.langs.map((l) => ({ lang: l, href: pageUrl(site.base, l, pageId) })),
        { lang: "x-default", href: pageId === "home" ? site.base : pageUrl(site.base, "en", pageId) },
      ];
  const canonical = notFound ? site.base : pageUrl(site.base, lang, pageId);
  const extra = notFound ? '<meta name="robots" content="noindex">' : "";
  return `<!doctype html>
<html lang="${lang}" dir="${m.dir}" class="no-js" data-theme="dark">
<head>
${head({ ctx, title, description, canonical, alternates, image, extra })}
</head>
<body data-page="${pageId}">
<a class="skip-link" href="#main">${esc(ctx.t("site.skip"))}</a>
${header(ctx, pageId)}
<main id="main" tabindex="-1">
${body}
</main>
${footer(ctx)}
</body>
</html>
`;
}
