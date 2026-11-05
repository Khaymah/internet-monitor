import { richText } from "./app.mjs";
import { esc } from "./util.mjs";

const TONE = {
  green: "var(--accent-green)",
  violet: "var(--accent-violet)",
  blue: "var(--accent-blue)",
  coral: "var(--accent-coral)",
};

const NEW_TAB = 'target="_blank" rel="noreferrer noopener"';

export function docHero({ eyebrow, title, intro, before = "", after = "" }) {
  return `<header class="doc-hero">
${before}${eyebrow ? `<p class="doc-eyebrow">${esc(eyebrow)}</p>` : ""}
<h1>${esc(title)}</h1>
${intro ? `<p class="secondary">${esc(intro)}</p>` : ""}${after}
</header>`;
}

export function appPage(body, cls = "") {
  return `<div class="wrap app-page">
<div class="doc-page${cls ? ` ${cls}` : ""}">
${body}
</div>
</div>`;
}

function newTab(a) {
  return `<span class="sr-only"> ${esc(a.t("opens_new_tab"))}</span>`;
}

function banner(a, tone, icon, text) {
  const role = tone === "warn" || tone === "danger" ? "alert" : "status";
  return `<div class="banner ${tone}" role="${role}">${a.svg(icon, 18)}<div class="banner-body">${esc(text)}</div></div>`;
}

function copyButton(a, name, value, withLabel) {
  const label = esc(a.t("contact_copy", { name }));
  const cls = withLabel ? "btn btn-edge btn-sm" : "btn btn-ghost btn-sm btn-icon";
  return `<button type="button" class="${cls} copy-btn" data-copy="${esc(value)}" aria-label="${label}" title="${label}">${a.svg(a.lucide.Copy, 15)}${withLabel ? label : ""}</button>`;
}

function toaster(a) {
  const L = a.lucide;
  return `<div class="toasts" aria-live="polite" aria-relevant="additions" data-ok="${esc(a.t("contact_copied"))}" data-error="${esc(a.t("contact_copy_failed"))}" data-close="${esc(a.t("close"))}">
<template data-icon="ok">${a.svg(L.CircleCheck, 18)}</template>
<template data-icon="error">${a.svg(L.CircleAlert, 18)}</template>
<template data-icon="close">${a.svg(L.X, 14)}</template>
</div>`;
}

function channelCard(a, { icon, accent, name, handle, body, href, copy }) {
  const external = href && !href.startsWith("mailto:");
  const open = href
    ? `<a class="btn btn-edge btn-sm" href="${esc(href)}"${external ? ` ${NEW_TAB}` : ""}>${esc(a.t("contact_open", { name }))}${external ? `${a.svg(a.lucide.ExternalLink, 14)}${newTab(a)}` : ""}</a>`
    : "";
  return `<li class="card card-flare contact-card" style="--flare: ${TONE[accent]}; --tone: ${TONE[accent]}">
<span class="contact-icon">${a.svg(icon, 20)}</span>
<h2>${esc(name)}</h2>
<p class="contact-handle ltr" dir="ltr">${esc(handle)}</p>
<p class="contact-best">${esc(body)}</p>
<div class="contact-actions">
${open}${copyButton(a, name, copy, !href)}
</div>
</li>`;
}

export function renderAbout(ctx) {
  const a = ctx.appx;
  const { about, donate } = a.data;
  const L = a.lucide;
  const principles = about.PRINCIPLES.map(
    (p) => `<li class="card">
<span class="icon-well">${a.svg(p.icon, 18)}</span>
<h3>${esc(a.t(p.title))}</h3>
<p>${esc(a.t(p.body))}</p>
</li>`,
  ).join("\n");
  const stable = ctx.releases.stable;
  const version = stable
    ? `<div><dt>${esc(a.t("about_version"))}</dt><dd class="num"><a href="${a.href("changelog")}">${esc(stable.tag.replace(/^v/, ""))}</a></dd></div>`
    : "";
  const links = [
    ["changelog", "nav_changelog"],
    ["faq", "nav_faq"],
    ["privacy", "nav_privacy"],
    ["terms", "nav_terms"],
  ]
    .map(([id, key]) => `<a href="${a.href(id)}">${esc(a.t(key))}</a>`)
    .join("\n");
  const support = donate.donateMethods().length
    ? `<a class="btn btn-ghost" href="${a.href("support")}">${a.svg(L.HeartHandshake, 17)}${esc(a.t("nav_support"))}</a>`
    : "";
  const logo = `<img src="${ctx.site.base}assets/favicon.png" alt="" width="64" height="64" class="about-logo">\n`;
  const body = `${docHero({ eyebrow: a.t("about_eyebrow"), title: a.t("about_title"), before: logo })}
<section class="card about-story" aria-label="${esc(a.t("about_eyebrow"))}">
${about.ABOUT_STORY.map((p) => `<p>${esc(a.t(p))}</p>`).join("\n")}
</section>
<section aria-labelledby="about-principles">
<h2 id="about-principles" class="about-heading">${esc(a.t("about_principles"))}</h2>
<ul class="about-principles">
${principles}
</ul>
</section>
<section class="card about-details" aria-labelledby="about-details">
<h2 id="about-details">${esc(a.t("about_details"))}</h2>
<dl>
<div><dt>${esc(a.t("about_publisher"))}</dt><dd><a href="${about.PUBLISHER_URL}" ${NEW_TAB}>${esc(about.PUBLISHER)}${newTab(a)}</a></dd></div>
<div><dt>${esc(a.t("about_developer"))}</dt><dd><a href="${about.DEVELOPER_URL}" ${NEW_TAB}>${esc(about.DEVELOPER)}${newTab(a)}</a></dd></div>
${version}
<div><dt>${esc(a.t("about_license"))}</dt><dd><a href="${a.href("terms")}">Elastic License 2.0</a></dd></div>
<div class="about-wide"><dt>${esc(a.t("about_source"))}</dt><dd><a href="${a.repoUrl}" ${NEW_TAB} class="ltr" dir="ltr">${esc(a.t("app_name"))}${newTab(a)}</a></dd></div>
</dl>
<nav class="about-links" aria-label="${esc(a.t("about_related"))}">
${links}
</nav>
</section>
<div class="doc-cta">
<a class="btn btn-edge" href="${a.href("contact")}">${a.svg(L.MessagesSquare, 17)}${esc(a.t("about_contact_cta"))}</a>
${support}
<a class="btn btn-ghost" href="${a.repoUrl}" ${NEW_TAB}>GitHub${a.svg(L.ExternalLink, 15)}${newTab(a)}</a>
</div>`;
  return {
    title: ctx.t("about.title"),
    description: ctx.t("about.description"),
    body: appPage(body),
  };
}

export function renderContact(ctx) {
  const a = ctx.appx;
  const cards = a.data.contact.CHANNELS.map((c) =>
    channelCard(a, {
      icon: c.icon,
      accent: c.accent,
      name: a.t(c.name),
      handle: c.handle,
      body: a.t(c.best),
      href: c.href,
      copy: c.href.startsWith("mailto:") ? c.href.slice(7) : c.href,
    }),
  ).join("\n");
  const body = `${docHero({ eyebrow: a.t("contact_eyebrow"), title: a.t("contact_title"), intro: a.t("contact_intro") })}
<ul class="contact-grid">
${cards}
</ul>
${banner(a, "warn", a.lucide.ShieldAlert, a.t("contact_safety"))}`;
  return {
    title: ctx.t("contact.title"),
    description: ctx.t("contact.description"),
    body: appPage(body) + toaster(a),
  };
}

export function renderSupport(ctx) {
  const a = ctx.appx;
  const methods = a.data.donate.donateMethods();
  const list = methods.length
    ? `<ul class="contact-grid">
${methods
  .map((m) =>
    channelCard(a, {
      icon: m.icon,
      accent: m.accent,
      name: a.t(m.name),
      handle: m.handle,
      body: a.t(m.how),
      href: m.href,
      copy: m.copy,
    }),
  )
  .join("\n")}
</ul>`
    : banner(a, "info", a.lucide.HeartHandshake, a.t("support_none"));
  const body = `${docHero({ eyebrow: a.t("support_eyebrow"), title: a.t("support_title"), intro: a.t("support_intro") })}
${banner(a, "warn", a.lucide.ShieldAlert, a.t("support_safety"))}
${list}
<p class="secondary">${esc(a.t("support_thanks"))}</p>`;
  return {
    title: ctx.t("support.title"),
    description: ctx.t("support.description"),
    body: appPage(body) + toaster(a),
  };
}

export function renderFaq(ctx) {
  const a = ctx.appx;
  let index = 0;
  const parts = a.data.faq
    .map((section) => {
      const items = section.items
        .map((item) => {
          index += 1;
          const side = index % 2 === 1 ? "end" : "start";
          return `<div class="faq-item" id="q-${item.id}" data-side="${side}" data-accent="${section.accent}">
<span class="faq-node" aria-hidden="true"></span>
<article class="faq-card" aria-labelledby="faq-q-${item.id}">
<div class="faq-q">
<span class="faq-idx num">${esc(a.t("faq_q", { n: a.fmt.number(index) }))}</span>
<h3 id="faq-q-${item.id}">${esc(a.t(item.q))}</h3>
</div>
<p class="faq-a">${richText(a.t(item.a), a.href)}</p>
</article>
</div>`;
        })
        .join("\n");
      return `<h2 class="faq-seg" data-accent="${section.accent}" id="faq-${section.id}">
<span class="faq-seg-pill"><span class="faq-seg-icon">${a.svg(section.icon, 16)}</span>${esc(a.t(section.title))}</span>
</h2>
${items}`;
    })
    .join("\n");
  const body = `${docHero({ eyebrow: a.t("faq_eyebrow"), title: a.t("faq_title"), intro: a.t("faq_intro") })}
<div class="faq-timeline">
${parts}
</div>
<div class="doc-cta">
<a class="btn btn-edge" href="${a.href("contact")}">${a.svg(a.lucide.MessagesSquare, 16)}${esc(a.t("faq_more"))}</a>
</div>`;
  return { title: ctx.t("faq.title"), description: ctx.t("faq.description"), body: appPage(body) };
}

function legalPage(ctx, kind, extra = []) {
  const a = ctx.appx;
  const doc = kind === "privacy" ? a.data.legal.PRIVACY : a.data.legal.TERMS;
  const prefix = `${kind}-`;
  const sections = [
    ...doc.sections.map((s) => ({
      id: s.id,
      title: a.t(s.title),
      body: s.body.map((p) => a.t(p)),
      list: s.list?.map((li) => a.t(li)),
      quote: s.quote,
    })),
    ...extra,
  ];
  const toc = sections.map((s) => `<a href="#${prefix}${s.id}">${esc(s.title)}</a>`).join("\n");
  const article = sections
    .map((s) => {
      const quote = s.quote
        ? `<blockquote>${esc(a.t(s.quote))}</blockquote>${ctx.lang !== "en" ? `\n<blockquote lang="en" dir="ltr">${esc(a.en(s.quote))}</blockquote>` : ""}`
        : "";
      const contact =
        s.id === "contact"
          ? `<a class="btn btn-edge btn-sm" href="${a.href("contact")}">${a.svg(a.lucide.MessagesSquare, 14)}${esc(a.t("nav_contact"))}</a>`
          : "";
      return `<section id="${prefix}${s.id}" aria-labelledby="${prefix}${s.id}-h">
<h2 id="${prefix}${s.id}-h" style="font-size: 1.2rem; font-weight: bolder">${esc(s.title)}</h2>
${s.body.map((p) => `<p>${esc(p)}</p>`).join("\n")}
${s.list ? `<ul>${s.list.map((li) => `<li>${esc(li)}</li>`).join("")}</ul>` : ""}
${quote}${contact}
</section>`;
    })
    .join("\n");
  const updated = `\n<p class="legal-updated">${esc(a.t("legal_updated", { date: a.fmt.date(doc.updated) }))}</p>`;
  const body = `${docHero({ title: a.t(`${kind}_title`), intro: a.t(doc.intro), after: updated })}
<div class="legal-layout">
<nav class="legal-toc" aria-label="${esc(a.t("legal_toc"))}">
<span class="legal-toc-title">${esc(a.t("legal_toc"))}</span>
${toc}
</nav>
<article class="card legal-body">
${article}
</article>
</div>`;
  return appPage(body, "legal");
}

export function renderPrivacy(ctx) {
  const site = { id: "website", title: ctx.t("privacy.site_title"), body: [ctx.t("privacy.site_body")] };
  return {
    title: ctx.t("privacy.title"),
    description: ctx.t("privacy.description"),
    body: legalPage(ctx, "privacy", [site]),
  };
}

export function renderTerms(ctx) {
  return {
    title: ctx.t("terms.title"),
    description: ctx.t("terms.description"),
    body: legalPage(ctx, "terms"),
  };
}
