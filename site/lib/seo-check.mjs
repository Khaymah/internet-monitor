import { existsSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { verificationMeta } from "./util.mjs";

const REQUIRED = {
  Organization: ["name", "url"],
  Person: ["name"],
  WebSite: ["name", "url", "inLanguage"],
  WebPage: ["name", "url", "inLanguage"],
  FAQPage: ["name", "url", "inLanguage", "mainEntity"],
  AboutPage: ["name", "url", "inLanguage"],
  ContactPage: ["name", "url", "inLanguage"],
  SoftwareApplication: ["name", "applicationCategory", "operatingSystem", "offers", "downloadUrl"],
  HowTo: ["name", "step"],
  VideoObject: ["name", "description", "thumbnailUrl", "uploadDate", "contentUrl"],
  BreadcrumbList: ["itemListElement"],
};

const PAGE_NEEDS = {
  "": ["WebSite", "WebPage", "SoftwareApplication"],
  "download/": ["SoftwareApplication", "HowTo", "BreadcrumbList"],
  "guide/": ["HowTo", "BreadcrumbList"],
  "faq/": ["FAQPage", "BreadcrumbList"],
  "about/": ["AboutPage", "BreadcrumbList"],
  "contact/": ["ContactPage", "BreadcrumbList"],
};

const META = [
  ["name", "description"],
  ["name", "keywords"],
  ["name", "robots"],
  ["name", "author"],
  ["name", "application-name"],
  ["name", "theme-color"],
  ["property", "og:type"],
  ["property", "og:site_name"],
  ["property", "og:title"],
  ["property", "og:description"],
  ["property", "og:url"],
  ["property", "og:locale"],
  ["property", "og:image"],
  ["property", "og:image:width"],
  ["property", "og:image:height"],
  ["property", "og:image:alt"],
  ["name", "twitter:card"],
  ["name", "twitter:image"],
];

function meta(html, attr, name) {
  const re = new RegExp(`<meta ${attr}="${name.replace(/[.:]/g, "\\$&")}" content="([^"]*)"`);
  return html.match(re)?.[1];
}

function nodes(value, out = []) {
  if (Array.isArray(value)) for (const v of value) nodes(v, out);
  else if (value && typeof value === "object") {
    if (value["@type"]) out.push(value);
    for (const v of Object.values(value)) nodes(v, out);
  }
  return out;
}

function checkNode(node, where, problems) {
  for (const field of REQUIRED[node["@type"]] ?? []) {
    const v = node[field];
    if (v === undefined || v === "" || (Array.isArray(v) && v.length === 0)) problems.push(`${where}: ${node["@type"]} without ${field}`);
  }
  if (node["@type"] === "Question" && !node.acceptedAnswer?.text) problems.push(`${where}: Question without an answer`);
  if (node["@type"] === "HowToStep" && !node.text) problems.push(`${where}: HowToStep without text`);
  if (node["@type"] === "ListItem" && !(node.position && node.name && node.item)) problems.push(`${where}: incomplete ListItem`);
  if (node["@type"] === "Offer" && (node.price === undefined || !node.priceCurrency)) problems.push(`${where}: incomplete Offer`);
}

export function checkSeo(dist, files, base) {
  const problems = [];
  const stats = { pages: 0, ldBlocks: 0, ldNodes: 0, types: {} };
  for (const f of files) {
    const html = readFileSync(f, "utf8");
    const where = relative(dist, f);
    const notFound = where === "404.html";
    stats.pages++;
    const title = html.match(/<title>([^<]*)<\/title>/)?.[1];
    if (!title) problems.push(`${where}: no <title>`);
    else if ([...title.replace(/&[a-z#0-9]+;/g, "x")].length > 60) problems.push(`${where}: title longer than 60 characters`);
    const desc = meta(html, "name", "description");
    if (desc && [...desc.replace(/&[a-z#0-9]+;/g, "x")].length > 155) problems.push(`${where}: description longer than 155 characters`);
    for (const [attr, name] of META) {
      if (notFound && name.startsWith("og:")) continue;
      if (!meta(html, attr, name)) problems.push(`${where}: missing <meta ${attr}="${name}">`);
    }
    const kw = (meta(html, "name", "keywords") ?? "").split(",").filter((k) => k.trim());
    if (!notFound && (kw.length < 8 || kw.length > 15)) problems.push(`${where}: ${kw.length} keywords (want 8 to 15)`);
    const robots = meta(html, "name", "robots") ?? "";
    if (notFound ? !robots.includes("noindex") : !robots.startsWith("index,follow")) problems.push(`${where}: wrong robots meta "${robots}"`);
    if (!notFound && !html.includes('hreflang="x-default"')) problems.push(`${where}: no x-default hreflang`);
    const h1 = (html.match(/<h1[\s>]/g) ?? []).length;
    if (h1 !== 1) problems.push(`${where}: ${h1} <h1> elements`);
    for (const img of html.matchAll(/<img\b[^>]*>/g)) {
      if (!/\bwidth="/.test(img[0]) || !/\bheight="/.test(img[0])) problems.push(`${where}: image without width/height`);
    }
    if (where === "index.html" || /^[a-z]{2}\/index\.html$/.test(where)) {
      for (const tag of verificationMeta().split("\n").filter(Boolean)) {
        if (!html.includes(tag)) problems.push(`${where}: verification tag missing: ${tag}`);
      }
    }
    const types = new Set();
    for (const m of html.matchAll(/<script type="application\/ld\+json">([^]*?)<\/script>/g)) {
      stats.ldBlocks++;
      let data;
      try {
        data = JSON.parse(m[1]);
      } catch (err) {
        problems.push(`${where}: JSON-LD does not parse (${err.message})`);
        continue;
      }
      if (data["@context"] !== "https://schema.org") problems.push(`${where}: JSON-LD without the schema.org context`);
      for (const node of nodes(data)) {
        stats.ldNodes++;
        types.add(node["@type"]);
        stats.types[node["@type"]] = (stats.types[node["@type"]] ?? 0) + 1;
        checkNode(node, where, problems);
      }
    }
    if (!notFound && types.size === 0) problems.push(`${where}: no JSON-LD`);
    const path = where.replace(/^[a-z]{2}\//, "").replace(/index\.html$/, "");
    if (/^[a-z]{2}\//.test(where)) {
      for (const t of PAGE_NEEDS[path] ?? (path ? ["BreadcrumbList"] : [])) {
        if (!types.has(t)) problems.push(`${where}: JSON-LD has no ${t}`);
      }
    }
  }
  const robots = existsSync(join(dist, "robots.txt")) ? readFileSync(join(dist, "robots.txt"), "utf8") : "";
  if (!/^Sitemap: https?:\/\/\S+sitemap\.xml$/m.test(robots)) problems.push("robots.txt: no sitemap line");
  for (const bot of ["Googlebot", "GPTBot", "ClaudeBot", "PerplexityBot"]) {
    if (!robots.includes(`User-agent: ${bot}`)) problems.push(`robots.txt: ${bot} is not listed`);
  }
  const sitemap = existsSync(join(dist, "sitemap.xml")) ? readFileSync(join(dist, "sitemap.xml"), "utf8") : "";
  const urls = (sitemap.match(/<url>/g) ?? []).length;
  const lastmods = (sitemap.match(/<lastmod>\d{4}-\d{2}-\d{2}<\/lastmod>/g) ?? []).length;
  if (!urls || urls !== lastmods) problems.push(`sitemap.xml: ${urls} urls but ${lastmods} lastmod dates`);
  for (const f of ["llms.txt", "llms-full.txt"]) {
    if (!existsSync(join(dist, f)) || !readFileSync(join(dist, f), "utf8").startsWith("# ")) problems.push(`${f}: missing or not llms.txt format`);
  }
  stats.sitemapUrls = urls;
  return { problems, stats, base };
}
