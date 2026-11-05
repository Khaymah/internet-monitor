import { icon } from "./icons.mjs";
import { pageUrl } from "./layout.mjs";
import { PLATFORMS, resolveAsset } from "./releases.mjs";
import { esc } from "./util.mjs";

function fileRow(ctx, file) {
  const asset = resolveAsset(ctx.releases, file.asset);
  const size = asset.size ? `<span class="size" dir="ltr">${esc(ctx.size(asset.size))}</span>` : "";
  const badge = asset.preview
    ? `<span class="badge badge-preview">${icon("flask", { size: 14 })}${ctx.e("download.preview_only")} · ${esc(asset.release.tag)}</span>`
    : "";
  const desc = file.desc ? `<span class="file-desc">${ctx.e(`download.${file.desc}`)}</span>` : "";
  const cls = file.main ? "btn-aurora" : "btn-edge";
  return `<div class="file${file.main ? " file-main" : ""}">
<div class="file-text">
<strong>${ctx.e(`download.${file.label}`)}</strong>
${desc}
<code class="file-name" dir="ltr">${esc(file.asset)}</code>
${badge}
</div>
<a class="btn ${cls}" href="${esc(asset.url)}" rel="noopener">${icon("download", { size: 18 })}<span>${ctx.e("download.btn")}</span>${size}</a>
</div>`;
}

function platformCard(ctx, p) {
  const groups = p.groups
    .map((g) => {
      const title = g.title ? `<h3 class="group-title">${ctx.e(`download.${g.title}`)}</h3>` : "";
      const arch = g.arch ? ` data-arch="${g.arch}"` : "";
      return `<div class="file-group"${arch}>${title}${g.files.map((f) => fileRow(ctx, f)).join("\n")}</div>`;
    })
    .join("\n");
  const note = p.note ? `<p class="hint">${ctx.md(`download.${p.note}`)}</p>` : "";
  const steps = ctx
    .raw(`download.steps.${p.id}`)
    .map((s) => `<li>${ctx.inline(s)}</li>`)
    .join("");
  const piLink =
    p.id === "pi"
      ? `<p><a class="text-link" href="${ctx.site.repoUrl}/blob/master/docs/headless.md" rel="noopener">${ctx.e("download.pi_guide")}${icon("arrow", { size: 16, cls: "flip-rtl" })}</a></p>`
      : "";
  return `<section class="card platform" id="os-${p.id}" data-os="${p.id}" aria-labelledby="os-${p.id}-h">
<header class="platform-head">
<span class="well">${icon(p.icon, { size: 22 })}</span>
<div>
<h2 id="os-${p.id}-h">${ctx.e(`download.os.${p.id}`)}</h2>
<p>${ctx.e(`download.os_sub.${p.id}`)}</p>
</div>
</header>
${note}
${groups}
<details class="install">
<summary>${icon("book", { size: 18 })}<span>${ctx.e("download.install_title")}</span>${icon("chevron", { size: 16, cls: "chev" })}</summary>
<p class="hint">${ctx.e("download.install_sub")}</p>
<ol class="steps">${steps}</ol>
${piLink}
</details>
</section>`;
}

export function renderDownload(ctx) {
  const { releases, site } = ctx;
  const latest = releases.stable
    ? `<p class="page-meta">${ctx.e("download.latest", { version: releases.stable.tag, date: ctx.date(releases.stable.date) })}</p>`
    : "";
  const fallback = releases.ok
    ? ""
    : `<div class="notice" role="note">${icon("alert", { size: 20 })}<p>${ctx.e("download.fallback")}</p><a class="btn btn-edge btn-sm" href="${esc(releases.page)}" rel="noopener">${ctx.e("download.fallback_btn")}</a></div>`;
  const detect = ["windows", "mac", "linux", "pi"]
    .map(
      (os) =>
        `<p class="detect-msg" data-os-msg="${os}" hidden>${ctx.e("download.detected", { os: ctx.t(`download.os.${os}`) })}</p>`,
    )
    .join("");
  const preview = releases.preview
    ? `<aside class="card edge preview-box" aria-labelledby="preview-h">
<span class="well">${icon("flask", { size: 22 })}</span>
<div>
<h2 id="preview-h">${ctx.e("download.preview_title")}</h2>
<p>${ctx.e("download.preview_body", { version: releases.preview.tag, date: ctx.date(releases.preview.date) })}</p>
</div>
<a class="btn btn-edge" href="${esc(releases.preview.url)}" rel="noopener">${ctx.e("download.preview_btn")}</a>
</aside>`
    : "";
  const sums = resolveAsset(releases, "SHA256SUMS");
  const body = `<section class="page-hero">
<div class="wrap">
<h1>${ctx.e("download.h1")}</h1>
<p class="lead">${ctx.e("download.intro")}</p>
${latest}
</div>
</section>
<div class="wrap page-body">
${fallback}
<div class="detect" aria-live="polite">
<p class="detect-msg" data-os-msg="none">${ctx.e("download.pick")}</p>
<p class="detect-msg" data-os-msg="phone" hidden>${ctx.e("download.phone")}</p>
${detect}
</div>
<div class="platforms">
${PLATFORMS.map((p) => platformCard(ctx, p)).join("\n")}
</div>
${preview}
<section class="advanced" aria-labelledby="adv-h">
<h2 id="adv-h">${ctx.e("download.advanced_title")}</h2>
<ul class="link-list">
<li><a href="${esc(sums.url)}" rel="noopener">${icon("key", { size: 16 })}${ctx.e("download.advanced_sums")}</a></li>
<li><a href="${esc(releases.all)}" rel="noopener">${icon("github", { size: 16 })}${ctx.e("download.advanced_all")}</a></li>
<li><a href="${site.repoUrl}#%EF%B8%8F-build-from-source" rel="noopener">${icon("linux", { size: 16 })}${ctx.e("download.advanced_source")}</a></li>
</ul>
<p class="hint"><a href="${pageUrl(site.base, ctx.lang, "guide")}">${ctx.e("guide.h1")}</a></p>
</section>
</div>`;
  return { title: ctx.t("download.title"), description: ctx.t("download.description"), body };
}
