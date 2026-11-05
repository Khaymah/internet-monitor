import { icon } from "./icons.mjs";
import { pageUrl } from "./layout.mjs";
import { hero } from "./page-docs.mjs";
import { PI_FILES, PLATFORMS, resolveAsset } from "./releases.mjs";
import { esc } from "./util.mjs";

function sizeOf(ctx, asset) {
  return asset.size ? ctx.size(asset.size) : "";
}

function previewBadge(ctx, asset) {
  return asset.preview
    ? `<span class="badge badge-preview">${icon("flask", { size: 14 })}${ctx.e("download.preview_only")} · ${esc(asset.release.tag)}</span>`
    : "";
}

function downloadButton(ctx, platform, variant, primary) {
  const asset = resolveAsset(ctx.releases, variant.asset);
  const size = sizeOf(ctx, asset);
  const chip = variant.arch ? ctx.t(`download.mac_${variant.arch}`) : "";
  const sub = [chip, size ? `‎${size}` : ""].filter(Boolean).join(" · ");
  const label = primary ? ctx.e("download.get", { os: ctx.t(`download.os.${platform.id}`) }) : ctx.e("download.btn");
  const arch = variant.arch ? ` data-arch="${variant.arch}"` : "";
  const cls = primary ? "btn btn-aurora btn-xl dl-btn" : "btn btn-edge dl-btn dl-btn-second";
  return `<a class="${cls}"${arch} href="${esc(asset.url)}" rel="noopener">${icon("download", { size: primary ? 24 : 18 })}<span class="dl-btn-text"><strong>${label}</strong>${sub ? `<small>${esc(sub)}</small>` : ""}</span></a>`;
}

function fileNames(ctx, variants) {
  const names = variants.map((v) => `<code dir="ltr"${v.arch ? ` data-arch="${v.arch}"` : ""}>${esc(v.asset)}</code>`).join(" ");
  return `<p class="hint dl-files">${ctx.e("download.file_label")} ${names}</p>`;
}

function primaryBlock(ctx, platform) {
  const tray = platform.editions.tray;
  const badge = previewBadge(ctx, resolveAsset(ctx.releases, tray[0].asset));
  const mac =
    platform.id === "mac"
      ? `<p class="hint mac-hint">${ctx.md("download.mac_which")}</p>
<button type="button" class="text-link mac-switch" hidden>${ctx.e("download.mac_show_both")}</button>`
      : "";
  return `<div class="dl-primary">
<div class="dl-variants">${tray.map((v) => downloadButton(ctx, platform, v, true)).join("")}</div>
${mac}
<p class="dl-edition"><strong>${ctx.e("download.tray")}</strong><span>${ctx.e("download.tray_desc")}</span>${badge}</p>
${fileNames(ctx, tray)}
<p class="hint dl-pick">${ctx.e("download.pick_first")}</p>
</div>`;
}

function secondaryBlock(ctx, platform) {
  const win = platform.editions.window;
  const badge = previewBadge(ctx, resolveAsset(ctx.releases, win[0].asset));
  return `<div class="dl-secondary">
<h3>${ctx.e("download.second_title")}</h3>
<p class="dl-edition"><strong>${ctx.e("download.window")}</strong><span>${ctx.e("download.window_desc")}</span>${badge}</p>
<div class="dl-variants">${win.map((v) => downloadButton(ctx, platform, v, false)).join("")}</div>
${fileNames(ctx, win)}
</div>`;
}

function stepsBlock(ctx, platform) {
  const steps = ctx.raw(`download.steps.${platform.id}`);
  const more = ctx.raw("download.more")[platform.id];
  const warn = ctx.raw("download.warn")[platform.id];
  const items = steps
    .map((s, i) => {
      const extra =
        more && i === 1
          ? `<details class="step-more"><summary>${ctx.e("download.more_btn")}${icon("chevron", { size: 14, cls: "chev" })}</summary><p>${ctx.md(`download.more.${platform.id}`)}</p></details>`
          : "";
      return `<li><span class="well">${icon(platform.stepIcons[i], { size: 20 })}</span><div class="dl-step-text"><p>${ctx.inline(s)}</p>${extra}</div></li>`;
    })
    .join("");
  const note = warn
    ? `<p class="dl-note">${icon("alert", { size: 18 })}<span>${ctx.md(`download.warn.${platform.id}`)}</span></p>`
    : "";
  return `<div class="dl-steps-wrap">
<h3>${ctx.e("download.install_title")}</h3>
<ol class="dl-steps">${items}</ol>
${note}
<p class="hint">${icon("folder", { size: 16 })} ${ctx.md("download.folder_tip")}</p>
</div>`;
}

function platformPanel(ctx, p) {
  return `<section class="card edge dl-os" id="os-${p.id}" data-os="${p.id}" aria-labelledby="os-${p.id}-h">
<header class="dl-os-head">
<span class="well well-lg">${icon(p.icon, { size: 26 })}</span>
<div>
<h2 id="os-${p.id}-h">${ctx.e(`download.os.${p.id}`)}</h2>
<p>${ctx.e(`download.os_sub.${p.id}`)}</p>
</div>
</header>
${primaryBlock(ctx, p)}
${secondaryBlock(ctx, p)}
${stepsBlock(ctx, p)}
</section>`;
}

function chooser(ctx) {
  const tiles = [...PLATFORMS, { id: "pi", icon: "pi", advanced: true }]
    .map(
      (p) =>
        `<li data-os="${p.id}"><a class="dl-tile" href="#os-${p.id}"><span class="well">${icon(p.icon, { size: 22 })}</span><span class="dl-tile-text"><strong>${ctx.e(`download.os.${p.id}`)}</strong><small>${ctx.e(p.advanced ? "download.advanced_title" : `download.os_sub.${p.id}`)}</small></span>${icon("arrow", { size: 18, cls: "flip-rtl" })}</a></li>`,
    )
    .join("");
  return `<details class="dl-other" id="dl-other" open>
<summary><span data-alt="${ctx.e("download.other_title_alt")}">${ctx.e("download.other_title")}</span>${icon("chevron", { size: 16, cls: "chev" })}</summary>
<p class="hint">${ctx.e("download.other_sub")}</p>
<ul class="dl-tiles">${tiles}</ul>
</details>`;
}

function piFile(ctx, file) {
  const asset = resolveAsset(ctx.releases, file.asset);
  const size = sizeOf(ctx, asset);
  return `<li class="dl-pi-file">
<div class="dl-pi-text"><strong>${ctx.e(`download.${file.label}`)}</strong><code dir="ltr">${esc(file.asset)}</code>${previewBadge(ctx, asset)}</div>
<a class="btn btn-edge btn-sm" href="${esc(asset.url)}" rel="noopener">${icon("download", { size: 16 })}<span>${ctx.e("download.btn")}</span>${size ? `<span class="size" dir="ltr">${esc(size)}</span>` : ""}</a>
</li>`;
}

function advanced(ctx) {
  const { releases, site } = ctx;
  const sums = resolveAsset(releases, "SHA256SUMS");
  const steps = ctx
    .raw("download.steps.pi")
    .map((s) => `<li>${ctx.inline(s)}</li>`)
    .join("");
  return `<details class="dl-advanced" id="os-pi">
<summary>${icon("pi", { size: 20 })}<span>${ctx.e("download.advanced_title")}</span>${icon("chevron", { size: 16, cls: "chev" })}</summary>
<p class="hint">${ctx.e("download.advanced_sub")}</p>
<h3>${ctx.e("download.os.pi")}</h3>
<p>${ctx.e("download.pi_note")}</p>
<ul class="dl-pi-files">${PI_FILES.map((f) => piFile(ctx, f)).join("")}</ul>
<ol class="steps">${steps}</ol>
<p><a class="text-link" href="${site.repoUrl}/blob/master/docs/headless.md" rel="noopener">${ctx.e("download.pi_guide")}${icon("arrow", { size: 16, cls: "flip-rtl" })}</a></p>
<ul class="link-list">
<li><a href="${esc(sums.url)}" rel="noopener">${icon("key", { size: 16 })}${ctx.e("download.advanced_sums")}</a></li>
<li><a href="${esc(releases.all)}" rel="noopener">${icon("github", { size: 16 })}${ctx.e("download.advanced_all")}</a></li>
<li><a href="${site.repoUrl}#%EF%B8%8F-build-from-source" rel="noopener">${icon("linux", { size: 16 })}${ctx.e("download.advanced_source")}</a></li>
</ul>
</details>`;
}

function previewBox(ctx) {
  const { preview } = ctx.releases;
  if (!preview) return "";
  return `<aside class="card edge preview-box" aria-labelledby="preview-h">
<span class="well">${icon("flask", { size: 22 })}</span>
<div>
<h2 id="preview-h">${ctx.e("download.preview_title")}</h2>
<p>${ctx.e("download.preview_body", { version: preview.tag, date: ctx.date(preview.date) })}</p>
</div>
<a class="btn btn-edge" href="${esc(preview.url)}" rel="noopener">${ctx.e("download.preview_btn")}</a>
</aside>`;
}

export function renderDownload(ctx) {
  const { releases, site } = ctx;
  const latest = releases.stable
    ? `<p class="page-meta">${ctx.e("download.latest", { version: releases.stable.tag, date: ctx.date(releases.stable.date) })}</p>`
    : "";
  const fallback = releases.ok
    ? ""
    : `<div class="banner warn" role="note">${icon("alert", { size: 18 })}<div class="banner-body">${ctx.e("download.fallback")}</div><div class="banner-actions"><a class="btn btn-edge btn-sm" href="${esc(releases.page)}" rel="noopener">${ctx.e("download.fallback_btn")}</a></div></div>`;
  const detect = ["windows", "mac", "linux", "pi"]
    .map(
      (os) =>
        `<p class="detect-msg" data-os-msg="${os}" hidden>${ctx.e(os === "pi" ? "download.detected_pi" : "download.detected", { os: ctx.t(`download.os.${os}`) })}</p>`,
    )
    .join("");
  const body = `${hero(ctx.e("download.h1"), ctx.e("download.intro"), latest, ctx.e("download.eyebrow"))}
<div class="wrap page-body narrow dl-page">
${fallback}
<div class="dl-detect" aria-live="polite">
<p class="detect-msg" data-os-msg="phone" hidden>${ctx.e("download.phone")}</p>
${detect}
</div>
${PLATFORMS.map((p) => platformPanel(ctx, p)).join("\n")}
${chooser(ctx)}
${advanced(ctx)}
${previewBox(ctx)}
<p class="hint"><a class="text-link" href="${pageUrl(site.base, ctx.lang, "guide")}">${ctx.e("download.help")}</a></p>
</div>`;
  return { title: ctx.t("download.title"), description: ctx.t("download.description"), body };
}
