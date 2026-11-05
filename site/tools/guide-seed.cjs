const fs = require("node:fs");
const http = require("node:http");
const net = require("node:net");
const path = require("node:path");

const DAY = 24 * 3600 * 1000;
const MIN = 60 * 1000;

const OUTAGES = [
  { daysAgo: 6.4, minutes: 9 },
  { daysAgo: 5.1, minutes: 24 },
  { daysAgo: 3.6, minutes: 6 },
  { daysAgo: 2.3, minutes: 41 },
  { daysAgo: 1.2, minutes: 13 },
  { daysAgo: 0.4, minutes: 28 },
];

const COUNTRY = { ar: "EG", en: "GB", fr: "other", tr: "other", ur: "other" };

function listen(server) {
  return new Promise((resolve) => server.listen(0, "127.0.0.1", () => resolve(server.address().port)));
}

async function startTargets() {
  const tcp = net.createServer((s) => s.destroy());
  const web = http.createServer((_, res) => {
    res.statusCode = 204;
    res.end();
  });
  const tcpPort = await listen(tcp);
  const webPort = await listen(web);
  return { tcpPort, webPort, close: () => [tcp, web].forEach((s) => s.close()) };
}

function dayKey(date) {
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function writeJsonl(dir, prefix, rows) {
  const byDay = new Map();
  for (const row of rows) {
    const key = dayKey(new Date(row.timestamp));
    if (!byDay.has(key)) byDay.set(key, []);
    byDay.get(key).push(JSON.stringify(row));
  }
  for (const [key, lines] of byDay) fs.writeFileSync(path.join(dir, `${prefix}${key}.jsonl`), `${lines.join("\n")}\n`);
}

function outageEvents(windows, now) {
  const events = [];
  let previous = now - 7 * DAY;
  const reason = (down) => ({
    tcp_ping_failed: down,
    http_failed: down,
    dns_failed: down,
    packet_loss_pct: down ? 100 : 0,
    avg_latency_ms: down ? 0 : 31,
  });
  for (const w of windows) {
    events.push({
      timestamp: new Date(w.start).toISOString(),
      event: "disconnected",
      duration_seconds: Math.round((w.start - previous) / 1000),
      reason: reason(true),
    });
    events.push({
      timestamp: new Date(w.end).toISOString(),
      event: "connected",
      duration_seconds: w.minutes * 60,
      reason: reason(false),
    });
    previous = w.end;
  }
  return events;
}

function minuteSamples(windows, now) {
  const inOutage = (t) => windows.some((w) => t >= w.start && t < w.end);
  const samples = [];
  const first = Math.floor((now - 7 * DAY) / MIN) * MIN;
  for (let t = first; t < now - 2 * MIN; t += MIN) {
    const down = inOutage(t);
    const latency = 22 + ((t / MIN) % 17);
    samples.push({
      timestamp: new Date(t).toISOString(),
      samples: 12,
      up_samples: down ? 0 : 12,
      avg_latency_ms: down ? 0 : latency,
      max_latency_ms: down ? 0 : latency + 14,
      jitter_ms: down ? 0 : 4,
      avg_loss_pct: down ? 100 : 0,
      tcp_fails: down ? 12 : 0,
      http_fails: down ? 12 : 0,
      dns_fails: down ? 12 : 0,
    });
  }
  return samples;
}

function speedTests(now) {
  const rows = [];
  for (let i = 0; i < 14; i++) {
    rows.push({
      timestamp: new Date(now - (6.8 - i * 0.48) * DAY).toISOString(),
      event: "speedtest",
      download_mbps: Math.round((36 + ((i * 7) % 11) + (i % 3)) * 10) / 10,
      upload_mbps: Math.round((8.2 + (i % 4) * 0.6) * 10) / 10,
      latency_ms: 24 + (i % 5) * 3,
      duration_seconds: 9,
      endpoints: ["speed.example"],
      parallel_connections: 4,
      triggered_by: "schedule",
    });
  }
  return rows;
}

function seedLogs(logs, now) {
  const windows = OUTAGES.map((o) => {
    const start = now - o.daysAgo * DAY;
    return { start, end: start + o.minutes * MIN, minutes: o.minutes };
  });
  writeJsonl(logs, "connectivity_", outageEvents(windows, now));
  writeJsonl(logs, "metrics_", minuteSamples(windows, now));
  writeJsonl(logs, "speedtest_", speedTests(now));
}

function seedCase(dir, lang, now) {
  const file = {
    profile: {
      country: COUNTRY[lang],
      response_hours: 0,
      provider: "Example Net",
      plan_name: "Home 50",
      plan_download_mbps: 50,
      plan_upload_mbps: 10,
      monthly_price: 0,
      currency: "",
      quota_gb: 0,
      quota_reset_day: 0,
      line_number: "0000 000 000",
      account_number: "1234567",
    },
    complaints: [
      {
        id: "c1a2b3c4d5e6f708",
        provider: "Example Net",
        ticket: "4821907",
        opened_at: new Date(now - 2.2 * DAY).toISOString(),
        channel: "hotline",
        agent: "",
        status: "open",
        notes: "",
      },
    ],
  };
  fs.writeFileSync(path.join(dir, "case.json"), JSON.stringify(file, null, 2));
}

async function seed(dir, lang) {
  fs.rmSync(dir, { recursive: true, force: true });
  const logs = path.join(dir, "logs");
  fs.mkdirSync(logs, { recursive: true });
  const targets = await startTargets();
  const now = Date.now();
  const config = {
    check_interval_sec: 5,
    ping_targets: [`127.0.0.1:${targets.tcpPort}`],
    http_targets: [`http://127.0.0.1:${targets.webPort}/generate_204`],
    dns_targets: ["localhost"],
    use_icmp: false,
    auto_update_check: false,
    router_status: false,
    language: lang,
    speed_test: { schedule_minutes: 0 },
  };
  fs.writeFileSync(path.join(dir, "config.json"), JSON.stringify(config, null, 2));
  seedLogs(logs, now);
  seedCase(dir, lang, now);
  return targets;
}

module.exports = { seed };
