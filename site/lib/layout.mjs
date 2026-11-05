import { icon } from "./icons.mjs";
import { esc, verificationMeta } from "./util.mjs";

export const PAGES = [
  { id: "home", path: "" },
  { id: "download", path: "download/" },
  { id: "guide", path: "guide/" },
  { id: "tutorial", path: "tutorial/" },
  { id: "faq", path: "faq/" },
  { id: "privacy", path: "privacy/" },
  { id: "changelog", path: "whats-new/" },
  { id: "about", path: "about/" },
  { id: "contact", path: "contact/" },
  { id: "support", path: "support/" },
  { id: "terms", path: "terms/" },
];

const NAV = ["guide", "tutorial", "faq", "changelog"];

const MORE = [
  { id: "about", icon: "Info" },
  { id: "contact", icon: "MessagesSquare" },
  { id: "support", icon: "HeartHandshake" },
  { id: "privacy", icon: "ShieldCheck" },
  { id: "terms", icon: "Scale" },
];

const APP_GROUP = [
  { id: "about", icon: "Info" },
  { id: "changelog", icon: "Sparkles" },
  { id: "faq", icon: "CircleHelp" },
  { id: "tutorial", icon: "CirclePlay" },
  { id: "contact", icon: "MessagesSquare" },
  { id: "privacy", icon: "ShieldCheck" },
  { id: "terms", icon: "Scale" },
  { id: "support", icon: "HeartHandshake" },
];

const APP_LABEL = { about: "nav_about", contact: "nav_contact", support: "nav_support", terms: "nav_terms" };

export function navLabel(ctx, id) {
  return APP_LABEL[id] ? ctx.appx.t(APP_LABEL[id]) : ctx.t(`nav.${id}`);
}

function groupLinks(ctx, items, pageId) {
  const { site, lang } = ctx;
  return items
    .map(({ id, icon: name }) => {
      const cur = id === pageId ? ' aria-current="page"' : "";
      return `<li><a href="${pageUrl(site.base, lang, id)}"${cur}>${ctx.appx.svg(ctx.appx.lucide[name], 15)}<span>${esc(navLabel(ctx, id))}</span></a></li>`;
    })
    .join("");
}

const OG_LOCALE = { en: "en_GB", ar: "ar_EG", fr: "fr_FR", tr: "tr_TR", ur: "ur_PK" };

export function pageUrl(base, lang, id) {
  const page = PAGES.find((p) => p.id === id) ?? PAGES[0];
  return `${base}${lang}/${page.path}`;
}

export const THEME_BOOT = `(function(){var d=document.documentElement;d.className=d.className.replace("no-js","js");try{var t=localStorage.getItem("im.site.theme");if(t!=="light"&&t!=="dark")t=window.matchMedia&&matchMedia("(prefers-color-scheme: light)").matches?"light":"dark";d.setAttribute("data-theme",t);}catch(e){}})();`;

export function head({ ctx, title, description, canonical, alternates, image, extra = "", scripts = [] }) {
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
${verificationMeta()}
<meta name="color-scheme" content="dark light">
<meta name="theme-color" content="#0c1015">
<link rel="icon" type="image/png" href="${site.base}assets/favicon.png">
<link rel="apple-touch-icon" href="${site.base}assets/favicon.png">
<link rel="stylesheet" href="${site.base}assets/site.css?v=${site.hash.css}">
<script>${THEME_BOOT}</script>
<script src="${site.base}assets/site.js?v=${site.hash.js}" defer></script>
${scripts.map((f) => `<script src="${site.base}assets/${f}?v=${site.hash[f]}" defer></script>`).join("\n")}
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
    return `<li><a href="${pageUrl(site.base, lang, id)}"${cur}>${esc(navLabel(ctx, id))}</a></li>`;
  }).join("");
  const moreLabel = esc(ctx.appx.t("nav_more"));
  const moreCur = MORE.some((m) => m.id === pageId) ? " is-current" : "";
  const more = `<details class="more-menu${moreCur}">
<summary>${moreLabel}${icon("chevron", { size: 16 })}</summary>
<ul class="more-list">${groupLinks(ctx, MORE, pageId)}</ul>
</details>`;
  const dlCur = pageId === "download" ? ' aria-current="page"' : "";
  return `<header class="site-header">
<div class="wrap header-row">
<a class="brand" href="${pageUrl(site.base, lang, "home")}"${pageId === "home" ? ' aria-current="page"' : ""}><img src="${site.base}assets/favicon.png" alt="" width="32" height="32"><span>${esc(ctx.t("site.name"))}</span></a>
<nav class="main-nav" id="site-nav" aria-label="${esc(ctx.t("site.main_nav"))}"><ul>${links}</ul>${more}<ul class="nav-more" aria-label="${moreLabel}">${groupLinks(ctx, MORE, pageId)}</ul></nav>
<div class="header-tools">
${langMenu(ctx, pageId)}
<button type="button" class="btn btn-edge btn-sm btn-icon theme-toggle" aria-label="${esc(ctx.t("site.theme"))}" title="${esc(ctx.t("site.theme"))}">${ctx.appx.svg(ctx.appx.lucide.Sun, 15, "i-sun")}${ctx.appx.svg(ctx.appx.lucide.Moon, 15, "i-moon")}</button>
<a class="btn btn-aurora btn-sm header-cta" href="${pageUrl(site.base, lang, "download")}"${dlCur}>${icon("download", { size: 16 })}<span>${esc(ctx.t("nav.download"))}</span></a>
<button type="button" class="btn btn-icon btn-ghost nav-toggle" aria-expanded="false" aria-controls="site-nav" aria-label="${esc(ctx.t("site.menu"))}">${icon("menu", { size: 20 })}</button>
</div>
</div>
</header>`;
}

function footer(ctx, pageId) {
  const { site, lang } = ctx;
  const links = ["home", "download", "guide"]
    .map((id) => `<li><a href="${pageUrl(site.base, lang, id)}"${id === pageId ? ' aria-current="page"' : ""}>${esc(navLabel(ctx, id))}</a></li>`)
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
<nav aria-label="${esc(ctx.t("site.main_nav"))}"><ul class="footer-links">${links}
<li><a href="${site.repoUrl}" rel="noopener">${icon("github", { size: 16 })} ${esc(ctx.t("footer.source"))}</a></li>
<li><a href="${site.repoUrl}/issues" rel="noopener">${esc(ctx.t("footer.issues"))}</a></li>
</ul></nav>
<nav aria-label="${esc(ctx.appx.t("nav_more"))}"><ul class="footer-links footer-app">${groupLinks(ctx, APP_GROUP, pageId)}</ul></nav>
<nav aria-label="${esc(ctx.t("footer.languages"))}"><ul class="footer-links">${langs}</ul></nav>
</div>
<div class="wrap footer-bottom"><p><a href="${site.repoUrl}/blob/master/LICENSE" rel="noopener">${esc(ctx.t("footer.license"))}</a></p><p>© ${site.year} <a href="https://github.com/Khaymah" rel="noopener">Khaymah</a></p></div>
</footer>`;
}

export function shell(ctx, { pageId, title, description, body, image, notFound = false, scripts = [] }) {
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
${head({ ctx, title, description, canonical, alternates, image, extra, scripts })}
</head>
<body data-page="${pageId}">
<a class="skip-link" href="#main">${esc(ctx.t("site.skip"))}</a>
${header(ctx, pageId)}
<main id="main" tabindex="-1">
${body}
</main>
${footer(ctx, pageId)}
</body>
</html>
`;
}
