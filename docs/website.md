# The website

The public website lives at <https://khaymah.github.io/internet-monitor/>. It's a static
site for people who aren't technical: what the app does, how to download and install it,
a video tour of each app page, a beginner's guide, and the app's own pages: FAQ, what's new, about,
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
| Video tours | `web/src/content/tours.json` (order and length); for each, `internet-monitor-tour-<id>.mp4` on the newest release that has it (for `settings`, also the older `internet-monitor-tutorial.mp4`) | build |
| Tour titles | `web/locales/<lang>/tour.json` | build |
| Subtitles and posters | `web/public/tutorial/<id>/<lang>.vtt` and `poster.jpg`, copied when present | build |
| Screenshots | `docs/store/<lang>/*.png`, English when a language has none | build |
| Colors, radii, gradients | `web/src/styles/tokens.css` | build |
| Fonts | `@fontsource/poppins` and `@fontsource/ibm-plex-sans-arabic`, self-hosted | build |

These pages are the app's own texts and data, so the site never says something different
from the app. Links inside the app's texts that point to app screens the site doesn't have
(Settings, History…) are shown as plain text. If the GitHub API can't be reached (for example a local build
without network), the build still succeeds: download buttons point to the latest release
page and each video shows "coming soon".

The five locale files must have the same keys in the same order, with lists of the same
length. The build stops with an error if they don't.

## Build it locally

You need [bun](https://bun.sh).

```bash
cd site && bun install --frozen-lockfile && cd ..
bun site/build.mjs          # writes site/dist
bun site/check-links.mjs    # checks links, anchors, alt texts, meta tags and JSON-LD
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
| `GOOGLE_SITE_VERIFICATION` | none | Google Search Console HTML-tag code, added to every page |
| `BING_SITE_VERIFICATION` | none | Bing Webmaster Tools code (`msvalidate.01`), added to every page |
| `YANDEX_VERIFICATION` | none | Yandex Webmaster code, added to every page |

## Pages

Each language has its own folder (`/en/`, `/ar/`, `/fr/`, `/tr/`, `/ur/`) with `hreflang`
links between them:

- `/` home, `/download/`, `/guide/`, `/tutorial/`, `/faq/`, `/whats-new/`, `/about/`,
  `/contact/`, `/support/`, `/privacy/`, `/terms/`.

The header has the main pages and a "More pages" menu; the footer and the phone menu
list the same group as the app's sidebar (About, What's new, FAQ, Videos, Contact, Privacy,
Terms of use, Support us). Switching the theme plays the app's circular reveal, and skips
it when the system asks for reduced motion.

The root `index.html` sends visitors to their browser's language (or the one they picked
before), with English as the fallback, and lists all five languages. There's also a
`404.html`, `sitemap.xml`, `robots.txt`, `llms.txt` and `llms-full.txt`. The site loads no outside scripts, fonts or
trackers and sets no cookies; it only remembers the theme and language in the browser.

## Adding or changing text

- Edit the same key in all five `site/locales/*.json` files. Text in `**double stars**`
  is shown in bold, and `{placeholders}` are filled at build time.
- FAQ, about, contact, support, privacy and terms changes go in `web/locales/` and
  `web/src/content/`, as for the app.
- Release notes for a prerelease go in `docs/releases/<tag>.md`, as before.

## SEO and search engines

Every page has its own title and description (from `site/locales`), keywords
(`seo.keywords` for the whole site plus `seo.page_keywords.<page>`), canonical and
`hreflang` links, Open Graph and Twitter cards, and JSON-LD: `WebSite`, the publisher
and developer, a breadcrumb on inner pages, `SoftwareApplication` on home and download,
`FAQPage` from the app's FAQ, `HowTo` on the guide and the install steps, one `VideoObject`
per published tour on the video page, and `AboutPage` / `ContactPage`. Each page starts with a short direct
answer (`seo.answer.<page>`) and a visible "last updated" date taken from the last
commit that touched its content (or the release / policy date). `robots.txt` allows the
main search and AI crawlers, `sitemap.xml` has a `lastmod` and the language alternates
for every page, and `llms.txt` / `llms-full.txt` give answer engines a plain-text summary.
`bun site/check-links.mjs` fails the build when a page loses one of these.

The share images (1200×630, one per language) are in `site/assets/og/`. After changing
the tagline or the screenshots, rebuild them (needs Playwright with Chromium):

```bash
bun site/build.mjs && node site/tools/og-images.cjs
```

### Google Search Console

1. Open [Google Search Console](https://search.google.com/search-console), **Add property**,
   choose **URL prefix** and enter `https://khaymah.github.io/internet-monitor/`.
2. Pick the **HTML tag** method and copy only the `content` value of the tag it shows.
3. In the repository, open **Settings → Secrets and variables → Actions → Variables** and
   add a variable named `GOOGLE_SITE_VERIFICATION` with that value.
4. Run the website again: **Actions → Website → Run workflow**. When it finishes, press
   **Verify** in Search Console.
5. In Search Console, open **Sitemaps** and submit `sitemap.xml`.

### Bing Webmaster Tools

1. Open [Bing Webmaster Tools](https://www.bing.com/webmasters), add
   `https://khaymah.github.io/internet-monitor/` and choose the **HTML meta tag** method
   (or import the site from Search Console, which needs no code).
2. Save the `content` value as the repository variable `BING_SITE_VERIFICATION`, run the
   website workflow again, then press **Verify**.
3. Submit `https://khaymah.github.io/internet-monitor/sitemap.xml` under **Sitemaps**.

Yandex works the same way with `YANDEX_VERIFICATION`. The tags are on the root page and on
every language page; an empty variable adds nothing.
