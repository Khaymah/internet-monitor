# The website

The public website lives at <https://khaymah.github.io/internet-monitor/>. It's a static
site for people who aren't technical: what the app does, how to download and install it,
a tutorial video, a beginner's guide, the FAQ, the privacy policy and what's new. It comes
in English, Arabic, French, Turkish and Urdu (Arabic and Urdu right to left), in a light
and a dark theme, with the same design tokens as the app.

## One-time setup (repository owner)

Open the repository's **Settings → Pages** and set **Source** to **GitHub Actions**.
That's all. The first deploy runs on the next release, push or manual run.

## How it updates itself

`.github/workflows/pages.yml` builds and deploys the site:

- when a release is published, edited or marked as a prerelease,
- on every push to `master` that touches `site/`, `docs/`, `web/locales/`,
  `web/public/tutorial/`, the FAQ or privacy content, the design tokens,
  `CHANGELOG.md` or `PRIVACY.md`,
- by hand from the Actions tab (**Website → Run workflow**).

Releases made by the `Auto Release` job use the built-in `GITHUB_TOKEN`, and GitHub
doesn't start other workflows for events made with that token. So the release job starts
the website workflow itself right after it publishes the release.

Pull requests don't deploy anything: the `Website (build / links)` job in `build.yml`
builds the site and checks every internal link, so a broken site fails CI.

## Where the content comes from

| What | Source | Read at |
| ---- | ------ | ------- |
| Site texts (home, download, guide…) | `site/locales/{en,ar,fr,tr,ur}.json` | build |
| FAQ | `web/locales/<lang>/faq.json`, grouped like `web/src/content/faq.ts` | build |
| Privacy policy | `web/locales/<lang>/legal.json` (privacy section), date from `web/src/content/legal.ts` | build |
| Language names, direction, date format | `web/locales/<lang>/meta.json` | build |
| What's new | `docs/releases/v*.md` (newest first) and the latest section of `CHANGELOG.md` | build |
| Versions, dates, file sizes, download links | GitHub REST API: latest stable release and newest prerelease | build |
| Tutorial video | `internet-monitor-tutorial.mp4` on the latest release (or the newest prerelease that has it) | build |
| Subtitles | `web/public/tutorial/<lang>.vtt`, copied when present | build |
| Screenshots | `docs/store/<lang>/*.png`, English when a language has none | build |
| Colors, radii, gradients | `web/src/styles/tokens.css` | build |
| Fonts | `@fontsource/poppins` and `@fontsource/ibm-plex-sans-arabic`, self-hosted | build |

The FAQ and privacy policy are the app's own texts, so the site never says something
different from the app. If the GitHub API can't be reached (for example a local build
without network), the build still succeeds: download buttons point to the latest release
page and the video shows "coming soon".

The five locale files must have the same keys in the same order, with lists of the same
length. The build stops with an error if they don't.

## Build it locally

You need [bun](https://bun.sh).

```bash
cd site && bun install --frozen-lockfile && cd ..
bun site/build.mjs          # writes site/dist
bun site/check-links.mjs    # checks every internal link, anchor and image alt
bun site/serve.mjs          # http://localhost:4173/internet-monitor/
```

Environment variables for the build:

| Variable | Default | Meaning |
| -------- | ------- | ------- |
| `GITHUB_TOKEN` | none | Used for the GitHub API when set (higher rate limit) |
| `SITE_OFFLINE` | none | `1` skips the API and uses the fallback links |
| `SITE_BASE` | `/internet-monitor/` | Path the site is served under |
| `SITE_ORIGIN` | `https://khaymah.github.io` | Used for canonical links, Open Graph and the sitemap |
| `SITE_REPO` | `Khaymah/internet-monitor` | Repository for releases and links |

## Pages

Each language has its own folder (`/en/`, `/ar/`, `/fr/`, `/tr/`, `/ur/`) with `hreflang`
links between them:

- `/` home, `/download/`, `/guide/`, `/tutorial/`, `/faq/`, `/privacy/`, `/whats-new/`.

The root `index.html` sends visitors to their browser's language (or the one they picked
before), with English as the fallback, and lists all five languages. There's also a
`404.html`, `sitemap.xml` and `robots.txt`. The site loads no outside scripts, fonts or
trackers and sets no cookies; it only remembers the theme and language in the browser.

## Adding or changing text

- Edit the same key in all five `site/locales/*.json` files. Text in `**double stars**`
  is shown in bold, and `{placeholders}` are filled at build time.
- FAQ and privacy changes go in `web/locales/`, as for the app.
- Release notes for a prerelease go in `docs/releases/<tag>.md`, as before.
