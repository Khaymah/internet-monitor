import { icon } from "./icons.mjs";
import { navLabel, PAGES, pageUrl } from "./layout.mjs";
import { esc } from "./util.mjs";

export function hero(title, lead, extra = "", eyebrow = "") {
  return `<section class="page-hero">
<div class="wrap app-page">
<header class="doc-hero">
${eyebrow ? `<p class="doc-eyebrow">${eyebrow}</p>` : ""}<h1>${title}</h1>
${lead ? `<p class="secondary">${lead}</p>` : ""}
${extra}
</header>
</div>
</section>`;
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
  const body = `${hero(ctx.e("guide.h1"), ctx.e("guide.intro"), "", ctx.e("guide.eyebrow"))}
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

export function renderNotFound(ctx) {
  const { site, lang } = ctx;
  const links = PAGES.map((p) => `<li><a href="${pageUrl(site.base, lang, p.id)}">${esc(navLabel(ctx, p.id))}</a></li>`).join("");
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
