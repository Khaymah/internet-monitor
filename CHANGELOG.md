# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

GitHub release notes for each tag are auto-generated from commits; this file is
the curated, human-readable summary.

## [Unreleased]

Target: **2.0.4**. Ships as GitHub prereleases `v2.0.4-rc.N` (not offered by the
in-app updater, which only follows the latest stable release) until the version is final.
What each rc changed, and how to test it, is in `docs/releases/v2.0.4-rc.N.md`.

### Added
- **Search engine verification** (rc.11): the website adds Google, Bing and Yandex
  verification tags from the repository variables of the same names.
- **Website** (rc.9): `site/`, a static GitHub Pages site in five languages (home,
  download, guide, tutorial, FAQ, privacy, what's new), rebuilt by `pages.yml` on every
  release; see `docs/website.md`.
- **Settings tour video** (rc.9): a narrated walkthrough of every setting at `/tutorial`,
  linked from Settings and the sidebar. The MP4 (`docs/tutorial/`, built by
  `tools/tutorial/`) is a release asset streamed from GitHub only on play; WebVTT
  subtitles in all five languages ship with the app. The CSP allows `media-src` from
  GitHub's download hosts.
- **Router sync speed and drop reason** (rc.8): `GetLinkLayerMaxBitRates` when the
  common link properties report no speed, `last_error` from `GetStatusInfo`, the
  `provider_dropped` outage cause and insight; router actions that fail with an HTTP
  error are not asked again. Checked against a Huawei HG630 V2.
- **Router page reading** (rc.8): optional `router_address`, `router_user` and
  `router_password` (secret). With a password, the Huawei web API
  (`/api/system/user_login`, `/api/ntwk/dslinfo`, `/api/ntwk/wan_st`) is read every 5
  minutes or less for SNR, attenuation, attainable rate, showtime and traffic counters;
  `POST /api/test-router`; `snr_low` insight; line resync counts as `line_down`. Only
  private addresses are contacted, and a rejected password is never retried.
- **Router status** (rc.7): UPnP IGD reading once a minute (`router_status`, on by
  default), `router_DATE.jsonl`, `GET /api/router`, a router card on the overview, and
  `router` on each outage row (`line_down`, `session_down`, `router_restarted`,
  `router_online`).
- **Data plan** (rc.7): `quota_gb` and `quota_reset_day` in the line details, usage from
  the router's counters, alerts at 80% and 100% (`quota` alert kind), and `line_sync`,
  `router_restarts` and `quota_pace` insights.
- **French, Turkish and Urdu** (rc.7).
- **Use it from your phone** (rc.7): a web app manifest and home-screen icons, so "Add to
  Home screen" gives a named Internet Monitor icon. When the dashboard is open to
  the network, Settings → App shows a QR code with the LAN link
  (`GET /api/phone`, `GET /api/phone/qr.png`), and the headless build prints the
  same code in the terminal at startup.
- **Alerts when the power is out (coming soon)** (rc.7): a card in Settings asks whether you
  would use alerts sent from outside your home during a power cut. "I want this"
  opens a GitHub page in your browser; the app sends nothing.
- **Privacy policy** (rc.6): `PRIVACY.md` in English and Arabic, the public
  copy of the in-app Privacy page, which now also covers `case.json`, ntfy,
  log timestamps, the outage traceroute and the Store data folder.
- **Compensation estimate** (rc.5): `monthly_price` and `currency` in the line
  details; the report and the complaint text show the pro-rata value of the
  provider's downtime, and Ofcom automatic-compensation days in the UK.
- **Peak hours** (rc.5): downtime by hour of day, the 7pm–midnight share, and
  evening vs daytime speed in the report.
- **Insights** (rc.5): `GET /api/insights` and a "What your data says" card on
  the overview and the Complaints page.
- **Complaint reminder** (rc.5): a weekly "Time to complain?" alert when the line
  keeps dropping and no complaint is open; WhatsApp and email share buttons.
- **Headless build** (rc.4): `cmd/headless` for Raspberry Pi, Linux servers and
  Docker (`--dir`, `--listen`), linux amd64/arm64/armv7 release binaries, a
  Dockerfile, docker-compose.yml and `docs/headless.md`. Listening on a LAN
  address keeps the access key and enforces same-origin Host/Origin checks.
- **Sealed logs** (rc.4): each finished day is sealed in a SHA-256 hash chain
  (`seals.jsonl`, `GET /api/seals`, "Log integrity" in the report), with
  optional RFC 3161 timestamps (`seal_timestamp`, `tsa_url`) that verify with
  `openssl ts -verify`.
- **Wi-Fi vs cable** (rc.4): `link` and `wifi_signal` on every minute and event,
  shown in the report and the complaint text.
- **Power-cut detection** (rc.4): `on_battery` on outages and boot times in
  `system_DATE.jsonl`; gaps that contain a boot are labeled as restarts.
- **ntfy push notifications** (rc.3): `ntfy_url` and optional `ntfy_token`, a
  new ntfy column in the notification matrix and `POST /api/test-ntfy`.
- **Generic webhooks** (rc.3): `webhook_url` accepts any https address (http
  on the local network); non-Discord/Slack hosts get a plain JSON payload.
- **Bufferbloat grade** (rc.3): the speed test measures ping at idle and under
  load and grades the added delay A+ to F.
- **Route trace on outage** (rc.3): the system traceroute runs when the line
  drops; the report explains where the route stopped, and the traces go into
  the evidence ZIP.
- **Complaints use the measured data** (rc.2): every complaint shows outages,
  downtime, uptime, coverage and speed vs plan for the 7 days before filing and
  since, with a verdict, a ready-to-send complaint text and an evidence ZIP for
  that period (`GET /api/case/evidence`, `GET /api/evidence?from=&to=`). A
  "last 7 days" card suggests logging a complaint when the line dropped.
- **Country-aware escalation** (rc.2): Egypt, Saudi Arabia, UK, US or other sets
  the regulator, the steps and the provider's response window (overridable).
- **Outage timeline** on the overview, 24 hours or 7 days (rc.2, `GET /api/timeline`).
- **Weekly summary** every Monday at 09:00 to the desktop, Discord/Slack and
  Telegram, with a "Send last 7 days now" button (rc.2, `POST /api/weekly/send`).
- Per-rc release notes in `docs/releases/`, used as the GitHub release text.

### Changed
- Microsoft Store publishing is paused (rc.7): the MSIX job runs only when the
  repository variable `STORE_BUILD` is `true`, and releases no longer attach it.
- A prerelease build now checks all releases for updates and is offered the
  next rc or the final version (rc.2). Stable builds still follow stable
  releases only.
- **Complaints page** (`/complaints`): your provider, plan speed and line number,
  plus a log of every complaint (number, date, channel, agent, status). Open
  complaints count down the provider's 72-hour response window and then suggest
  escalating to the regulator, with the Egyptian NTRA steps (155 / My NTRA).
  Stored locally in `case.json`; `GET /api/case`, `POST /api/case/profile`,
  `POST /api/case/complaints`, `DELETE /api/case/complaints/{id}`.
- **Monthly report: monitoring coverage.** Minutes with no checks (computer off
  or asleep, app closed) are listed as "not monitored" instead of being silently
  skipped. A gap that ends straight into an outage is marked as a likely power
  cut, and that outage is flagged as possibly the router restarting.
- **Monthly report: speed tests vs plan** (typical, lowest/highest, share of the
  plan speed, tests under half the plan), the line details, and the month's complaints.
- **Evidence pack** (`GET /api/evidence?month=YYYY-MM`, "Download evidence pack"
  on the report): a ZIP with the report, the raw logs, `case.json` and a
  `SHA256SUMS.txt` so anyone can confirm nothing was edited.
- Settings → Speed test shows how much data each test (and the schedule) uses.
- `docs/MARKET_RESEARCH.ar.md`: market, competitor and growth research behind
  these features.
- `auto_update_check` setting (Settings → App, "Check for updates automatically")
  to turn off the background update check. The manual button keeps working.
- README section listing every server the app contacts, how often, and the
  setting that changes or disables it.

### Fixed
- Router page reading works with routers that serve it only over HTTPS with an old TLS
  setup (Huawei HG630 V2): https first, then http, legacy TLS for the home router only,
  and the test shows why it failed (rc.10).

## [2.0.0] - 2026-10-24

The v2 rebuild.

### Added
- **React dashboard** (React 19 + Vite + TypeScript in `web/`, embedded into the
  binaries via `web/embed.go`) with a live overview: status, plain-language
  diagnosis, latency chart, jitter, packet loss, availability for today / 7 / 30
  days, and per-target status.
- **History screen** with a per-day log, filters, and CSV export for a day, the
  last 7 days, or the last 30 days.
- **Monthly report page** at `/report?month=YYYY-MM` with "Print or save as PDF"
  (replaces `report.html`).
- Speed test screen with gauge, today's runs, scheduled tests and alert threshold.
- Custom alert sound upload (MP3/WAV, up to 30 s / 10 MB); Settings shows whether
  the custom sound is active and its length, with "Try the sound" and "Play like
  a real alert".
- **New default alert sound.**
- Dark and light themes, Arabic/English with full RTL, layout for phone-size windows.
  Switching themes reveals the new one in a circle from the button you clicked
  (instant when the system asks for reduced motion).
- In-app pages: **What's new** (every release since v0.0.1, filterable by type),
  **FAQ**, **About**, **Contact** (WhatsApp, Discord, email, GitHub), **Privacy**
  and **Terms of use**, linked from the sidebar.
- Unsaved-changes guard in Settings.
- **Support us** page (`/support`) listing PayPal, Vodafone Cash and InstaPay donation
  details from `web/src/content/donate.ts`; methods left empty are hidden, and the page
  and its links stay hidden until at least one is filled in.
- **What's new after an update**: the first time the dashboard opens on a new
  version it shows that release's changes once, with a link to the full list.
- FAQ answers link to the exact setting they mention, and the setting is
  scrolled into view and highlighted.
- **Notification controls** (`notifications` in `config.json`): sound on/off and
  volume, quiet hours (mute the sound or the whole desktop alert, per weekday,
  windows across midnight), an on/off switch per event (down, unstable, restored,
  speed test) and per channel (desktop, webhook, Telegram), a minimum outage
  length before alerting, and repeat reminders while offline. Defaults keep the
  previous behaviour.
- **Notification history**: every alert, with where it was sent or why it was
  muted, in `logs/notifications_YYYY-MM-DD.jsonl`; `GET /api/notifications` and
  `POST /api/notifications/read`, plus a live `notification` event.
- Richer Windows toasts: a coloured status icon, the time, an "Open dashboard"
  button, and a new alert replaces the previous one instead of piling up.
- Webhook and Telegram messages follow the app language.
- CI `web` job (typecheck, lint, test, build); Go jobs reuse its `web-dist` artifact.

### Changed
- **License: the Elastic License 2.0 replaces MIT.** The app stays free for
  personal and commercial use and the source stays public; offering it to others
  as a hosted or managed service is not allowed. Versions before 2.0.0 were
  released under MIT.
- The Arabic interface, pages, alerts and Webhook/Telegram messages are now
  written in Egyptian colloquial Arabic instead of Modern Standard Arabic.
- The fixed 4-second desktop alert cooldown is gone; a quick drop-and-return now
  shows both alerts. Use the minimum outage length to hide short blips.
- Repository layout: entry points in `cmd/tray` and `cmd/gui`, packages under
  `internal/`; the old vanilla-JS UI in `internal/server/assets` is removed.
- Building from source now requires [bun](https://bun.sh) (CI pins 1.3.14) in
  addition to Go. `make build`, `build-debug`, `build-gui` and `scripts\build*.cmd`
  build the web UI first.
- Settings saving is conflict-safe.
- The top-bar language switcher is an icon button; its menu lists the languages.
- Latency and jitter are shown in the most readable unit (`46 ms`, `1.06 s`, `2 min 5 s`)
  in the dashboard, chart axis, desktop alerts, tray tooltip and Webhook/Telegram messages.
- Config handling moved to a hardened config store; the Go toolchain is
  upgraded to the version declared in `go.mod`.
- Dev mode: `go run ./cmd/tray` plus `bun run dev` in `web/` for the UI. The
  `.air.toml` config is no longer documented.
- CI refuses to publish a release unless every binary is present.
- Translations moved from one file per language to a folder per language with one
  nested JSON file per page (`web/locales/<lang>/*.json`); changelog strings are
  grouped by version (`changelog.v2.0.0.items.1`).
- Error messages name the actual problem (rate limited, webhook not set up or
  rejected, update already running) instead of a generic failure, and a failed
  webhook test can be retried right away.
- Availability is rounded down, so any downtime never displays as 100%.

### Fixed
- The WhatsApp link on the Contact page now opens from the desktop window.
- **Uploaded custom alert sounds silently fell back to the default.** The
  uploaded file is now validated and converted, and is the sound that plays.
- The auto-updater never selects the GUI asset for a tray install.

### Security
- The local dashboard binds `127.0.0.1` and requires a private session key that
  is kept for up to 24 hours, so an open dashboard reconnects on its own after a
  restart; open it from the tray menu or app window.
- Webhook URL and Telegram token are never sent back to the browser (shown as
  "Saved — hidden").
- Webhook URLs must be `https` and on a Discord or Slack host.
- The updater is pinned to GitHub hosts.

## [0.10.0] - 2026-06-01

### Added
- **Manual "Check for updates" button** in the dashboard header — forces a fresh
  GitHub Releases check on demand instead of waiting for the background poll
  (`POST /api/update/check`).

### Changed
- **CI now mints a version tag and release only when every build passes.** The
  version job computes the next semver as a dry run; the tag and GitHub release
  are created by the release job, which is skipped if any platform build fails.
  A failing push no longer burns a version number.

### Fixed
- Workflow file failed to parse (an unquoted `:` in a step name) — every CI run
  failed at startup with no jobs.
- `gofmt`, cross-platform `tray.Logf` / `sound.Logf` declarations, and unused
  non-Windows tray stubs that broke the Linux `vet` / `golangci-lint` gates.

## [0.9.4] - 2026-06-01

The headline release that landed the Phase 0 + Phase 2 roadmap (PR #2).

### Added
- **Monthly outage report (PDF)** for ISP disputes — month picker, charts,
  chronological event log, per-layer cause breakdown. Neutral by default with an
  opt-in branding footer; handles outages that cross a month boundary. (Closes #3)
- **Upload speed test** with live progress, alongside download and ping.
- **speedtest.net-style circular gauge** — animated needle + arc across the
  ping → download → upload phases, with result cards.
- **Scheduled automatic speed tests** (`speed_test.schedule_minutes`).
- **Telegram notifications** (in addition to Discord / Slack).
- **Localized OS notifications** driven by `config.language` — no more mixed AR/EN.
- Per-target concurrent probing, DNS-vs-HTTP-vs-outage failure classification,
  LAN/ISP diagnosis, optional ICMP ping, latency jitter, and a Prometheus
  `/metrics` endpoint.
- Availability widget, copy-summary, and date-range CSV export.
- Self-update verifies the downloaded binary against the release `SHA256SUMS`.

### Changed
- Live config hot-reload from the dashboard (targets / thresholds / interval /
  webhook) without a restart.
- Unified ringtone playback through a single native backend — the previous sound
  is always stopped first, so test / preview / real-alert chimes never overlap.
- Config is sanitized on load and save; `check_interval_sec=0` can no longer
  panic the ticker.
- README leads with the ISP-dispute use case (EN + AR/RTL).

### Fixed
- Graceful HTTP shutdown; dashboard `ListenAndServe` errors are logged, not
  swallowed.
- CSRF guard, panic recovery, and request-body size limits on the local API.

## [0.8.1] - 2026-05-12

Last release before the 0.9.x → 0.10.0 feature wave. Earlier tags
(0.0.1 – 0.8.1) predate this changelog; see the GitHub Releases page for their
auto-generated notes.

[2.0.0]: https://github.com/Khaymah/internet-monitor/releases/tag/v2.0.0
[0.10.0]: https://github.com/Khaymah/internet-monitor/releases/tag/v0.10.0
[0.9.4]: https://github.com/Khaymah/internet-monitor/releases/tag/v0.9.4
[0.8.1]: https://github.com/Khaymah/internet-monitor/releases/tag/v0.8.1
