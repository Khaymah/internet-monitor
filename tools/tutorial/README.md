# Settings tour video

`build.cjs` turns `scenes.json` into the narrated settings tour:

- `docs/tutorial/internet-monitor-tutorial.mp4`: H.264 + AAC, 1280×720, 15 fps, `+faststart`.
  The release job uploads it as a release asset; the app streams it from GitHub only when
  someone presses play on the Settings tour page (`/tutorial`).
- `web/public/tutorial/{en,ar,fr,tr,ur}.vtt`: subtitles, served by the app itself.
- `web/public/tutorial/poster.jpg`: the still shown before playing.

## What it does

1. Speaks each scene's English `text.en` with Piper (cached per text in the work folder).
   `say` in `scenes.json` respells words the voice would read badly (for example `ntfy`).
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

1. Build the dashboard and the headless app, then start it with a data folder whose
   `config.json` has quiet hours on and the router status on, so every setting is visible,
   and open to the network so the phone card shows (its QR code and address are blurred in
   the video):

   ```bash
   cd web && bun install --frozen-lockfile && bun run build && cd ..
   go build -ldflags="-X main.Version=v2.0.4" -o /tmp/imh ./cmd/headless
   /tmp/imh --dir /tmp/im-tour-data --listen 0.0.0.0
   ```

2. Copy the `http://127.0.0.1:8765/?session=…` link it prints and run:

   ```bash
   node tools/tutorial/build.cjs \
     --url "http://127.0.0.1:8765/?session=…" \
     --voice /path/to/en_US-lessac-medium.onnx \
     --piper piper
   ```

3. Rebuild the web UI (`bun run build`) so the app embeds the new subtitles and poster.

Options: `--work <dir>` (cache, default a temp folder), `--video <file>`, `--subtitles <dir>`,
`--scenes <file>`, and `--only audio` (print each scene's length) or `--only subtitles`
(rewrite the `.vtt` files after editing translations, without new screenshots). `PIPER`,
`PIPER_VOICE` and `IM_TUTORIAL_URL` work in place of the flags.

## Editing scenes

Each scene in `scenes.json` has:

- `id`, `caption` (the pill in the corner) and `nav` (the section to mark in the Settings list).
- `path`: the page to open.
- `targets`: what to outline. A CSS selector string; `{ "field": "Label" }` for the field or
  switch row whose label reads exactly that (English UI); or `{ "css": "…", "text": "…",
  "index": 0 }` for the innermost element matching the selector that contains the text.
- `scroll` (optional): indexes into `targets` to center instead of all of them.
- `setup` (optional): steps before the still, `{ "click": target }`, `{ "fill": target,
  "value": "10" }`, `{ "wait": 400 }` or `{ "css": "…" }`.
- `text`: the narration in `en` and the same words as subtitles in `ar`, `fr`, `tr` and `ur`.
  Arabic follows the Egyptian register of `web/locales/ar`; quote setting names the way the
  app shows them in that language.

Keep the whole tour between 5 and 8 minutes (`--only audio` prints the total) and the video
under 15 MB.
