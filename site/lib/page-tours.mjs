import { icon } from "./icons.mjs";
import { pageUrl } from "./layout.mjs";
import { hero } from "./page-docs.mjs";
import { esc } from "./util.mjs";

export const LEGACY_SETTINGS_ASSET = "internet-monitor-tutorial.mp4";

export function tourAssets(id) {
  const name = `internet-monitor-tour-${id}.mp4`;
  return id === "settings" ? [name, LEGACY_SETTINGS_ASSET] : [name];
}

export function resolveTours(releases, tours) {
  return tours.map((tour) => {
    const names = tourAssets(tour.id);
    const release = releases.list.find((r) => names.some((n) => r.assets[n]));
    if (!release) return { ...tour, video: null };
    const asset = release.assets[names.find((n) => release.assets[n])];
    return { ...tour, video: { url: asset.url, release } };
  });
}

function clock(intl, seconds) {
  const total = Math.max(0, Math.round(seconds));
  const minutes = new Intl.NumberFormat(intl).format(Math.floor(total / 60));
  const rest = new Intl.NumberFormat(intl, { minimumIntegerDigits: 2 }).format(total % 60);
  return `${minutes}:${rest}`;
}

function tourCard(ctx, tour) {
  const { site, lang } = ctx;
  const intl = site.meta[lang].intl;
  const title = esc(tour.text.title);
  const length = tour.seconds > 0 ? `<span class="tour-time">${clock(intl, tour.seconds)}</span>` : "";
  let media;
  if (tour.video) {
    const tracks = tour.tracks
      .map(
        (tr) =>
          `<track kind="subtitles" src="${site.base}tutorial/${tour.id}/${tr.lang}.vtt" srclang="${tr.lang}" label="${esc(site.meta[tr.lang].name)}"${tr.lang === lang ? " default" : ""}>`,
      )
      .join("\n");
    const poster = tour.poster ? ` poster="${site.base}tutorial/${tour.id}/poster.jpg"` : "";
    const note = tour.video.release.prerelease
      ? `<p class="hint">${ctx.e("tutorial.preview_note", { version: tour.video.release.tag })}</p>`
      : "";
    media = `<div class="video-frame">
<video controls preload="none" playsinline${poster} aria-label="${ctx.e("tutorial.video_label", { title: tour.text.title })}">
<source src="${esc(tour.video.url)}" type="video/mp4">
${tracks}
</video>
</div>
${note}
<p><a class="text-link" href="${esc(tour.video.url)}" rel="noopener">${icon("download", { size: 16 })}${ctx.e("tutorial.download_video")}</a></p>`;
  } else {
    media = `<div class="tour-soon">
<span class="well">${icon("play", { size: 22 })}</span>
<strong>${ctx.e("tutorial.soon_title")}</strong>
<span>${ctx.e("tutorial.soon_body")}</span>
</div>`;
  }
  return `<article class="card edge tour" id="${tour.id}">
<div class="tour-head"><h2>${title}</h2>${length}</div>
<p>${esc(tour.text.desc)}</p>
${media}
</article>`;
}

export function renderTutorial(ctx) {
  const { site, lang, tours } = ctx;
  const withTracks = tours.find((t) => t.tracks.length);
  const langs = withTracks ? withTracks.tracks.map((tr) => site.meta[tr.lang].name) : [];
  const subtitles = langs.length
    ? ctx.e("tutorial.subtitles", {
        langs: new Intl.ListFormat(site.meta[lang].intl, { type: "conjunction" }).format(langs),
      })
    : ctx.e("tutorial.no_subtitles");
  const body = `${hero(ctx.e("tutorial.h1"), ctx.e("tutorial.intro"), `<p class="hint">${subtitles}</p>`)}
<div class="wrap page-body">
<div class="tour-grid">
${tours.map((tour) => tourCard(ctx, tour)).join("\n")}
</div>
<p class="center"><a class="btn btn-aurora btn-lg" href="${pageUrl(site.base, lang, "guide")}">${icon("book", { size: 18 })}${ctx.e("tutorial.next")}</a></p>
</div>`;
  return { title: ctx.t("tutorial.title"), description: ctx.t("tutorial.description"), body };
}
