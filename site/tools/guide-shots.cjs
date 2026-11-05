const { spawn, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { seed } = require("./guide-seed.cjs");

const SITE = path.join(__dirname, "..");
const ROOT = path.join(SITE, "..");
const LANGS = ["en", "ar", "fr", "tr", "ur"];
const THEMES = ["light", "dark"];
const WIDTH = 900;
const QUALITY = 0.82;
const VIEW = { width: 1280, height: 2000 };

const RECENT = 'section[aria-labelledby="c-recent"]';

const SHOTS = [
  {
    id: "status",
    path: "/",
    ready: "#hero-status",
    targets: ['section[aria-labelledby="hero-status"]'],
    padding: 24,
  },
  {
    id: "details",
    path: "/complaints",
    ready: "#c-profile",
    targets: ['section[aria-labelledby="c-profile"]'],
    padding: 20,
  },
  {
    id: "trouble",
    path: "/complaints",
    ready: "#c-recent",
    targets: ['.nav a[href="/complaints"]', `${RECENT} .banner`],
    frame: [RECENT],
    padding: 20,
  },
  {
    id: "send",
    path: "/complaints",
    ready: "#c-recent",
    open: `${RECENT} .card-head button`,
    wait: "textarea.textarea",
    targets: [`${RECENT} textarea ~ .row > button`],
    frame: [`${RECENT} .field-label`, `${RECENT} textarea`],
    padding: 20,
  },
];

function parseArgs(argv) {
  const opts = {
    bin: "",
    out: path.join(SITE, "assets", "guide"),
    work: path.join(os.tmpdir(), "im-guide-shots"),
    only: "",
  };
  for (let i = 2; i < argv.length; i++) {
    const key = argv[i].replace(/^--/, "");
    if (!(key in opts)) throw new Error(`unknown option --${key}`);
    opts[key] = argv[++i];
  }
  return opts;
}

function buildBinary(opts) {
  if (opts.bin) return opts.bin;
  fs.mkdirSync(opts.work, { recursive: true });
  const bin = path.join(opts.work, "imh");
  const r = spawnSync("go", ["build", "-o", bin, "./cmd/headless"], { cwd: ROOT, encoding: "utf8" });
  if (r.status !== 0) throw new Error(`go build failed: ${r.stderr}`);
  return bin;
}

function startApp(bin, dir) {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, ["--dir", dir, "--listen", "127.0.0.1"], { stdio: ["ignore", "pipe", "pipe"] });
    let text = "";
    const timer = setTimeout(() => reject(new Error(`app did not print a link:\n${text}`)), 20000);
    child.stdout.on("data", (chunk) => {
      text += chunk;
      const m = text.match(/http:\/\/127\.0\.0\.1:\d+\/\?session=\S+/);
      if (m) {
        clearTimeout(timer);
        resolve({ child, url: m[0] });
      }
    });
    child.on("error", reject);
  });
}

function stopApp(child) {
  return new Promise((resolve) => {
    child.on("exit", resolve);
    child.kill("SIGTERM");
    setTimeout(() => child.kill("SIGKILL"), 5000);
  });
}

function pageMark(groups) {
  return groups.map((selectors, g) => {
    let count = 0;
    for (const sel of selectors) {
      for (const el of document.querySelectorAll(sel)) {
        el.setAttribute("data-shot", String(g));
        count++;
      }
    }
    return count;
  });
}

function pageOverlay({ theme, framePad }) {
  const style = document.createElement("style");
  style.textContent =
    "*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important}";
  document.head.appendChild(style);
  const vw = window.innerWidth;
  const vh = window.innerHeight;
  const rect = (el) => el.getBoundingClientRect();
  const marked = [...document.querySelectorAll('[data-shot="0"]')].map(rect);
  const frames = [...document.querySelectorAll('[data-shot="1"]')].map(rect);
  const holes = marked.map((r) => ({ x: r.left - 6, y: r.top - 6, w: r.width + 12, h: r.height + 12 }));
  const all = [...marked, ...frames];
  const box = {
    x: Math.max(0, Math.min(...all.map((r) => r.left)) - framePad),
    y: Math.max(0, Math.min(...all.map((r) => r.top)) - framePad),
  };
  box.w = Math.min(vw, Math.max(...all.map((r) => r.right)) + framePad) - box.x;
  box.h = Math.min(vh, Math.max(...all.map((r) => r.bottom)) + framePad) - box.y;
  const rr = (r, k) =>
    `M${r.x + k} ${r.y}H${r.x + r.w - k}Q${r.x + r.w} ${r.y} ${r.x + r.w} ${r.y + k}V${r.y + r.h - k}Q${r.x + r.w} ${r.y + r.h} ${r.x + r.w - k} ${r.y + r.h}H${r.x + k}Q${r.x} ${r.y + r.h} ${r.x} ${r.y + r.h - k}V${r.y + k}Q${r.x} ${r.y} ${r.x + k} ${r.y}Z`;
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("width", vw);
  svg.setAttribute("height", vh);
  const ring = theme === "dark" ? "#8fb0ff" : "#2f5bd6";
  svg.setAttribute(
    "style",
    `position:fixed;inset:0;z-index:2147483646;pointer-events:none;filter:drop-shadow(0 0 8px ${ring}88)`,
  );
  const dim = document.createElementNS(ns, "path");
  dim.setAttribute("d", `M0 0H${vw}V${vh}H0Z${holes.map((r) => rr(r, 12)).join("")}`);
  dim.setAttribute("fill", theme === "dark" ? "rgba(3,5,9,0.62)" : "rgba(18,26,44,0.5)");
  dim.setAttribute("fill-rule", "evenodd");
  svg.appendChild(dim);
  for (const r of holes) {
    const line = document.createElementNS(ns, "path");
    line.setAttribute("d", rr(r, 12));
    line.setAttribute("fill", "none");
    line.setAttribute("stroke", ring);
    line.setAttribute("stroke-width", "3");
    svg.appendChild(line);
  }
  document.body.appendChild(svg);
  return box;
}

async function toWebp(page, png) {
  const url = `data:image/png;base64,${png.toString("base64")}`;
  const data = await page.evaluate(
    async ({ url, width, quality }) => {
      const img = new Image();
      img.src = url;
      await img.decode();
      const scale = Math.min(1, width / img.naturalWidth);
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(img.naturalWidth * scale);
      canvas.height = Math.round(img.naturalHeight * scale);
      const g = canvas.getContext("2d");
      g.imageSmoothingQuality = "high";
      g.drawImage(img, 0, 0, canvas.width, canvas.height);
      return { uri: canvas.toDataURL("image/webp", quality), w: canvas.width, h: canvas.height };
    },
    { url, width: WIDTH, quality: QUALITY },
  );
  return { buffer: Buffer.from(data.uri.split(",")[1], "base64"), w: data.w, h: data.h };
}

async function shoot(ctx, tool, origin, label, theme, shot, outDir) {
  const page = await ctx.newPage();
  await page.goto(origin + shot.path);
  await page.waitForSelector(shot.ready);
  await page.waitForTimeout(1500);
  if (shot.open) {
    await page.locator(shot.open).first().click();
    await page.waitForSelector(shot.wait);
    await page.waitForTimeout(600);
  }
  const counts = await page.evaluate(pageMark, [shot.targets, shot.frame ?? []]);
  if (counts[0] === 0) throw new Error(`${label}/${shot.id}: targets not found`);
  const box = await page.evaluate(pageOverlay, { theme, framePad: shot.padding ?? 16 });
  const png = await page.screenshot({ clip: { x: box.x, y: box.y, width: box.w, height: box.h } });
  const webp = await toWebp(tool, png);
  fs.writeFileSync(path.join(outDir, `${shot.id}-${theme}.webp`), webp.buffer);
  await page.close();
  console.log(`${label}/${shot.id}-${theme}: ${webp.w}x${webp.h}, ${(webp.buffer.length / 1024).toFixed(0)} KB`);
  return { w: webp.w, h: webp.h };
}

async function prepare(browser, app, lang, theme) {
  const ctx = await browser.newContext({
    viewport: VIEW,
    deviceScaleFactor: 2,
    colorScheme: theme,
    reducedMotion: "reduce",
    locale: lang,
  });
  await ctx.addInitScript(
    ([l, t]) => {
      localStorage.setItem("im.lang", l);
      localStorage.setItem("im.theme", t);
    },
    [lang, theme],
  );
  const start = await ctx.newPage();
  await start.goto(app.url);
  await start.waitForTimeout(800);
  await start.evaluate((l) => fetch(`/api/language?lang=${l}`, { method: "POST" }), lang);
  await start.close();
  return ctx;
}

async function runLang(browser, tool, bin, opts, lang, manifest) {
  const targets = await seed(path.join(opts.work, `data-${lang}`), lang);
  const app = await startApp(bin, path.join(opts.work, `data-${lang}`));
  const origin = new URL(app.url).origin;
  const outDir = path.join(opts.out, lang);
  fs.mkdirSync(outDir, { recursive: true });
  try {
    for (const theme of THEMES) {
      const ctx = await prepare(browser, app, lang, theme);
      for (const shot of SHOTS) {
        manifest[`${lang}/${shot.id}-${theme}`] = await shoot(ctx, tool, origin, lang, theme, shot, outDir);
      }
      await ctx.close();
    }
  } finally {
    await stopApp(app.child);
    targets.close();
  }
}

(async () => {
  const opts = parseArgs(process.argv);
  const bin = buildBinary(opts);
  const { chromium } = require("playwright");
  const browser = await chromium.launch();
  const tool = await browser.newPage();
  fs.mkdirSync(opts.out, { recursive: true });
  const manifestFile = path.join(opts.out, "manifest.json");
  const manifest = fs.existsSync(manifestFile) ? JSON.parse(fs.readFileSync(manifestFile, "utf8")) : {};
  for (const lang of LANGS.filter((l) => !opts.only || l === opts.only)) {
    await runLang(browser, tool, bin, opts, lang, manifest);
  }
  const sorted = Object.fromEntries(Object.entries(manifest).sort(([a], [b]) => a.localeCompare(b)));
  fs.writeFileSync(manifestFile, `${JSON.stringify(sorted, null, 1)}\n`);
  await browser.close();
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
