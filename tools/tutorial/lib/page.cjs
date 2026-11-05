function pageScene(specs) {
  if (!document.getElementById("tour-still")) {
    const style = document.createElement("style");
    style.id = "tour-still";
    style.textContent =
      "*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important}";
    document.head.appendChild(style);
  }

  const resolveOne = (spec) => {
    if (typeof spec === "string") return document.querySelector(spec);
    if (spec.field) {
      const labels = [...document.querySelectorAll(".field-label")].filter(
        (el) => el.textContent.trim() === spec.field,
      );
      const label = labels[spec.index ?? 0];
      return label ? label.closest(".switch-row") || label.closest(".field") : null;
    }
    const all = [...document.querySelectorAll(spec.css)].filter(
      (el) => !spec.text || el.textContent.includes(spec.text),
    );
    const inner = all.filter((el) => !all.some((o) => o !== el && el.contains(o)));
    const el = inner[spec.index ?? 0] ?? null;
    return el && spec.find ? el.querySelector(spec.find) : el;
  };

  for (const el of document.querySelectorAll("[data-tour]")) el.removeAttribute("data-tour");
  const out = [];
  for (const [i, spec] of specs.entries()) {
    const el = resolveOne(spec);
    if (!el) return { missing: JSON.stringify(spec) };
    el.setAttribute("data-tour", String(i));
    out.push(i);
  }
  return { ok: out.length };
}

function pageFocus({ count, scroll }) {
  const els = [...Array(count).keys()].map((i) => document.querySelector(`[data-tour="${i}"]`));
  const scrollEls = scroll ? els.filter((_, i) => scroll.includes(i)) : els;
  const topbar = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--layout-topbar")) || 72;
  const fixed = scrollEls.every((el) => getComputedStyle(el).position === "fixed");
  if (!fixed) {
    const rects = scrollEls.map((el) => el.getBoundingClientRect());
    const top = Math.min(...rects.map((r) => r.top)) + window.scrollY;
    const bottom = Math.max(...rects.map((r) => r.bottom)) + window.scrollY;
    const room = window.innerHeight - topbar - 24;
    const h = bottom - top;
    const y = h <= room ? top - topbar - 12 - (room - h) / 2 : top - topbar - 16;
    window.scrollTo({ top: Math.max(0, y), behavior: "instant" });
  }
}

function pageOverlay({ count, caption, nav }) {
  if (nav) {
    for (const a of document.querySelectorAll(".settings-nav a")) {
      if (a.textContent.trim() === nav) a.setAttribute("aria-current", "true");
      else a.removeAttribute("aria-current");
    }
  }
  const pad = 8;
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  const rects = [...Array(count).keys()].map((i) => {
    const r = document.querySelector(`[data-tour="${i}"]`).getBoundingClientRect();
    return { x: r.left - pad, y: r.top - pad, w: r.width + pad * 2, h: r.height + pad * 2 };
  });
  const groups = [];
  for (const r of rects) {
    const near = groups.find(
      (g) => r.x < g.x + g.w + 24 && g.x < r.x + r.w + 24 && r.y < g.y + g.h + 24 && g.y < r.y + r.h + 24,
    );
    if (near) {
      const x = Math.min(near.x, r.x);
      const y = Math.min(near.y, r.y);
      near.w = Math.max(near.x + near.w, r.x + r.w) - x;
      near.h = Math.max(near.y + near.h, r.y + r.h) - y;
      near.x = x;
      near.y = y;
    } else groups.push({ ...r });
  }
  const clip = groups.map((g) => {
    const x = Math.max(4, g.x);
    const y = Math.max(4, g.y);
    return { x, y, w: Math.min(vw - 4, g.x + g.w) - x, h: Math.min(vh - 4, g.y + g.h) - y };
  });
  const rr = (r, k) =>
    `M${r.x + k} ${r.y}H${r.x + r.w - k}Q${r.x + r.w} ${r.y} ${r.x + r.w} ${r.y + k}V${r.y + r.h - k}Q${r.x + r.w} ${r.y + r.h} ${r.x + r.w - k} ${r.y + r.h}H${r.x + k}Q${r.x} ${r.y + r.h} ${r.x} ${r.y + r.h - k}V${r.y + k}Q${r.x} ${r.y} ${r.x + k} ${r.y}Z`;
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("width", vw);
  svg.setAttribute("height", vh);
  svg.setAttribute(
    "style",
    "position:fixed;inset:0;z-index:2147483646;pointer-events:none;filter:drop-shadow(0 0 10px rgba(110,150,255,.55))",
  );
  const dim = document.createElementNS(ns, "path");
  dim.setAttribute("d", `M0 0H${vw}V${vh}H0Z${clip.map((r) => rr(r, 14)).join("")}`);
  dim.setAttribute("fill", "rgba(3,5,9,0.68)");
  dim.setAttribute("fill-rule", "evenodd");
  svg.appendChild(dim);
  for (const r of clip) {
    const ring = document.createElementNS(ns, "path");
    ring.setAttribute("d", rr(r, 14));
    ring.setAttribute("fill", "none");
    ring.setAttribute("stroke", "#8fb0ff");
    ring.setAttribute("stroke-width", "3");
    svg.appendChild(ring);
  }
  svg.setAttribute("data-tour-overlay", "");
  document.body.appendChild(svg);
  if (caption) {
    const tag = document.createElement("div");
    tag.textContent = caption;
    tag.setAttribute(
      "style",
      "position:fixed;left:20px;bottom:20px;z-index:2147483647;padding:8px 16px;border-radius:999px;" +
        "background:linear-gradient(90deg,#3b6cff,#b14cff);color:#fff;font:600 15px/1.2 Poppins,system-ui,sans-serif;" +
        "letter-spacing:.2px;box-shadow:0 6px 24px rgba(0,0,0,.45)",
    );
    tag.setAttribute("data-tour-overlay", "");
    document.body.appendChild(tag);
  }
}

async function runStep(page, step, where) {
  if (step.css) await page.addStyleTag({ content: step.css });
  if (step.wait) await page.waitForTimeout(step.wait);
  if (step.waitFor) await page.waitForSelector(step.waitFor, { timeout: step.timeout ?? 60000 });
  const target = step.click ?? step.fill ?? step.hover;
  if (!target) return;
  const r = await page.evaluate(pageScene, [target]);
  if (r.missing) throw new Error(`${where}: step target not found ${r.missing}`);
  const el = page.locator('[data-tour="0"]');
  if (step.click) await el.click();
  else if (step.hover) await el.hover(step.at ? { position: step.at } : undefined);
  else await el.locator("input, textarea").first().fill(step.value);
  await page.evaluate(() => document.querySelector('[data-tour="0"]')?.removeAttribute("data-tour"));
}

async function still(page, scene, file) {
  await page.evaluate(() => {
    for (const el of document.querySelectorAll("[data-tour-overlay]")) el.remove();
  });
  for (const step of scene.setup ?? []) await runStep(page, step, scene.id);
  let r = await page.evaluate(pageScene, scene.focus ?? scene.targets);
  if (r.missing) throw new Error(`${scene.id}: target not found ${r.missing}`);
  await page.evaluate(pageFocus, { count: r.ok, scroll: scene.scroll });
  await page.waitForTimeout(500);
  if (scene.after) {
    for (const step of scene.after) await runStep(page, step, scene.id);
    r = await page.evaluate(pageScene, scene.targets);
    if (r.missing) throw new Error(`${scene.id}: target not found after its steps ${r.missing}`);
    await page.waitForTimeout(300);
  }
  await page.evaluate(pageOverlay, { count: r.ok, caption: scene.caption, nav: scene.nav });
  await page.screenshot({ path: file });
}

module.exports = { still };
