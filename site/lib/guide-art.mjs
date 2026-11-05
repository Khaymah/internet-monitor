import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { icon } from "./icons.mjs";
import { esc } from "./util.mjs";

const GUIDE_DIR = join(dirname(fileURLToPath(import.meta.url)), "..", "assets", "guide");

export const GUIDE_STEPS = [{ art: "file" }, { art: "tray" }, { shot: "status" }, { shot: "details" }, { shot: "trouble" }, { shot: "send" }];

let manifest = null;

function sizes() {
  if (!manifest) manifest = JSON.parse(readFileSync(join(GUIDE_DIR, "manifest.json"), "utf8"));
  return manifest;
}

export function guideShot(ctx, id, theme) {
  const key = `${ctx.lang}/${id}-${theme}`;
  const size = sizes()[key];
  if (!size) throw new Error(`guide screenshot ${key} is missing, run site/tools/guide-shots.cjs`);
  return { src: `${ctx.site.base}assets/guide/${key}.webp`, width: size.w, height: size.h };
}

export function guideImageUrl(ctx, index) {
  const step = GUIDE_STEPS[index];
  if (!step?.shot) return null;
  const shot = guideShot(ctx, step.shot, "light");
  return `${ctx.site.origin}${shot.src}`;
}

function picture(ctx, id, alt) {
  const light = guideShot(ctx, id, "light");
  const dark = guideShot(ctx, id, "dark");
  const img = (cls, s) =>
    `<img class="${cls}" src="${s.src}" width="${s.width}" height="${s.height}" alt="${esc(alt)}" loading="lazy" decoding="async">`;
  return `<figure class="step-shot" style="--w:${light.width}px">${img("shot-light", light)}${img("shot-dark", dark)}</figure>`;
}

function fileArt(ctx, alt) {
  return `<figure class="step-art art-file" role="img" aria-label="${esc(alt)}">
<div class="art-tile is-hit" aria-hidden="true">
<img src="${ctx.site.base}assets/favicon.png" width="56" height="56" alt="">
<span class="art-name" dir="ltr">${ctx.e("guide.art.file")}</span>
<span class="art-pointer">${icon("pointer", { size: 26 })}</span>
</div>
<span class="art-arrow" aria-hidden="true">${icon("download", { size: 26 })}</span>
</figure>`;
}

function trayArt(ctx, alt) {
  const dots = [0, 1, 2].map(() => `<span class="art-dot"></span>`).join("");
  return `<figure class="step-art art-tray" role="img" aria-label="${esc(alt)}">
<div class="art-stage" aria-hidden="true">
<div class="art-menu">
<span class="art-menu-item is-hit"><img src="${ctx.site.base}assets/favicon.png" width="18" height="18" alt="">${ctx.e("guide.art.menu")}</span>
</div>
<div class="art-bar">
<span class="art-more">${icon("chevron", { size: 16, cls: "up" })}</span>
<span class="art-app is-hit"><img src="${ctx.site.base}assets/favicon.png" width="22" height="22" alt=""></span>
${dots}
<span class="art-clock" dir="ltr">${ctx.e("guide.art.clock")}</span>
</div>
</div>
</figure>`;
}

export function guideVisual(ctx, index, alt) {
  const step = GUIDE_STEPS[index];
  if (step.shot) return picture(ctx, step.shot, alt);
  return step.art === "file" ? fileArt(ctx, alt) : trayArt(ctx, alt);
}
