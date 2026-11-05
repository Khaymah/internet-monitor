import { readFileSync } from "node:fs";

const KEEP =
  /\.(doc-|faq-|legal-|about-|contact-|cl-|card\b|icon-well|btn\b|btn-|banner|chip\b|tag\b|text-link|toasts?\b|num\b|secondary\b|muted\b|ltr\b|spin\b)|::view-transition|\.theme-switching/;
const DROP = /\.(topbar|sidebar|savebar|settings|dialog|bell|ss-|notice|matrix|whats-new|report|complaint|phone-qr|table|gate|hero|nav-item|tutorial|about-nav)/;
const KEYFRAMES = new Set(["rise", "spin"]);

function blocks(css) {
  const out = [];
  let i = 0;
  while (i < css.length) {
    const open = css.indexOf("{", i);
    if (open < 0) break;
    const prelude = css.slice(i, open).trim();
    let depth = 1;
    let j = open + 1;
    while (j < css.length && depth > 0) {
      if (css[j] === "{") depth++;
      else if (css[j] === "}") depth--;
      j++;
    }
    out.push({ prelude, body: css.slice(open + 1, j - 1) });
    i = j;
  }
  return out;
}

function pick(css) {
  const kept = [];
  for (const { prelude, body } of blocks(css)) {
    if (prelude.startsWith("@keyframes")) {
      if (KEYFRAMES.has(prelude.split(/\s+/)[1])) kept.push(`${prelude} {${body}}`);
      continue;
    }
    if (prelude.startsWith("@media")) {
      if (/print|pointer/.test(prelude)) continue;
      const inner = pick(body);
      if (inner) kept.push(`${prelude} {\n${inner}\n}`);
      continue;
    }
    if (prelude.startsWith("@")) continue;
    const selectors = prelude
      .split(",")
      .map((s) => s.trim())
      .filter((s) => KEEP.test(s) && !DROP.test(s));
    if (selectors.length) kept.push(`${selectors.join(",\n")} {${body}}`);
  }
  return kept.join("\n\n");
}

export function appCss(file) {
  const css = readFileSync(file, "utf8").replace(/\/\*[^]*?\*\//g, "");
  return pick(css);
}
