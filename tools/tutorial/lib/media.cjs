const { spawnSync } = require("node:child_process");
const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const FPS = 15;
const FADE = 0.4;
const LEAD = 0.3;
const TAIL = 0.5;

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

function synthesize(opts, video, say) {
  const dir = path.join(opts.work, "audio");
  fs.mkdirSync(dir, { recursive: true });
  const voiceId = path.basename(opts.voice);
  return video.scenes.map((scene) => {
    const line = spoken(scene.text.en, say).replace(/\s+/g, " ").trim();
    const hash = crypto.createHash("sha256").update(`${voiceId}\n${line}`).digest("hex").slice(0, 16);
    const file = path.join(dir, `${scene.id}-${hash}.wav`);
    if (!fs.existsSync(file)) {
      run(opts.piper, ["-m", opts.voice, "-f", file, "--sentence-silence", "0.3"], `${line}\n`);
    }
    return { file, seconds: duration(file) };
  });
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

function render(work, out, frames, audio, timing, title) {
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
  fs.mkdirSync(path.dirname(out.video), { recursive: true });
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
    `title=${title}`,
    out.video,
  ]);
  fs.mkdirSync(path.dirname(out.poster), { recursive: true });
  run("ffmpeg", ["-y", "-v", "error", "-i", out.posterFrame ?? frames[0], "-vf", "scale=960:-2", "-q:v", "5", out.poster]);
}

module.exports = { duration, plan, render, synthesize };
