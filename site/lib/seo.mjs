import { readFileSync } from "node:fs";
import { join } from "node:path";
import { guideImageUrl } from "./guide-art.mjs";
import { esc } from "./util.mjs";

const SOURCES = {
  home: ["site/locales", "site/lib/page-home.mjs", "docs/store"],
  download: ["site/locales", "site/lib/page-download.mjs"],
  guide: ["site/locales", "site/lib/page-docs.mjs", "site/lib/guide-art.mjs", "site/assets/guide"],
  tutorial: ["site/locales", "web/public/tutorial"],
  faq: ["web/src/content/faq.ts", "web/locales/*/faq.json"],
  changelog: ["web/src/content/changelog.ts", "web/locales/*/changelog.json", "docs/releases"],
  about: ["web/src/content/about.ts", "web/locales/*/about.json"],
  contact: ["web/src/content/contact.ts", "web/locales/*/about.json"],
  support: ["web/src/content/donate.ts", "web/locales/*/support.json"],
};

const PAGE_TYPE = { home: "WebPage", faq: "FAQPage", about: "AboutPage", contact: "ContactPage" };

export const ROBOTS = "index,follow,max-image-preview:large,max-snippet:-1,max-video-preview:-1";
export const OS_LIST = "Windows 10, Windows 11, macOS, Linux, Raspberry Pi OS";

function gitDate(root, paths) {
  try {
    const out = Bun.spawnSync(["git", "log", "-1", "--format=%cs", "--", ...paths], { cwd: root });
    const date = out.stdout.toString().trim();
    return /^\d{4}-\d{2}-\d{2}$/.test(date) ? date : null;
  } catch {
    return null;
  }
}

function latest(...dates) {
  return dates
    .filter(Boolean)
    .map((d) => String(d).slice(0, 10))
    .sort()
    .pop();
}

export function pageDates(root, releases, appData) {
  const today = new Date().toISOString().slice(0, 10);
  const dates = {};
  for (const [id, paths] of Object.entries(SOURCES)) dates[id] = gitDate(root, paths) ?? today;
  dates.download = latest(dates.download, releases.stable?.date, releases.preview?.date);
  dates.tutorial = latest(dates.tutorial, releases.tutorial?.release.date);
  dates.changelog = latest(dates.changelog, appData.releases.find((r) => r.date)?.date);
  dates.privacy = appData.legal.PRIVACY.updated;
  dates.terms = appData.legal.TERMS.updated;
  return dates;
}

export function vttDuration(file) {
  try {
    const stamps = [...readFileSync(file, "utf8").matchAll(/-->\s*(?:(\d+):)?(\d+):(\d+)\.\d+/g)];
    const last = stamps.pop();
    if (!last) return null;
    const total = Number(last[1] ?? 0) * 3600 + Number(last[2]) * 60 + Number(last[3]);
    return `PT${Math.floor(total / 60)}M${total % 60}S`;
  } catch {
    return null;
  }
}

export function keywords(ctx, pageId) {
  const page = ctx.raw(`seo.page_keywords.${pageId}`) ?? [];
  return [...page, ...ctx.raw("seo.keywords")].join(", ");
}

export function withIntro(ctx, pageId, body) {
  const at = body.indexOf("</header>");
  if (pageId === "home" || at < 0 || !body.includes('class="doc-hero"')) return body;
  const intro = pageIntro(ctx, pageId, { updated: !body.includes('class="legal-updated"') });
  return body.slice(0, at) + intro + "\n" + body.slice(at);
}

function pageIntro(ctx, pageId, { updated = true } = {}) {
  const answer = ctx.raw(`seo.answer.${pageId}`);
  const date = ctx.dates[pageId];
  const parts = [];
  if (answer) parts.push(`<p class="page-answer">${esc(answer)}</p>`);
  if (updated && date) {
    parts.push(
      `<p class="legal-updated"><time datetime="${date}">${esc(ctx.appx.t("legal_updated", { date: ctx.appx.fmt.date(date) }))}</time></p>`,
    );
  }
  return parts.length ? `\n${parts.join("\n")}` : "";
}

function plain(text) {
  return String(text)
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/`([^`]+)`/g, "$1");
}

function abs(ctx, path) {
  return ctx.site.origin + path;
}

function software(ctx, url) {
  const { site, releases, lang } = ctx;
  const base = site.origin + site.base;
  const stable = releases.stable;
  const app = {
    "@type": "SoftwareApplication",
    "@id": `${base}#software`,
    name: "Internet Monitor",
    alternateName: ctx.t("site.name"),
    description: ctx.t("home.description"),
    url: abs(ctx, ctx.url("home")),
    applicationCategory: "UtilitiesApplication",
    operatingSystem: OS_LIST,
    inLanguage: lang,
    offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
    downloadUrl: abs(ctx, ctx.url("download")),
    installUrl: abs(ctx, ctx.url("download")),
    screenshot: ctx.shots.files.map((f) => `${base}assets/shots/${ctx.shots.lang}/${f}`),
    image: `${base}assets/og/${lang}.png`,
    license: `${site.repoUrl}/blob/master/LICENSE`,
    featureList: ctx.raw("home.benefits").map((b) => b.title),
    publisher: { "@id": `${base}#org` },
    author: { "@id": `${base}#developer` },
    isAccessibleForFree: true,
    mainEntityOfPage: url,
  };
  if (stable) {
    app.softwareVersion = stable.tag.replace(/^v/, "");
    app.datePublished = stable.date.slice(0, 10);
  }
  return app;
}

function howTo(name, description, steps, url, lang) {
  const image = steps.find((s) => s.image)?.image;
  return {
    "@type": "HowTo",
    name,
    description,
    inLanguage: lang,
    url,
    ...(image ? { image } : {}),
    step: steps.map((s, i) => ({
      "@type": "HowToStep",
      position: i + 1,
      name: s.name,
      text: s.text,
      ...(s.image ? { image: s.image } : {}),
    })),
  };
}

export function structuredData(ctx, pageId, title, description) {
  const { site, lang } = ctx;
  const base = site.origin + site.base;
  const url = abs(ctx, ctx.url(pageId));
  const website = `${base}${lang}/#website`;
  const graph = [
    {
      "@type": "Organization",
      "@id": `${base}#org`,
      name: "Khaymah",
      url: "https://github.com/Khaymah",
      logo: `${base}assets/favicon.png`,
    },
    { "@type": "Person", "@id": `${base}#developer`, name: "Sabry Dawood", url: "https://github.com/sabrydawood" },
    {
      "@type": "WebSite",
      "@id": website,
      url: abs(ctx, ctx.url("home")),
      name: ctx.t("site.name"),
      alternateName: "Internet Monitor",
      inLanguage: lang,
      publisher: { "@id": `${base}#org` },
    },
  ];
  const page = {
    "@type": PAGE_TYPE[pageId] ?? "WebPage",
    "@id": `${url}#webpage`,
    url,
    name: title,
    description,
    inLanguage: lang,
    isPartOf: { "@id": website },
    primaryImageOfPage: `${base}assets/og/${lang}.png`,
    dateModified: ctx.dates[pageId],
  };
  if (pageId !== "home") {
    page.breadcrumb = {
      "@type": "BreadcrumbList",
      itemListElement: [
        { "@type": "ListItem", position: 1, name: ctx.t("nav.home"), item: abs(ctx, ctx.url("home")) },
        { "@type": "ListItem", position: 2, name: ctx.navLabel(pageId), item: url },
      ],
    };
  }
  if (pageId === "faq") {
    page.mainEntity = ctx.appx.data.faq.flatMap((s) =>
      s.items.map((item) => ({
        "@type": "Question",
        name: ctx.appx.t(item.q),
        acceptedAnswer: { "@type": "Answer", text: plain(ctx.appx.t(item.a)) },
      })),
    );
  }
  if (pageId === "about" || pageId === "contact") page.about = { "@id": `${base}#software` };
  graph.push(page);
  if (pageId === "home" || pageId === "download") graph.push(software(ctx, url));
  if (pageId === "guide") {
    const steps = ctx
      .raw("guide.steps")
      .map((s, i) => ({ name: s.title, text: plain(s.body), image: guideImageUrl(ctx, i) }));
    graph.push(howTo(ctx.t("guide.h1"), ctx.t("guide.intro"), steps, url, lang));
  }
  if (pageId === "download") {
    for (const os of ["windows", "mac", "linux", "pi"]) {
      const steps = ctx.raw(`download.steps.${os}`).map((s) => ({ name: plain(s).split(/[.!?،。۔]/)[0], text: plain(s) }));
      const name = `${ctx.t("download.install_title")}: ${ctx.t(`download.os.${os}`)}`;
      graph.push(howTo(name, ctx.t("download.install_sub"), steps, `${url}#os-${os}`, lang));
    }
  }
  if (pageId === "tutorial" && ctx.releases.tutorial) {
    const tut = ctx.releases.tutorial;
    const video = {
      "@type": "VideoObject",
      name: ctx.t("tutorial.h1"),
      description: ctx.t("tutorial.intro"),
      thumbnailUrl: ctx.shots.files[0] ? `${base}assets/shots/${ctx.shots.lang}/${ctx.shots.files[0]}` : `${base}assets/og/${lang}.png`,
      contentUrl: tut.url,
      uploadDate: tut.release.date,
      inLanguage: "en",
      caption: ctx.tracks.map((tr) => ({
        "@type": "MediaObject",
        contentUrl: `${base}tutorial/${tr.lang}.vtt`,
        encodingFormat: "text/vtt",
        inLanguage: tr.lang,
      })),
    };
    if (ctx.videoDuration) video.duration = ctx.videoDuration;
    graph.push(video);
  }
  const json = JSON.stringify({ "@context": "https://schema.org", "@graph": graph }).replace(/</g, "\\u003c");
  return `<script type="application/ld+json">${json}</script>`;
}
