const { execFileSync, spawnSync } = require("node:child_process");
const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const ROOT = path.resolve(__dirname, "../..");
const LANGS = ["en", "ar", "fr", "tr", "ur"];
const FPS = 15;
const FADE = 0.4;
const LEAD = 0.3;
const TAIL = 0.5;
const CUE_CHARS = 90;
const LINE_CHARS = 44;
const SIZE = { width: 1280, height: 720 };

function parseArgs(argv) {
  const opts = {
    url: process.env.IM_TUTORIAL_URL || "",
    piper: process.env.PIPER || "piper",
    voice: process.env.PIPER_VOICE || "",
    work: path.join(os.tmpdir(), "im-tutorial"),
    scenes: path.join(__dirname, "scenes.json"),
    video: path.join(ROOT, "docs/tutorial/internet-monitor-tutorial.mp4"),
    subtitles: path.join(ROOT, "web/public/tutorial"),
    only: "",
  };
  for (let i = 2; i < argv.length; i++) {
    const key = argv[i].replace(/^--/, "");
    if (!(key in opts)) throw new Error(`unknown option --${key}`);
    opts[key] = argv[++i];
  }
  if (!opts.url && !opts.only) throw new Error("pass --url with the dashboard link that contains ?session=");
  if (!opts.voice) throw new Error("pass --voice with the Piper .onnx voice file");
  return opts;
}

function run(cmd, args, input) {
  const r = spawnSync(cmd, args, { input, encoding: "utf8", maxBuffer: 64 << 20 });
  if (r.status !== 0) throw new Error(`${cmd} failed (${r.status}): ${r.stderr || r.error}`);
  return r.stdout;
}

function duration(file) {
  const out = run("ffprobe", ["-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", file]);
  return Number.parseFloat(out.trim());
}

function spoken(text, say) {
  let out = text;
  for (const [word, sound] of Object.entries(say)) {
    out = out.replace(new RegExp(`\\b${word.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`, "g"), sound);
  }
  return out;
}

function synthesize(opts, script) {
  const dir = path.join(opts.work, "audio");
  fs.mkdirSync(dir, { recursive: true });
  const voiceId = path.basename(opts.voice);
  return script.scenes.map((scene) => {
    const line = spoken(scene.text.en, script.say ?? {}).replace(/\s+/g, " ").trim();
    const hash = crypto.createHash("sha256").update(`${voiceId}\n${line}`).digest("hex").slice(0, 16);
    const file = path.join(dir, `${scene.id}-${hash}.wav`);
    if (!fs.existsSync(file)) {
      run(opts.piper, ["-m", opts.voice, "-f", file, "--sentence-silence", "0.3"], `${line}\n`);
    }
    return { file, seconds: duration(file) };
  });
}

function pageScene(specs) {
  const style = document.createElement("style");
  style.textContent =
    "*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important}";
  document.head.appendChild(style);

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
    return inner[spec.index ?? 0] ?? null;
  };

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
    document.body.appendChild(tag);
  }
}

async function capture(opts, script) {
  const { chromium } = require("playwright");
  const dir = path.join(opts.work, "frames");
  fs.mkdirSync(dir, { recursive: true });
  const browser = await chromium.launch();
  const ctx = await browser.newContext({
    viewport: SIZE,
    deviceScaleFactor: 1,
    colorScheme: "dark",
    reducedMotion: "reduce",
  });
  const start = await ctx.newPage();
  await start.goto(opts.url);
  await start.waitForTimeout(800);
  await start.evaluate(async () => {
    localStorage.setItem("im.lang", "en");
    localStorage.setItem("im.theme", "dark");
    await fetch("/api/language?lang=en", { method: "POST" });
  });
  const origin = new URL(opts.url).origin;
  await start.close();
  const frames = [];
  for (const scene of script.scenes) {
    const page = await ctx.newPage();
    page.on("dialog", (d) => d.accept());
    await page.goto(origin + scene.path);
    await page.waitForSelector(scene.ready ?? ".settings-layout");
    await page.waitForTimeout(1200);
    if (script.css) await page.addStyleTag({ content: script.css });
    for (const step of scene.setup ?? []) {
      if (step.css) await page.addStyleTag({ content: step.css });
      if (step.wait) await page.waitForTimeout(step.wait);
      if (step.click || step.fill) {
        const r = await page.evaluate(pageScene, [step.click ?? step.fill]);
        if (r.missing) throw new Error(`${scene.id}: setup target not found ${r.missing}`);
        const el = page.locator('[data-tour="0"]');
        if (step.click) await el.click();
        else await el.locator("input").first().fill(step.value);
        await page.evaluate(() => document.querySelector('[data-tour="0"]')?.removeAttribute("data-tour"));
      }
    }
    const r = await page.evaluate(pageScene, scene.targets);
    if (r.missing) throw new Error(`${scene.id}: target not found ${r.missing}`);
    await page.evaluate(pageFocus, { count: r.ok, scroll: scene.scroll });
    await page.waitForTimeout(500);
    await page.evaluate(pageOverlay, { count: r.ok, caption: scene.caption, nav: scene.nav });
    const file = path.join(dir, `${scene.id}.png`);
    await page.screenshot({ path: file });
    frames.push(file);
    await page.close();
    process.stdout.write(`frame ${scene.id}\n`);
  }
  await browser.close();
  return frames;
}

function plan(audio) {
  let at = 0;
  return audio.map((a) => {
    const slot = Math.ceil((LEAD + a.seconds + TAIL) * FPS) / FPS;
    const s = { start: at, slot, speechStart: at + LEAD, speechEnd: at + LEAD + a.seconds };
    at += slot;
    return s;
  });
}

function render(opts, frames, audio, timing) {
  const work = opts.work;
  const n = frames.length;
  const audioGraph = [];
  for (let i = 0; i < n; i++) {
    audioGraph.push(
      `[${i}:a]aresample=48000,aformat=channel_layouts=mono,adelay=${Math.round(LEAD * 1000)}:all=1,apad=whole_dur=${timing[i].slot}[a${i}]`,
    );
  }
  audioGraph.push(`${timing.map((_, i) => `[a${i}]`).join("")}concat=n=${n}:v=0:a=1,volume=-1.5dB[out]`);
  const audioScript = path.join(work, "audio.filter");
  fs.writeFileSync(audioScript, audioGraph.join(";\n"));
  const narration = path.join(work, "narration.wav");
  run("ffmpeg", [
    "-y",
    "-v",
    "error",
    ...audio.flatMap((a) => ["-i", a.file]),
    "-filter_complex_script",
    audioScript,
    "-map",
    "[out]",
    narration,
  ]);

  const videoGraph = [];
  for (let i = 0; i < n; i++) videoGraph.push(`[${i}:v]fps=${FPS},format=yuv420p,setsar=1[v${i}]`);
  let last = "v0";
  for (let i = 1; i < n; i++) {
    const label = i === n - 1 ? "vout" : `x${i}`;
    videoGraph.push(`[${last}][v${i}]xfade=transition=fade:duration=${FADE}:offset=${timing[i].start.toFixed(4)}[${label}]`);
    last = label;
  }
  const videoScript = path.join(work, "video.filter");
  fs.writeFileSync(videoScript, videoGraph.join(";\n"));
  const inputs = frames.flatMap((f, i) => {
    const len = timing[i].slot + (i === n - 1 ? 0 : FADE);
    return ["-loop", "1", "-framerate", String(FPS), "-t", len.toFixed(4), "-i", f];
  });
  fs.mkdirSync(path.dirname(opts.video), { recursive: true });
  run("ffmpeg", [
    "-y",
    "-v",
    "error",
    ...inputs,
    "-i",
    narration,
    "-filter_complex_script",
    videoScript,
    "-map",
    `[${n === 1 ? "v0" : "vout"}]`,
    "-map",
    `${n}:a`,
    "-c:v",
    "libx264",
    "-preset",
    "slow",
    "-tune",
    "stillimage",
    "-crf",
    "22",
    "-pix_fmt",
    "yuv420p",
    "-r",
    String(FPS),
    "-c:a",
    "aac",
    "-b:a",
    "80k",
    "-ac",
    "1",
    "-movflags",
    "+faststart",
    "-metadata",
    "title=Internet Monitor: a tour of the Settings page",
    opts.video,
  ]);
  const poster = path.join(opts.subtitles, "poster.jpg");
  fs.mkdirSync(opts.subtitles, { recursive: true });
  run("ffmpeg", ["-y", "-v", "error", "-i", frames[0], "-vf", "scale=960:-2", "-q:v", "5", poster]);
}

function sentences(text) {
  return text
    .replace(/[ \t\r\n]+/g, " ")
    .trim()
    .split(/(?<=[.!?؟۔])[ ]+/)
    .filter(Boolean);
}

function breakAt(text, target, maxLeft, maxRight = maxLeft) {
  let best = -1;
  let bestCost = Number.POSITIVE_INFINITY;
  let quoted = 0;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if ("«“".includes(ch)) quoted++;
    if ("»”".includes(ch)) quoted = Math.max(0, quoted - 1);
    if (ch !== " " || i > maxLeft || text.length - i - 1 > maxRight) continue;
    const before = text[i - 1] ?? "";
    let cost = Math.abs(i - target);
    if (/[.!?؟۔]/.test(before)) cost -= 16;
    else if (/[,،;:]/.test(before)) cost -= 8;
    if (quoted > 0) cost += 14;
    if (cost < bestCost) {
      bestCost = cost;
      best = i;
    }
  }
  return best;
}

function splitLong(sentence) {
  const out = [];
  let rest = sentence;
  while (rest.length > CUE_CHARS) {
    const parts = Math.ceil(rest.length / CUE_CHARS);
    const at = breakAt(rest, rest.length / parts, CUE_CHARS, Number.POSITIVE_INFINITY);
    if (at < 0) break;
    out.push(rest.slice(0, at));
    rest = rest.slice(at + 1);
  }
  out.push(rest);
  return out;
}

function cuesOf(text) {
  const cues = [];
  let open = false;
  for (const sentence of sentences(text)) {
    const pieces = splitLong(sentence);
    const prev = cues[cues.length - 1];
    if (pieces.length === 1 && open && prev.length + 1 + sentence.length <= CUE_CHARS) {
      cues[cues.length - 1] = `${prev} ${sentence}`;
      continue;
    }
    cues.push(...pieces);
    open = pieces.length === 1;
  }
  return cues;
}

function twoLines(cue) {
  if (cue.length <= LINE_CHARS) return cue;
  const at = breakAt(cue, cue.length / 2, Math.max(LINE_CHARS + 8, Math.ceil(cue.length / 2)));
  return at < 0 ? cue : `${cue.slice(0, at)}\n${cue.slice(at + 1)}`;
}

function stamp(t) {
  const ms = Math.round(t * 1000);
  const h = Math.floor(ms / 3600000);
  const m = Math.floor((ms % 3600000) / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  const pad = (v, n = 2) => String(v).padStart(n, "0");
  return `${pad(h)}:${pad(m)}:${pad(s)}.${pad(ms % 1000, 3)}`;
}

function subtitles(opts, script, timing) {
  fs.mkdirSync(opts.subtitles, { recursive: true });
  for (const lang of LANGS) {
    const lines = ["WEBVTT", "", `NOTE Internet Monitor settings tour (${lang}). Generated by tools/tutorial/build.cjs.`, ""];
    let n = 0;
    script.scenes.forEach((scene, i) => {
      const text = scene.text[lang];
      if (!text) throw new Error(`${scene.id}: no ${lang} text`);
      const cues = cuesOf(text);
      const total = cues.reduce((sum, c) => sum + c.length, 0);
      const { speechStart } = timing[i];
      const speechEnd = timing[i].start + timing[i].slot - 0.1;
      let at = speechStart;
      for (const cue of cues) {
        const end = at + ((speechEnd - speechStart) * cue.length) / total;
        n += 1;
        lines.push(String(n), `${stamp(at)} --> ${stamp(end)}`, twoLines(cue), "");
        at = end;
      }
    });
    fs.writeFileSync(path.join(opts.subtitles, `${lang}.vtt`), lines.join("\n"));
  }
}

async function main() {
  const opts = parseArgs(process.argv);
  const script = JSON.parse(fs.readFileSync(opts.scenes, "utf8"));
  fs.mkdirSync(opts.work, { recursive: true });
  for (const tool of ["ffmpeg", "ffprobe"]) execFileSync(tool, ["-version"], { stdio: "ignore" });
  const audio = synthesize(opts, script);
  const timing = plan(audio);
  const total = timing[timing.length - 1].start + timing[timing.length - 1].slot;
  process.stdout.write(`narration ${total.toFixed(1)} s over ${timing.length} scenes\n`);
  if (opts.only === "audio") {
    for (const [i, scene] of script.scenes.entries()) process.stdout.write(`${scene.id} ${timing[i].slot.toFixed(1)} s\n`);
    return;
  }
  if (opts.only === "subtitles") {
    subtitles(opts, script, timing);
    return;
  }
  const frames = await capture(opts, script);
  render(opts, frames, audio, timing);
  subtitles(opts, script, timing);
  process.stdout.write(`video ${opts.video} (${(fs.statSync(opts.video).size / 1048576).toFixed(1)} MB)\n`);
}

main().catch((e) => {
  process.stderr.write(`${e.stack || e}\n`);
  process.exit(1);
});
