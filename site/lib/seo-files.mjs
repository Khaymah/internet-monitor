import { PAGES, pageUrl, THEME_BOOT } from "./layout.mjs";
import { OS_LIST, ROBOTS } from "./seo.mjs";
import { esc, verificationMeta } from "./util.mjs";

const CRAWLERS = [
  "Googlebot",
  "Bingbot",
  "GPTBot",
  "OAI-SearchBot",
  "ChatGPT-User",
  "ClaudeBot",
  "Claude-Web",
  "PerplexityBot",
  "Google-Extended",
  "Applebot-Extended",
];

export function rootPage(site, dicts) {
  const en = dicts.en;
  const home = `${site.origin}${site.base}`;
  const links = site.langs
    .map(
      (l) =>
        `<li><a class="card" href="${site.base}${l}/" hreflang="${l}" lang="${l}" dir="${site.meta[l].dir}" data-lang="${l}">${esc(site.meta[l].name)}<span>${esc(dicts[l].root.pick)}</span></a></li>`,
    )
    .join("");
  const alternates = site.langs
    .map((l) => `<link rel="alternate" hreflang="${l}" href="${home}${l}/">`)
    .join("\n");
  const redirect = `(function(){var langs="${site.langs.join(",")}".split(",");var pick=null;try{var s=localStorage.getItem("im.site.lang");if(langs.indexOf(s)>=0)pick=s;}catch(e){}if(!pick){var list=navigator.languages&&navigator.languages.length?navigator.languages:[navigator.language||"en"];for(var i=0;i<list.length&&!pick;i++){var c=String(list[i]).toLowerCase().split("-")[0];if(langs.indexOf(c)>=0)pick=c;}}location.replace("${site.base}"+(pick||"en")+"/"+location.search+location.hash);})();`;
  const ld = JSON.stringify({
    "@context": "https://schema.org",
    "@graph": [
      { "@type": "Organization", "@id": `${home}#org`, name: "Khaymah", url: "https://github.com/Khaymah", logo: `${home}assets/favicon.png` },
      {
        "@type": "WebSite",
        "@id": `${home}#website`,
        url: home,
        name: "Internet Monitor",
        inLanguage: site.langs,
        publisher: { "@id": `${home}#org` },
      },
    ],
  }).replace(/</g, "\\u003c");
  const image = `${home}assets/og/en.png`;
  return `<!doctype html>
<html lang="en" dir="ltr" class="no-js" data-theme="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(en.root.title)}</title>
<meta name="description" content="${esc(en.home.description)}">
<meta name="keywords" content="${esc([...en.seo.page_keywords.home, ...en.seo.keywords].join(", "))}">
<meta name="robots" content="${ROBOTS}">
<meta name="author" content="Khaymah">
<meta name="application-name" content="Internet Monitor">
<link rel="canonical" href="${home}">
${alternates}
<link rel="alternate" hreflang="x-default" href="${home}">
${verificationMeta()}
<meta property="og:type" content="website">
<meta property="og:site_name" content="Internet Monitor">
<meta property="og:title" content="${esc(en.home.title)}">
<meta property="og:description" content="${esc(en.home.description)}">
<meta property="og:url" content="${home}">
<meta property="og:locale" content="en_GB">
<meta property="og:image" content="${image}">
<meta property="og:image:type" content="image/png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="${esc(en.seo.og_alt)}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="${esc(en.home.title)}">
<meta name="twitter:description" content="${esc(en.home.description)}">
<meta name="twitter:image" content="${image}">
<meta name="twitter:image:alt" content="${esc(en.seo.og_alt)}">
<meta name="color-scheme" content="dark light">
<meta name="theme-color" content="#0c1015">
<link rel="icon" type="image/png" href="${site.base}assets/favicon.png">
<link rel="stylesheet" href="${site.base}assets/site.css?v=${site.hash.css}">
<script>${THEME_BOOT}${redirect}</script>
<script type="application/ld+json">${ld}</script>
</head>
<body class="root-page">
<main class="wrap root-main">
<img src="${site.base}assets/favicon.png" alt="${esc(en.site.logo_alt)}" width="72" height="72">
<h1 class="aurora-text">${esc(en.site.name)}</h1>
<p class="lead">${esc(en.home.hero_sub)}</p>
<h2 class="sr-only">${esc(en.root.pick)}</h2>
<ul class="root-langs">${links}</ul>
</main>
</body>
</html>
`;
}

export function sitemap(site, dates) {
  const urls = PAGES.map((p) =>
    site.langs
      .map((l) => {
        const alts = [
          ...site.langs.map((a) => `<xhtml:link rel="alternate" hreflang="${a}" href="${site.origin}${pageUrl(site.base, a, p.id)}"/>`),
          `<xhtml:link rel="alternate" hreflang="x-default" href="${site.origin}${p.id === "home" ? site.base : pageUrl(site.base, "en", p.id)}"/>`,
        ].join("");
        return `<url><loc>${site.origin}${pageUrl(site.base, l, p.id)}</loc><lastmod>${dates[p.id]}</lastmod>${alts}</url>`;
      })
      .join("\n"),
  ).join("\n");
  const today = new Date().toISOString().slice(0, 10);
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
<url><loc>${site.origin}${site.base}</loc><lastmod>${dates.home ?? today}</lastmod></url>
${urls}
</urlset>
`;
}

export function robots(site) {
  const named = CRAWLERS.map((bot) => `User-agent: ${bot}\nAllow: /`).join("\n\n");
  return `${named}\n\nUser-agent: *\nAllow: /\n\nSitemap: ${site.origin}${site.base}sitemap.xml\n`;
}

function plain(text) {
  return String(text)
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/\*\*([^*]+)\*\*/g, "$1");
}

function link(ctx, id) {
  return `${ctx.site.origin}${ctx.url(id)}`;
}

export function llmsTxt(ctx) {
  const pages = PAGES.map((p) => `- [${ctx.navLabel(p.id)}](${link(ctx, p.id)}): ${ctx.t(`${p.id}.description`)}`).join("\n");
  const langs = ctx.site.langs
    .filter((l) => l !== "en")
    .map((l) => `- [${ctx.site.meta[l].name}](${ctx.site.origin}${ctx.site.base}${l}/)`)
    .join("\n");
  return `# Internet Monitor

> ${plain(ctx.t("home.description"))}

${plain(ctx.t("home.hero_sub"))} It runs on ${OS_LIST}, needs no account, keeps its data on the user's computer, and is free for personal and commercial use under the Elastic License 2.0. Made by Khaymah (developer: Sabry Dawood).

## Pages

${pages}

## Other languages

${langs}

## More

- [Full text for language models](${ctx.site.origin}${ctx.site.base}llms-full.txt)
- [Source code and releases](${ctx.site.repoUrl})
- [Sitemap](${ctx.site.origin}${ctx.site.base}sitemap.xml)
`;
}

export function llmsFull(ctx) {
  const a = ctx.appx;
  const lines = [`# Internet Monitor`, "", `> ${plain(ctx.t("home.description"))}`, "", `Website: ${link(ctx, "home")}`, `Source code: ${ctx.site.repoUrl}`];
  const stable = ctx.releases.stable;
  if (stable) lines.push(`Latest version: ${stable.tag} (${stable.date.slice(0, 10)})`);
  lines.push("", "## What it does", "", plain(ctx.t("home.hero_sub")), "");
  for (const b of ctx.raw("home.benefits")) lines.push(`- **${b.title}**: ${plain(b.body)}`);
  lines.push("", `## ${ctx.t("home.promise_title")}`, "", plain(ctx.t("home.promise_sub")), "");
  for (const p of ctx.raw("home.promise")) lines.push(`- ${plain(p)}`);
  lines.push("", "## Download and install", "", `${ctx.raw("seo.answer.download")} Download page: ${link(ctx, "download")}`, "");
  for (const os of ["windows", "mac", "linux", "pi"]) {
    lines.push(`### ${ctx.t(`download.os.${os}`)} (${ctx.t(`download.os_sub.${os}`)})`, "");
    ctx.raw(`download.steps.${os}`).forEach((s, i) => lines.push(`${i + 1}. ${plain(s)}`));
    lines.push("");
  }
  lines.push(`## ${ctx.t("guide.h1")}`, "", plain(ctx.t("guide.intro")), "");
  ctx.raw("guide.steps").forEach((s, i) => lines.push(`${i + 1}. **${s.title}**: ${plain(s.body)}`));
  lines.push("", `### ${ctx.t("guide.tip_title")}`, "");
  for (const tip of ctx.raw("guide.tips")) lines.push(`- ${plain(tip)}`);
  lines.push("", `## ${a.t("faq_title")}`, "");
  for (const section of a.data.faq) {
    lines.push(`### ${a.t(section.title)}`, "");
    for (const item of section.items) lines.push(`**${a.t(item.q)}**`, "", plain(a.t(item.a)), "");
  }
  const privacy = a.data.legal.PRIVACY;
  lines.push(`## ${a.t("privacy_title")}`, "", `${a.t(privacy.intro)} (${a.t("legal_updated", { date: privacy.updated })})`, "");
  for (const s of privacy.sections) {
    lines.push(`### ${a.t(s.title)}`, "");
    for (const p of s.body) lines.push(a.t(p), "");
    for (const li of s.list ?? []) lines.push(`- ${a.t(li)}`);
    if (s.list) lines.push("");
  }
  lines.push(`Full policy: ${link(ctx, "privacy")}`, "", "## Contact", "");
  for (const c of a.data.contact.CHANNELS) lines.push(`- ${a.t(c.name)}: ${c.href.replace(/^mailto:/, "")} (${a.t(c.best)})`);
  lines.push("");
  return lines.join("\n");
}
