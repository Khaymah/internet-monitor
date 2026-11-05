const { execFileSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { duration, plan, render, synthesize } = require("./lib/media.cjs");
const { still } = require("./lib/page.cjs");
const { writeSubtitles } = require("./lib/subtitles.cjs");

const ROOT = path.resolve(__dirname, "../..");
const SIZE = { width: 1280, height: 720 };
const TOUR = JSON.parse(fs.readFileSync(path.join(__dirname, "tour.json"), "utf8"));

function parseArgs(argv) {
  const opts = {
    url: process.env.IM_TUTORIAL_URL || "",
    piper: process.env.PIPER || "piper",
    voice: process.env.PIPER_VOICE || "",
    work: path.join(os.tmpdir(), "im-tutorial"),
    videos: path.join(ROOT, "docs/tutorial"),
    subtitles: path.join(ROOT, "web/public/tutorial"),
    manifest: path.join(ROOT, "web/src/content/tours.json"),
    only: "",
    ids: [],
  };
  for (let i = 2; i < argv.length; i++) {
    if (!argv[i].startsWith("--")) {
      opts.ids.push(argv[i]);
      continue;
    }
    const key = argv[i].slice(2);
    if (!(key in opts) || key === "ids") throw new Error(`unknown option --${key}`);
    opts[key] = argv[++i];
  }
  if (opts.ids.length === 0 || opts.ids.includes("all")) opts.ids = [...TOUR.videos];
  for (const id of opts.ids) {
    if (!TOUR.videos.includes(id)) throw new Error(`unknown video ${id}; pick from ${TOUR.videos.join(", ")} or all`);
  }
  if (!opts.url && !opts.only) throw new Error("pass --url with the dashboard link that contains ?session=");
  if (!opts.voice) throw new Error("pass --voice with the Piper .onnx voice file");
  return opts;
}

function loadVideo(id) {
  const video = JSON.parse(fs.readFileSync(path.join(__dirname, "videos", `${id}.json`), "utf8"));
  const seen = new Set();
  for (const scene of video.scenes) {
    if (seen.has(scene.id)) throw new Error(`${id}: scene ${scene.id} appears twice`);
    seen.add(scene.id);
  }
  return video;
}

async function openSession(opts) {
  const { chromium } = require("playwright");
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
  await start.close();
  return { browser, ctx, origin: new URL(opts.url).origin };
}

async function capture(session, opts, id, video) {
  const dir = path.join(opts.work, id, "frames");
  fs.mkdirSync(dir, { recursive: true });
  const frames = [];
  let page = null;
  for (const scene of video.scenes) {
    if (!scene.keep || !page) {
      await page?.close();
      page = await session.ctx.newPage();
      page.on("dialog", (d) => d.accept());
      await page.goto(session.origin + (scene.path ?? video.path));
      await page.waitForSelector(scene.ready ?? video.ready ?? ".page-head");
      await page.waitForTimeout(scene.settle ?? video.settle ?? 1200);
      if (video.css) await page.addStyleTag({ content: video.css });
    }
    const file = path.join(dir, `${scene.id}.png`);
    await still(page, scene, file);
    frames.push(file);
    process.stdout.write(`frame ${id}/${scene.id}\n`);
  }
  await page?.close();
  return frames;
}

function writeManifest(opts, lengths) {
  const old = fs.existsSync(opts.manifest) ? JSON.parse(fs.readFileSync(opts.manifest, "utf8")) : [];
  const known = new Map(old.map((v) => [v.id, v.seconds]));
  const list = TOUR.videos.map((id) => ({ id, seconds: lengths[id] ?? known.get(id) ?? 0 }));
  fs.writeFileSync(opts.manifest, `${JSON.stringify(list, null, 2)}\n`);
}

async function main() {
  const opts = parseArgs(process.argv);
  fs.mkdirSync(opts.work, { recursive: true });
  for (const tool of ["ffmpeg", "ffprobe"]) execFileSync(tool, ["-version"], { stdio: "ignore" });
  const jobs = opts.ids.map((id) => {
    const video = loadVideo(id);
    const audio = synthesize(opts, video, TOUR.say);
    const timing = plan(audio);
    const total = timing[timing.length - 1].start + timing[timing.length - 1].slot;
    process.stdout.write(`${id}: narration ${total.toFixed(1)} s over ${timing.length} scenes\n`);
    if (opts.only === "audio") {
      for (const [i, scene] of video.scenes.entries()) process.stdout.write(`  ${scene.id} ${timing[i].slot.toFixed(1)} s\n`);
    }
    return { id, video, audio, timing };
  });
  if (opts.only === "audio") return;
  for (const job of jobs) writeSubtitles(path.join(opts.subtitles, job.id), job.id, job.video, job.timing);
  if (opts.only === "subtitles") return;

  const session = await openSession(opts);
  const lengths = {};
  try {
    for (const { id, video, audio, timing } of jobs) {
      const frames = await capture(session, opts, id, video);
      const out = {
        video: path.join(opts.videos, `${id}.mp4`),
        poster: path.join(opts.subtitles, id, "poster.jpg"),
        posterFrame: frames[Math.max(0, video.scenes.findIndex((s) => s.poster))],
      };
      render(path.join(opts.work, id), out, frames, audio, timing, video.title);
      lengths[id] = Math.round(duration(out.video) * 10) / 10;
      process.stdout.write(`video ${out.video} (${(fs.statSync(out.video).size / 1048576).toFixed(1)} MB, ${lengths[id]} s)\n`);
    }
  } finally {
    await session.browser.close();
  }
  writeManifest(opts, lengths);
}

main().catch((e) => {
  process.stderr.write(`${e.stack || e}\n`);
  process.exit(1);
});
