# The website

The public website lives at <https://khaymah.github.io/internet-monitor/>. It's a static
site for people who aren't technical: what the app does, how to download and install it,
a tutorial video, a beginner's guide, and the app's own pages: FAQ, what's new, about,
contact, support us, privacy policy and terms of use. It comes in English, Arabic, French,
Turkish and Urdu (Arabic and Urdu right to left), in a light and a dark theme. The pages
that also exist inside the app use the app's own CSS, markup, icons and texts, so they look
and read the same in both places.

## One-time setup (repository owner)

Open the repository's **Settings → Pages** and set **Source** to **GitHub Actions**.
That's all. The first deploy runs on the next release, push or manual run.

## How it updates itself

`.github/workflows/pages.yml` builds and deploys the site:

- when a release is published, edited or marked as a prerelease,
- on every push to `master` that touches `site/`, `docs/`, `web/locales/`,
  `web/public/tutorial/`, the app's page content (`web/src/content/`), its styles,
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
| FAQ, what's new, about, contact, support us, privacy, terms | the app's `web/src/content/*.ts` (imported as modules) and texts in `web/locales/<lang>/*.json` | build |
| Look of those pages, buttons, cards, banners | the matching rules of `web/src/styles/app.css` (picked by `site/lib/app-css.mjs`) | build |
| Icons on those pages | the app's own `lucide-react` icons, rendered to SVG | build |
| Language names, direction, date format | `web/locales/<lang>/meta.json` | build |
| What's new, next release | `docs/releases/v*.md` (newest first, in English) | build |
| Versions, dates, file sizes, download links | GitHub REST API: latest stable release and newest prerelease | build |
| Tutorial video | `internet-monitor-tutorial.mp4` on the latest release (or the newest prerelease that has it) | build |
| Subtitles | `web/public/tutorial/<lang>.vtt`, copied when present | build |
| Screenshots | `docs/store/<lang>/*.png`, English when a language has none | build |
| Colors, radii, gradients | `web/src/styles/tokens.css` | build |
| Fonts | `@fontsource/poppins` and `@fontsource/ibm-plex-sans-arabic`, self-hosted | build |

These pages are the app's own texts and data, so the site never says something different
from the app. Links inside the app's texts that point to app screens the site doesn't have
(Settings, History…) are shown as plain text. If the GitHub API can't be reached (for example a local build
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

- `/` home, `/download/`, `/guide/`, `/tutorial/`, `/faq/`, `/whats-new/`, `/about/`,
  `/contact/`, `/support/`, `/privacy/`, `/terms/`.

The header has the main pages and a "More pages" menu; the footer and the phone menu
list the same group as the app's sidebar (About, What's new, FAQ, Video, Contact, Privacy,
Terms of use, Support us). Switching the theme plays the app's circular reveal, and skips
it when the system asks for reduced motion.

The root `index.html` sends visitors to their browser's language (or the one they picked
before), with English as the fallback, and lists all five languages. There's also a
`404.html`, `sitemap.xml` and `robots.txt`. The site loads no outside scripts, fonts or
trackers and sets no cookies; it only remembers the theme and language in the browser.

## Adding or changing text

- Edit the same key in all five `site/locales/*.json` files. Text in `**double stars**`
  is shown in bold, and `{placeholders}` are filled at build time.
- FAQ, about, contact, support, privacy and terms changes go in `web/locales/` and
  `web/src/content/`, as for the app.
- Release notes for a prerelease go in `docs/releases/<tag>.md`, as before.
