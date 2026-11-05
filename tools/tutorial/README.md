# Video tours

`build.cjs` turns the scene files in `videos/` into narrated video tours, one per app page,
in this order (from `tour.json`): `overview`, `history`, `notifications`, `speed`,
`complaints`, `report`, `settings`. For each video `<id>` it writes:

- `docs/tutorial/<id>.mp4`: H.264 + AAC, 1280×720, 15 fps, `+faststart`. The release job
  uploads every file there as the release asset `internet-monitor-tour-<id>.mp4`; the app
  streams it from GitHub only when someone presses play on the Video tours page
  (`/tutorial?v=<id>`).
- `web/public/tutorial/<id>/{en,ar,fr,tr,ur}.vtt`: subtitles, served by the app itself.
- `web/public/tutorial/<id>/poster.jpg`: the still shown before playing and on the gallery card.
- Its length in `web/src/content/tours.json`, which the app and the website read.

Only the videos you name are rebuilt; the others and their lengths stay as they are.

## What it does

1. Speaks each scene's English `text.en` with Piper (cached per text in the work folder).
   `say` in `tour.json` respells words the voice would read badly (for example `ntfy`).
2. Opens the running dashboard in Chromium at 1280×720, dark theme, English, and takes one
   still per scene: it scrolls the targets into view, dims the rest of the page, outlines
   the targets and adds the caption pill.
3. Shows each still for its narration plus a short pause, with a cross-fade between scenes.
4. Writes the subtitles for every language from the same timings, split into cues of up
   to two lines and timed in proportion to their length.

## Requirements

- Node.js 20 or newer with Playwright and its Chromium (`npm i -g playwright && npx playwright install chromium`).
- ffmpeg and ffprobe 6 or newer on `PATH`.
- Piper TTS 1.x (`pip install piper-tts`) and the voice `en_US-lessac-medium` from
  [rhasspy/piper-voices](https://huggingface.co/rhasspy/piper-voices) (v1.0.0): the `.onnx`
  file with its `.onnx.json` next to it.

## Run it

1. Build the dashboard and the headless app, then start it with a data folder that makes
   every page look lived in:
   - `config.json`: quiet hours and the router status on, a router address and password so
     the router card shows the noise margin, and open to the network so the Settings phone
     card shows (its QR code and address are blurred in the video).
   - A few days of logs with outages that have router causes, notifications that were sent
     and held back, today's speed tests, and an open complaint with a ticket number.

   ```bash
   cd web && bun install --frozen-lockfile && bun run build && cd ..
   go build -ldflags="-X main.Version=v2.0.4" -o /tmp/imh ./cmd/headless
   /tmp/imh --dir /tmp/im-tour-data --listen 0.0.0.0
   ```

2. Copy the `http://127.0.0.1:8765/?session=…` link it prints and run, for all videos:

   ```bash
   node tools/tutorial/build.cjs \
     --url "http://127.0.0.1:8765/?session=…" \
     --voice /path/to/en_US-lessac-medium.onnx \
     --piper piper
   ```

   or only the ones that changed: add their ids at the end, for example `... speed complaints`.

3. Rebuild the web UI (`bun run build`) so the app embeds the new subtitles, posters and lengths.

Notes:

- The `speed` video starts a real speed test and waits for its result (up to two and a
  half minutes); point `speed_test` at a test server you control if you don't want it to
  measure the real line.
- The `complaints` video logs and saves a new complaint in the data folder; start from a
  copy of the data so you can run it again.

Options: `--work <dir>` (cache, default a temp folder), `--videos <dir>`, `--subtitles <dir>`,
`--manifest <file>`, and `--only audio` (print each scene's length) or `--only subtitles`
(rewrite the `.vtt` files after editing translations, without new screenshots). `PIPER`,
`PIPER_VOICE` and `IM_TUTORIAL_URL` work in place of the flags.

## Editing scenes

`videos/<id>.json` has the video's `title` (MP4 metadata), the page `path`, a `ready`
selector to wait for, an optional `settle` time in ms and optional `css` added to every
still, then the `scenes`. Each scene has:

- `id`, `caption` (the pill in the corner), optional `nav` (the section to mark in the
  Settings list) and optional `poster: true` (use this still as the poster).
- `path` (optional): another page to open.
- `targets`: what to outline. A CSS selector string; `{ "field": "Label" }` for the field or
  switch row whose label reads exactly that (English UI); or `{ "css": "…", "text": "…",
  "index": 0, "find": "…" }` for the innermost element matching the selector that contains
  the text, optionally narrowed to the first `find` inside it.
- `scroll` (optional): indexes into `targets` to center instead of all of them.
- `setup` (optional): steps before the still, `{ "click": target }`, `{ "fill": target,
  "value": "10" }`, `{ "hover": target }`, `{ "wait": 400 }`, `{ "waitFor": "selector" }`
  or `{ "css": "…" }`.
- `after` and `focus` (optional): steps run after scrolling to `focus` (for example a hover
  that opens a tooltip); `targets` are then outlined, so they can include what the steps
  opened.
- `keep: true` (optional): continue on the previous scene's page instead of opening it
  again, so state such as a finished speed test or an opened form stays.
- `text`: the narration in `en` and the same words as subtitles in `ar`, `fr`, `tr` and `ur`.
  Arabic follows the Egyptian register of `web/locales/ar`; quote UI names the way the app
  shows them in that language.

Keep each page video between 1 and 3 minutes and the Settings one under 8 (`--only audio`
prints the lengths), and every video under 15 MB.
