import { existsSync, readdirSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { join } from "node:path";
import { esc } from "./util.mjs";

const LUCIDE_IMPORT = /from\s+"lucide-react"/g;

export const SITE_ROUTES = {
  changelog: "changelog",
  faq: "faq",
  tutorial: "tutorial",
  privacy: "privacy",
  terms: "terms",
  about: "about",
  contact: "contact",
  support: "support",
};

function flatten(tree, out, prefix = "") {
  for (const [name, value] of Object.entries(tree)) {
    if (value && typeof value === "object") flatten(value, out, `${prefix}${name}.`);
    else out[prefix + name] = value;
  }
  return out;
}

export async function loadApp(root, siteDir) {
  const req = createRequire(join(siteDir, "package.json"));
  const lucidePath = req.resolve("lucide-react");
  const contentDir = join(root, "web", "src", "content");
  Bun.plugin({
    name: "app-content",
    setup(build) {
      build.onLoad({ filter: /web[\\/]src[\\/]content[\\/][^\\/]+\.ts$/ }, (args) => ({
        loader: "ts",
        contents: readFileSync(args.path, "utf8").replace(LUCIDE_IMPORT, `from ${JSON.stringify(lucidePath)}`),
      }));
    },
  });
  const load = (name) => import(join(contentDir, `${name}.ts`));
  const [about, contact, donate, faq, legal, changelog, kinds, view, text, react, server, lucide] = await Promise.all([
    load("about"),
    load("contact"),
    load("donate"),
    load("faq"),
    load("legal"),
    load("changelog"),
    load("changeKinds"),
    load("changelogView"),
    load("text"),
    import(req.resolve("react")),
    import(req.resolve("react-dom/server")),
    import(lucidePath),
  ]);

  const localesDir = join(root, "web", "locales");
  const messages = {};
  for (const lang of readdirSync(localesDir)) {
    const dir = join(localesDir, lang);
    if (!existsSync(join(dir, "meta.json"))) continue;
    const out = {};
    for (const file of readdirSync(dir).filter((f) => f.endsWith(".json") && f !== "meta.json")) {
      flatten(JSON.parse(readFileSync(join(dir, file), "utf8")), out);
    }
    messages[lang] = out;
  }

  function translator(lang) {
    const own = messages[lang] ?? {};
    const fallback = messages.en ?? {};
    return (key, vars) => {
      const template = own[key] ?? fallback[key];
      if (typeof template !== "string") throw new Error(`app string ${key} is missing`);
      if (!vars) return template;
      return template.replace(/\{(\w+)\}/g, (m, name) => (Object.hasOwn(vars, name) ? String(vars[name]) : m));
    };
  }

  function svg(component, size = 16, cls = "") {
    const html = server.renderToStaticMarkup(
      react.createElement(component, { size, "aria-hidden": true, className: cls || undefined }),
    );
    return html.replace("<svg ", '<svg focusable="false" ');
  }

  return {
    about,
    contact,
    donate,
    faq: faq.FAQ,
    legal,
    releases: changelog.RELEASES,
    kinds: changelog.KINDS,
    kindStyle: kinds.CHANGE_KIND,
    buildTimeline: view.buildTimeline,
    nextMonth: view.NEXT_MONTH,
    repoUrl: text.REPO_URL,
    lucide,
    svg,
    translator,
  };
}

export function richText(text, href) {
  const LINK = /\[([^\]]+)\]\(([a-z]+)(?:#([a-z0-9-]+))?\)/g;
  let out = "";
  let last = 0;
  for (const m of text.matchAll(LINK)) {
    out += esc(text.slice(last, m.index));
    const target = SITE_ROUTES[m[2]] ? href(SITE_ROUTES[m[2]], m[3]) : null;
    out += target ? `<a class="text-link" href="${esc(target)}">${esc(m[1])}</a>` : `<span class="app-ref">${esc(m[1])}</span>`;
    last = m.index + m[0].length;
  }
  return out + esc(text.slice(last));
}

export function appFormat(intl) {
  const dateFmt = new Intl.DateTimeFormat(intl, { year: "numeric", month: "long", day: "numeric" });
  const monthFmt = new Intl.DateTimeFormat(intl, { month: "long", year: "numeric" });
  const numFmt = new Intl.NumberFormat(intl, { maximumFractionDigits: 0 });
  const local = (iso) => {
    const [y, m, d] = iso.split("-").map(Number);
    return new Date(y, (m ?? 1) - 1, d ?? 1);
  };
  return {
    date: (iso) => dateFmt.format(local(iso)),
    month: (key) => monthFmt.format(local(`${key}-01`)),
    number: (n) => numFmt.format(n),
  };
}
