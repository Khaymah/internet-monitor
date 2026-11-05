import { icon } from "./icons.mjs";
import { PAGES, pageUrl } from "./layout.mjs";
import { markdown } from "./markdown.mjs";
import { esc, fill, inline } from "./util.mjs";

function hero(title, lead, extra = "", narrow = true) {
  return `<section class="page-hero">
<div class="wrap${narrow ? " narrow" : ""}">
<h1>${title}</h1>
${lead ? `<p class="lead">${lead}</p>` : ""}
${extra}
</div>
</section>`;
}

export function renderTutorial(ctx) {
  const { releases, site, lang } = ctx;
  const tut = releases.tutorial;
  let main;
  if (tut) {
    const tracks = ctx.tracks
      .map(
        (tr) =>
          `<track kind="subtitles" src="${site.base}tutorial/${tr.lang}.vtt" srclang="${tr.lang}" label="${esc(site.meta[tr.lang].name)}"${tr.lang === lang ? " default" : ""}>`,
      )
      .join("\n");
    const poster = ctx.shots.files[0] ? ` poster="${site.base}assets/shots/${ctx.shots.lang}/${ctx.shots.files[0]}"` : "";
    const note = tut.release.prerelease ? `<p class="hint">${ctx.e("tutorial.preview_note", { version: tut.release.tag })}</p>` : "";
    main = `<div class="video-frame frame frame-glow">
<video controls preload="none" playsinline${poster} aria-label="${ctx.e("tutorial.video_label")}">
<source src="${esc(tut.url)}" type="video/mp4">
${tracks}
</video>
</div>
<p class="hint">${
      ctx.tracks.length
        ? ctx.e("tutorial.subtitles", {
            langs: new Intl.ListFormat(site.meta[lang].intl, { type: "conjunction" }).format(
              ctx.tracks.map((tr) => site.meta[tr.lang].name),
            ),
          })
        : ctx.e("tutorial.no_subtitles")
    }</p>
${note}
<p><a class="text-link" href="${esc(tut.url)}" rel="noopener">${icon("download", { size: 16 })}${ctx.e("tutorial.download_video")}</a></p>`;
  } else {
    main = `<div class="card edge soon">
<span class="well well-lg">${icon("play", { size: 26 })}</span>
<h2>${ctx.e("tutorial.soon_title")}</h2>
<p>${ctx.e("tutorial.soon_body")}</p>
</div>`;
  }
  const body = `${hero(ctx.e("tutorial.h1"), ctx.e("tutorial.intro"))}
<div class="wrap page-body narrow">
${main}
<p class="center"><a class="btn btn-aurora btn-lg" href="${pageUrl(site.base, lang, "guide")}">${icon("book", { size: 18 })}${ctx.e("tutorial.next")}</a></p>
</div>`;
  return { title: ctx.t("tutorial.title"), description: ctx.t("tutorial.description"), body };
}

export function renderGuide(ctx) {
  const { site, lang } = ctx;
  const steps = ctx
    .raw("guide.steps")
    .map(
      (s, i) => `<li class="guide-step">
<span class="step-num" aria-hidden="true">${i + 1}</span>
<div class="card">
<h2>${esc(s.title)}</h2>
<p>${ctx.inline(s.body)}</p>
</div>
</li>`,
    )
    .join("\n");
  const tips = ctx
    .raw("guide.tips")
    .map((tip) => `<li>${ctx.inline(tip)}</li>`)
    .join("");
  const piSteps = ctx
    .raw("guide.pi_steps")
    .map((s) => `<li>${ctx.inline(s)}</li>`)
    .join("");
  const body = `${hero(ctx.e("guide.h1"), ctx.e("guide.intro"))}
<div class="wrap page-body narrow">
<ol class="guide-steps">
${steps}
</ol>
<aside class="card tips" aria-labelledby="tips-h">
<h2 id="tips-h">${icon("lightbulb", { size: 20 })}${ctx.e("guide.tip_title")}</h2>
<ul>${tips}</ul>
</aside>
<p class="center"><a class="btn btn-aurora btn-lg" href="${pageUrl(site.base, lang, "download")}">${icon("download", { size: 18 })}${ctx.e("guide.cta")}</a></p>
<section class="card edge pi" id="raspberry-pi" aria-labelledby="pi-h">
<span class="well well-lg">${icon("pi", { size: 26 })}</span>
<h2 id="pi-h">${ctx.e("guide.pi_title")}</h2>
<p>${ctx.e("guide.pi_intro")}</p>
<ol class="steps">${piSteps}</ol>
<pre dir="ltr"><code>chmod +x internet-monitor-headless-linux-arm64
./internet-monitor-headless-linux-arm64 --dir ~/internet-monitor --listen 0.0.0.0</code></pre>
<p class="row-links">
<a class="btn btn-edge" href="${pageUrl(site.base, lang, "download")}#os-pi">${icon("download", { size: 18 })}${ctx.e("download.os.pi")}</a>
<a class="text-link" href="${site.repoUrl}/blob/master/docs/headless.md" rel="noopener">${ctx.e("guide.pi_more")}${icon("arrow", { size: 16, cls: "flip-rtl" })}</a>
</p>
</section>
</div>`;
  return { title: ctx.t("guide.title"), description: ctx.t("guide.description"), body };
}

export function renderFaq(ctx) {
  const faq = ctx.app.faq;
  const answer = (text) => inline(text, () => null);
  const sections = faq.sections
    .map(
      (s) => `<section class="faq-section" aria-labelledby="faq-${s.id}">
${s.title ? `<h2 id="faq-${s.id}">${esc(s.title)}</h2>` : `<h2 id="faq-${s.id}" class="sr-only">${esc(faq.title)}</h2>`}
<div class="faq-list">
${s.items
  .map(
    (item) => `<details class="card faq-item" id="q-${item.id}">
<summary><span>${esc(item.q)}</span>${icon("chevron", { size: 18, cls: "chev" })}</summary>
<p>${answer(item.a)}</p>
</details>`,
  )
  .join("\n")}
</div>
</section>`,
    )
    .join("\n");
  const body = `${hero(esc(faq.title), esc(faq.intro), `<p class="page-meta">${ctx.e("faq.from_app")}</p>`)}
<div class="wrap page-body narrow">
${sections}
<p class="center"><a class="btn btn-edge" href="${ctx.site.repoUrl}/issues" rel="noopener">${icon("github", { size: 18 })}${ctx.e("faq.more")}</a></p>
</div>`;
  return { title: ctx.t("faq.title"), description: ctx.t("faq.description"), body };
}

export function renderPrivacy(ctx) {
  const p = ctx.app.privacy;
  const updated = ctx.app.privacyUpdated
    ? `<p class="page-meta">${esc(fill(p.updatedLabel, { date: ctx.date(ctx.app.privacyUpdated) }))} · ${ctx.e("privacy.from_app")}</p>`
    : `<p class="page-meta">${ctx.e("privacy.from_app")}</p>`;
  const toc = [...p.sections.map((s) => ({ id: `privacy-${s.id}`, title: s.title })), { id: "website", title: ctx.t("privacy.site_title") }]
    .map((s) => `<li><a href="#${s.id}">${esc(s.title)}</a></li>`)
    .join("");
  const sections = p.sections
    .map(
      (s) => `<section id="privacy-${s.id}" aria-labelledby="privacy-${s.id}-h">
<h2 id="privacy-${s.id}-h">${esc(s.title)}</h2>
${s.paragraphs.map((x) => `<p>${esc(x)}</p>`).join("\n")}
${s.list.length ? `<ul>${s.list.map((x) => `<li>${esc(x)}</li>`).join("")}</ul>` : ""}
</section>`,
    )
    .join("\n");
  const body = `${hero(esc(p.title), esc(p.intro), updated, false)}
<div class="wrap page-body legal-layout">
<nav class="legal-toc" aria-label="${esc(p.toc)}">
<p class="legal-toc-title">${esc(p.toc)}</p>
<ul>${toc}</ul>
</nav>
<article class="card legal-body">
${sections}
<section id="website" aria-labelledby="website-h">
<h2 id="website-h">${ctx.e("privacy.site_title")}</h2>
<p>${ctx.e("privacy.site_body")}</p>
</section>
</article>
</div>`;
  return { title: ctx.t("privacy.title"), description: ctx.t("privacy.description"), body };
}

function mdLink(repoUrl) {
  return (href) => {
    if (/^(https?:|mailto:)/.test(href)) return href;
    if (href.startsWith("#")) return null;
    return `${repoUrl}/blob/master/${href.replace(/^\.?\//, "")}`;
  };
}

export function renderChangelog(ctx) {
  const { site } = ctx;
  const link = mdLink(site.repoUrl);
  const latest = ctx.app.changelog;
  const summary = latest
    ? `<section class="card md-card" aria-labelledby="summary-h">
<h2 id="summary-h">${ctx.e("changelog.summary")}: <span lang="en" dir="ltr">${esc(latest.heading)}</span></h2>
<div class="md" lang="en" dir="ltr">${markdown(latest.md, { baseLevel: 3, link })}</div>
</section>`
    : "";
  const notes = ctx.app.releaseNotes
    .map(
      (n, i) => `<details class="card release"${i === 0 ? " open" : ""}>
<summary><span class="release-tag" dir="ltr">${esc(n.tag)}</span>${icon("chevron", { size: 18, cls: "chev" })}</summary>
<div class="md" lang="en" dir="ltr">${markdown(n.md, { baseLevel: 3, link })}</div>
<p><a class="text-link" href="${site.repoUrl}/releases/tag/${encodeURIComponent(n.tag)}" rel="noopener">${icon("github", { size: 16 })}${ctx.e("changelog.on_github")}</a></p>
</details>`,
    )
    .join("\n");
  const body = `${hero(ctx.e("changelog.h1"), ctx.e("changelog.intro"))}
<div class="wrap page-body narrow">
${notes ? `<section aria-labelledby="previews-h"><h2 id="previews-h" class="section-title">${ctx.e("changelog.previews")}</h2>${notes}</section>` : ""}
${summary}
</div>`;
  return { title: ctx.t("changelog.title"), description: ctx.t("changelog.description"), body };
}

export function renderNotFound(ctx) {
  const { site, lang } = ctx;
  const links = PAGES.map((p) => `<li><a href="${pageUrl(site.base, lang, p.id)}">${ctx.e(`nav.${p.id}`)}</a></li>`).join("");
  const langs = site.langs
    .filter((l) => l !== lang)
    .map((l) => `<li><a href="${site.base}${l}/" lang="${l}" hreflang="${l}">${esc(site.meta[l].name)}</a></li>`)
    .join("");
  const body = `${hero(ctx.e("notfound.h1"), ctx.e("notfound.body"))}
<div class="wrap page-body narrow">
<ul class="link-list cols">${links}</ul>
<ul class="link-list cols">${langs}</ul>
</div>`;
  return { title: ctx.t("notfound.title"), description: ctx.t("notfound.body"), body };
}
