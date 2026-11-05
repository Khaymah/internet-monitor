import { esc, inline } from "./util.mjs";

const ITEM = /^(\s*)([-*+]|\d+[.)])\s+(.*)$/;
const TASK = /^\[( |x|X)\]\s+/;

function renderInline(text, link) {
  const task = text.match(TASK);
  if (!task) return inline(text, link);
  const done = task[1].toLowerCase() === "x";
  return `<span class="md-task" aria-hidden="true">${done ? "☑" : "☐"}</span> ${inline(text.slice(task[0].length), link)}`;
}

function renderList(lines, link) {
  const items = [];
  for (const line of lines) {
    const m = line.match(ITEM);
    if (m) items.push({ indent: m[1].length, ordered: /\d/.test(m[2]), text: m[3] });
    else if (line.trim() && items.length) items[items.length - 1].text += ` ${line.trim()}`;
  }
  let html = "";
  const stack = [];
  for (const item of items) {
    while (stack.length && item.indent < stack[stack.length - 1].indent) {
      html += `</li></${stack.pop().tag}>`;
    }
    const top = stack[stack.length - 1];
    if (top && item.indent === top.indent) {
      html += "</li><li>";
    } else {
      const tag = item.ordered ? "ol" : "ul";
      stack.push({ indent: item.indent, tag });
      html += `<${tag}><li>`;
    }
    html += renderInline(item.text, link);
  }
  while (stack.length) html += `</li></${stack.pop().tag}>`;
  return html;
}

function renderTable(lines, link) {
  const cells = (line) =>
    line
      .trim()
      .replace(/^\||\|$/g, "")
      .split("|")
      .map((c) => c.trim());
  const [head, , ...body] = lines;
  const th = cells(head)
    .map((c) => `<th scope="col">${inline(c, link)}</th>`)
    .join("");
  const rows = body.map((r) => `<tr>${cells(r).map((c) => `<td>${inline(c, link)}</td>`).join("")}</tr>`).join("");
  return `<div class="md-table"><table><thead><tr>${th}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

export function markdown(source, { baseLevel = 3, link = (href) => href } = {}) {
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  const levels = lines.map((l) => l.match(/^(#{1,6})\s/)?.[1].length).filter(Boolean);
  const shift = levels.length ? baseLevel - Math.min(...levels) : 0;
  const out = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (!line.trim()) {
      i++;
      continue;
    }
    const fence = line.match(/^\s*```(\w*)/);
    if (fence) {
      const code = [];
      i++;
      while (i < lines.length && !/^\s*```/.test(lines[i])) code.push(lines[i++]);
      i++;
      out.push(`<pre><code>${esc(code.join("\n"))}</code></pre>`);
      continue;
    }
    const heading = line.match(/^(#{1,6})\s+(.*?)\s*#*$/);
    if (heading) {
      const level = Math.min(6, Math.max(2, heading[1].length + shift));
      out.push(`<h${level}>${inline(heading[2], link)}</h${level}>`);
      i++;
      continue;
    }
    if (/^\s*(---|\*\*\*|___)\s*$/.test(line)) {
      out.push("<hr>");
      i++;
      continue;
    }
    if (/^\s*>/.test(line)) {
      const quote = [];
      while (i < lines.length && /^\s*>/.test(lines[i])) quote.push(lines[i++].replace(/^\s*>\s?/, ""));
      out.push(`<blockquote>${markdown(quote.join("\n"), { baseLevel, link })}</blockquote>`);
      continue;
    }
    if (/^\s*\|/.test(line) && /^\s*\|?\s*:?-{3,}/.test(lines[i + 1] ?? "")) {
      const rows = [];
      while (i < lines.length && /^\s*\|/.test(lines[i])) rows.push(lines[i++]);
      out.push(renderTable(rows, link));
      continue;
    }
    if (ITEM.test(line)) {
      const block = [];
      while (i < lines.length) {
        const cur = lines[i];
        if (ITEM.test(cur) || (cur.trim() && /^\s+/.test(cur))) {
          block.push(cur);
          i++;
        } else if (!cur.trim() && (ITEM.test(lines[i + 1] ?? "") || /^\s+\S/.test(lines[i + 1] ?? ""))) {
          i++;
        } else break;
      }
      out.push(renderList(block, link));
      continue;
    }
    const para = [lines[i++].trim()];
    while (
      i < lines.length &&
      lines[i].trim() &&
      !/^(#{1,6}\s|\s*```|\s*>|\s*\|)/.test(lines[i]) &&
      !ITEM.test(lines[i])
    ) {
      para.push(lines[i++].trim());
    }
    out.push(`<p>${inline(para.join(" "), link)}</p>`);
  }
  return out.join("\n");
}
