import { markdown } from "./markdown.mjs";
import { appPage, docHero } from "./page-app.mjs";
import { compareVersions, esc } from "./util.mjs";

function anchorOf(id) {
  return `release-${id.replace(/\./g, "-")}`;
}

function versionLabel(a, r) {
  if (r.version === null) return a.t("cl_next");
  if (r.from) return a.t("cl_versions", { from: r.from, to: r.version });
  return `v${r.version}`;
}

function mdLink(repoUrl) {
  return (href) => {
    if (/^(https?:|mailto:)/.test(href)) return href;
    if (href.startsWith("#")) return null;
    return `${repoUrl}/blob/master/${href.replace(/^\.?\//, "")}`;
  };
}

function cardHead(a, { panel, expanded, top, title, badges = "" }) {
  return `<h3 class="cl-card-head">
<button type="button" aria-expanded="${expanded}" aria-controls="${panel}">
<span class="cl-card-top">${top}</span>
<span class="cl-card-title">${title}</span>
<span class="cl-badges">${badges}</span>
${a.svg(a.lucide.ChevronDown, 18, "cl-chevron")}
</button>
</h3>`;
}

function releaseItem(a, { release, groups, total }, latestId, open) {
  const K = a.data.kindStyle;
  const anchor = anchorOf(release.id);
  const panel = `${anchor}-panel`;
  const top = `<span class="cl-version num">${esc(versionLabel(a, release))}</span>${
    release.date ? `<time datetime="${release.date}">${esc(a.fmt.date(release.date))}</time>` : ""
  }${release.id === latestId ? `<span class="tag cl-latest">${esc(a.t("cl_latest"))}</span>` : ""}`;
  const badges = `${groups
    .map(
      (g) =>
        `<span class="cl-badge num" style="--kind: ${K[g.kind].tone}" title="${esc(a.t(K[g.kind].label))}" data-kind="${g.kind}">${a.svg(K[g.kind].icon, 12)}<span class="sr-only">${esc(a.t(K[g.kind].label))}</span>${esc(a.fmt.number(g.items.length))}</span>`,
    )
    .join("")}<span class="sr-only">${esc(a.t("cl_change_count", { n: a.fmt.number(total) }))}</span>`;
  const search = [release.version, release.from, release.id, a.t(release.title)].filter(Boolean).join(" ");
  const body = groups
    .map(
      (g) => `<section class="cl-group" style="--kind: ${K[g.kind].tone}" data-kind="${g.kind}" aria-labelledby="${panel}-${g.kind}">
<h4 id="${panel}-${g.kind}" class="cl-group-head"><span class="cl-group-icon">${a.svg(K[g.kind].icon, 14)}</span>${esc(a.t(K[g.kind].label))}<span class="cl-group-count num">${esc(a.fmt.number(g.items.length))}</span></h4>
<ul>${g.items.map((key) => `<li>${esc(a.t(key))}</li>`).join("")}</ul>
</section>`,
    )
    .join("\n");
  return `<li id="${anchor}" data-release="${release.id}" data-search="${esc(search)}" class="cl-release"${release.version === null ? " data-next" : ""}>
<span class="cl-node" aria-hidden="true"></span>
<article class="card cl-card"${open ? " data-open" : ""}>
${cardHead(a, { panel, expanded: open, top, title: esc(a.t(release.title)), badges })}
<div id="${panel}" class="cl-panel"${open ? "" : " hidden"}>
${body}
</div>
</article>
</li>`;
}

function previewItem(ctx, notes) {
  const a = ctx.appx;
  const anchor = "release-next";
  const panel = `${anchor}-panel`;
  const link = mdLink(ctx.site.repoUrl);
  const base = notes[0].tag.replace(/-.*$/, "");
  const top = `<span class="cl-version num">${esc(a.t("cl_next"))}</span><span class="tag" dir="ltr">${esc(base)}</span>`;
  const body = notes
    .map(
      (n) => `<section class="cl-group cl-preview" style="--kind: var(--accent-blue)">
<h4 class="cl-group-head" dir="ltr"><span class="num">${esc(n.tag)}</span><a class="text-link cl-preview-link" href="${ctx.site.repoUrl}/releases/tag/${encodeURIComponent(n.tag)}" target="_blank" rel="noreferrer noopener" dir="auto">${ctx.e("changelog.on_github")}</a></h4>
<div class="md" lang="en" dir="ltr">${markdown(n.md, { baseLevel: 4, link })}</div>
</section>`,
    )
    .join("\n");
  return `<li id="${anchor}" data-release="next" data-search="${esc(notes.map((n) => n.tag).join(" "))}" data-preview class="cl-release" data-next>
<span class="cl-node" aria-hidden="true"></span>
<article class="card cl-card">
${cardHead(a, { panel, expanded: false, top, title: ctx.e("changelog.preview_title") })}
<div id="${panel}" class="cl-panel" hidden>
${body}
</div>
</article>
</li>`;
}

export function renderChangelog(ctx) {
  const a = ctx.appx;
  const { releases, kinds, kindStyle: K, nextMonth } = a.data;
  const timeline = a.data.buildTimeline(releases, "all", "", a.t);
  const latestId = releases.find((r) => r.version !== null)?.id;
  const firstId = releases[0]?.id;
  const released = releases.find((r) => r.version !== null)?.version;
  const previews = ctx.app.releaseNotes.filter(
    (n) => n.tag.includes("-") && (!released || compareVersions(n.tag.replace(/-.*$/, ""), released) > 0),
  );
  const counts = { new: 0, improved: 0, fixed: 0, security: 0 };
  for (const r of releases) for (const i of r.items) counts[i.kind] += 1;
  const stats = [
    ["cl_stat_releases", releases.length],
    ["cl_stat_new", counts.new],
    ["cl_stat_improved", counts.improved],
    ["cl_stat_fixed", counts.fixed + counts.security],
  ]
    .map(([key, n]) => `<div class="cl-stat"><dt>${esc(a.t(key))}</dt><dd class="num">${esc(a.fmt.number(n))}</dd></div>`)
    .join("");
  const monthLabel = (key) => (key === nextMonth ? a.t("cl_in_progress") : a.fmt.month(key));
  const months = [
    ...(previews.length
      ? [{ key: "next", label: a.t("cl_in_progress"), map: [{ id: "next", label: a.t("cl_next") }], items: [previewItem(ctx, previews)] }]
      : []),
    ...timeline.months.map((m) => ({
      key: m.key,
      label: monthLabel(m.key),
      map: m.releases.map(({ release }) => ({ id: release.id, label: release.version === null ? a.t("cl_next") : `v${release.version}` })),
      items: m.releases.map((r) => releaseItem(a, r, latestId, r.release.id === firstId)),
    })),
  ];
  const map = months
    .map(
      (m) => `<div class="cl-map-month"><span class="cl-map-label">${esc(m.label)}</span>${m.map
        .map((r) => `<a href="#${anchorOf(r.id)}" class="num">${esc(r.label)}</a>`)
        .join("")}</div>`,
    )
    .join("\n");
  const sections = months
    .map(
      (m) => `<section class="cl-month" aria-labelledby="month-${m.key}">
<h2 id="month-${m.key}" class="cl-month-head"><span>${esc(m.label)}</span></h2>
<ol class="cl-releases">
${m.items.join("\n")}
</ol>
</section>`,
    )
    .join("\n");
  const chips = ["all", ...kinds]
    .map((k) => {
      const style = k === "all" ? "" : ` style="--kind: ${K[k].tone}"`;
      const icon = k === "all" ? "" : a.svg(K[k].icon, 14);
      const label = k === "all" ? a.t("cl_all") : a.t(K[k].label);
      return `<button type="button" class="chip" data-filter="${k}" aria-pressed="${k === "all"}"${style}>${icon}${esc(label)}</button>`;
    })
    .join("");
  const stringsAttr = esc(
    JSON.stringify({
      results: a.t("cl_results"),
      expand: a.t("cl_expand_all"),
      collapse: a.t("cl_collapse_all"),
      intl: ctx.site.meta[ctx.lang].intl,
    }),
  );
  const body = `${docHero({
    eyebrow: a.t("cl_eyebrow"),
    title: a.t("cl_title"),
    intro: a.t("cl_intro"),
    after: `\n<dl class="cl-stats">${stats}</dl>`,
  })}
<div class="cl-toolbar">
<label class="cl-search">${a.svg(a.lucide.Search, 16)}<span class="sr-only">${esc(a.t("cl_search"))}</span><input type="search" placeholder="${esc(a.t("cl_search_placeholder"))}"></label>
<fieldset class="cl-filter" aria-label="${esc(a.t("cl_filter"))}">${chips}</fieldset>
</div>
<div class="cl-summary">
<p role="status" class="cl-count"></p>
<button type="button" class="btn btn-ghost btn-sm cl-toggle-all">${esc(a.t("cl_expand_all"))}</button>
</div>
<div class="cl-layout" data-strings="${stringsAttr}">
<nav class="cl-map" aria-label="${esc(a.t("cl_jump"))}">
<span class="cl-map-title">${esc(a.t("cl_map"))}</span>
${map}
</nav>
<div class="cl-timeline">
<div class="card cl-empty" hidden>
<p>${esc(a.t("cl_no_results"))}</p>
<button type="button" class="btn btn-edge btn-sm cl-clear">${a.svg(a.lucide.X, 14)}${esc(a.t("cl_clear"))}</button>
</div>
${sections}
</div>
</div>`;
  return {
    title: ctx.t("changelog.title"),
    description: ctx.t("changelog.description"),
    body: appPage(body, "cl-page"),
    scripts: ["changelog.js"],
  };
}
