import { icon } from "./icons.mjs";
import { pageUrl } from "./layout.mjs";
import { esc } from "./util.mjs";

const BENEFIT_STYLE = [
  { icon: "clock", accent: "var(--accent-blue)" },
  { icon: "router", accent: "var(--accent-violet)" },
  { icon: "send", accent: "var(--accent-coral)" },
  { icon: "gauge", accent: "var(--accent-green)" },
];

export function shotKey(file) {
  return file.replace(/^\d+-/, "").replace(/\.png$/, "");
}

export function osLabels(ctx) {
  return ["windows", "mac", "linux"]
    .map((os) => `data-label-${os}="${ctx.e("home.cta_download_os", { os: ctx.t(`download.os.${os}`) })}"`)
    .join(" ");
}

export function renderHome(ctx) {
  const { site, lang, releases } = ctx;
  const shots = ctx.shots;
  const hero = shots.files.find((f) => shotKey(f) === "overview") ?? shots.files[0];
  const rest = shots.files.filter((f) => f !== hero);
  const shotSrc = (f) => `${site.base}assets/shots/${shots.lang}/${f}`;
  const version = releases.stable
    ? `<p class="hero-meta">${ctx.e("home.version_note", { version: releases.stable.tag, date: ctx.date(releases.stable.date) })}</p>`
    : "";
  const benefits = ctx
    .raw("home.benefits")
    .map((b, i) => {
      const s = BENEFIT_STYLE[i % BENEFIT_STYLE.length];
      return `<article class="card card-flare benefit" style="--flare: ${s.accent}">
<span class="well">${icon(s.icon, { size: 22 })}</span>
<h3>${esc(b.title)}</h3>
<p>${esc(b.body)}</p>
</article>`;
    })
    .join("\n");
  const gallery = rest
    .map((f) => {
      const key = shotKey(f);
      const caption = ctx.raw(`home.shots.${key}`) ?? "";
      return `<figure class="shot">
<div class="frame"><img src="${shotSrc(f)}" alt="${esc(caption)}" width="1920" height="1080" loading="lazy" decoding="async"></div>
<figcaption>${esc(caption)}</figcaption>
</figure>`;
    })
    .join("\n");
  const promise = ctx
    .raw("home.promise")
    .map((p) => `<li>${icon("check", { size: 18 })}<span>${esc(p)}</span></li>`)
    .join("");
  const dl = pageUrl(site.base, lang, "download");
  const cta = `<a class="btn btn-aurora btn-xl" href="${dl}">${icon("download", { size: 20 })}<span data-os-label ${osLabels(ctx)}>${ctx.e("home.cta_download")}</span></a>`;

  const body = `<section class="hero">
<div class="wrap hero-inner">
<p class="eyebrow">${ctx.e("home.eyebrow")}</p>
<h1 class="hero-title"><span class="aurora-text">${ctx.e("home.hero_title")}</span></h1>
<p class="hero-sub">${ctx.e("home.hero_sub")}</p>
<div class="hero-actions">
${cta}
<a class="btn btn-edge btn-xl" href="${pageUrl(site.base, lang, "guide")}">${ctx.e("home.cta_guide")}${icon("arrow", { size: 18, cls: "flip-rtl" })}</a>
</div>
${version}
</div>
${
  hero
    ? `<div class="wrap"><figure class="hero-shot"><div class="frame frame-glow"><img src="${shotSrc(hero)}" alt="${ctx.e("home.hero_alt")}" width="1920" height="1080" fetchpriority="high" decoding="async"></div></figure></div>`
    : ""
}
</section>

<section class="section" aria-labelledby="benefits-h">
<div class="wrap">
<header class="section-head">
<h2 id="benefits-h">${ctx.e("home.benefits_title")}</h2>
<p>${ctx.e("home.benefits_sub")}</p>
</header>
<div class="benefits">
${benefits}
</div>
</div>
</section>

${
  gallery
    ? `<section class="section" aria-labelledby="shots-h">
<div class="wrap">
<header class="section-head">
<h2 id="shots-h">${ctx.e("home.shots_title")}</h2>
<p>${ctx.e("home.shots_sub")}</p>
</header>
<div class="gallery">
${gallery}
</div>
</div>
</section>`
    : ""
}

<section class="section" aria-labelledby="promise-h">
<div class="wrap">
<div class="card edge promise">
<div class="promise-text">
<span class="well well-lg">${icon("shield", { size: 28 })}</span>
<h2 id="promise-h">${ctx.e("home.promise_title")}</h2>
<p>${ctx.e("home.promise_sub")}</p>
<a class="text-link" href="${pageUrl(site.base, lang, "privacy")}">${ctx.e("home.promise_more")}${icon("arrow", { size: 16, cls: "flip-rtl" })}</a>
</div>
<ul class="checks">${promise}</ul>
</div>
</div>
</section>

<section class="section final" aria-labelledby="final-h">
<div class="wrap final-inner">
<h2 id="final-h">${ctx.e("home.final_title")}</h2>
<p>${ctx.e("home.final_sub")}</p>
${cta}
</div>
</section>`;

  return {
    title: ctx.t("home.title"),
    description: ctx.t("home.description"),
    body,
  };
}
