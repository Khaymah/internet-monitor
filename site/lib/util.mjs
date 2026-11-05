import { mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

export function esc(value) {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function inline(text, linkHref = (href) => href) {
  const codes = [];
  let out = esc(text).replace(/`([^`]+)`/g, (_, code) => {
    codes.push(`<code>${code}</code>`);
    return `\u0000${codes.length - 1}\u0000`;
  });
  out = out
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|[\s(])\*([^*\s][^*]*)\*(?=[\s).,;:!?]|$)/g, "$1<em>$2</em>")
    .replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, href) => {
      const target = linkHref(href.replace(/&amp;/g, "&"));
      if (!target) return `<strong>${label}</strong>`;
      const external = /^https?:/.test(target);
      return `<a href="${esc(target)}"${external ? ' rel="noopener"' : ""}>${label}</a>`;
    });
  return out.replace(/\u0000(\d+)\u0000/g, (_, i) => codes[Number(i)]);
}

export function fill(template, vars = {}) {
  return String(template).replace(/\{(\w+)\}/g, (m, key) => (key in vars ? `⁨${vars[key]}⁩` : m));
}

export function lookup(obj, key) {
  return key.split(".").reduce((node, part) => (node == null ? undefined : node[part]), obj);
}

export function shape(value) {
  if (Array.isArray(value)) return `[${value.map(shape).join(",")}]`;
  if (value && typeof value === "object") {
    return `{${Object.entries(value)
      .map(([k, v]) => `${k}:${shape(v)}`)
      .join(",")}}`;
  }
  return typeof value;
}

export function formatDate(intl, iso) {
  if (!iso) return "";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat(intl, {
    year: "numeric",
    month: "long",
    day: "numeric",
    timeZone: "UTC",
  }).format(date);
}

export function formatSize(intl, bytes) {
  if (!bytes) return "";
  const mb = bytes / (1024 * 1024);
  const n = new Intl.NumberFormat(intl, { maximumFractionDigits: mb < 10 ? 1 : 0 }).format(mb);
  return `${n} MB`;
}

export function write(file, content) {
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, content);
}

export function compareVersions(a, b) {
  const parse = (v) => {
    const [core, pre = ""] = v.replace(/^v/, "").split("-");
    const nums = core.split(".").map(Number);
    const preNum = pre ? Number(pre.replace(/\D+/g, "")) || 0 : Number.POSITIVE_INFINITY;
    return [...nums, preNum];
  };
  const pa = parse(a);
  const pb = parse(b);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const x = pa[i] ?? 0;
    const y = pb[i] ?? 0;
    if (x !== y) return x > y ? 1 : -1;
  }
  return 0;
}

const VERIFY = [
  ["google-site-verification", "GOOGLE_SITE_VERIFICATION"],
  ["msvalidate.01", "BING_SITE_VERIFICATION"],
  ["yandex-verification", "YANDEX_VERIFICATION"],
];

export function verificationMeta(env = process.env) {
  return VERIFY.map(([name, key]) => [name, String(env[key] ?? "").trim()])
    .filter(([, value]) => /^[A-Za-z0-9_\-.=]{6,200}$/.test(value))
    .map(([name, value]) => `<meta name="${name}" content="${value}">`)
    .join("\n");
}
