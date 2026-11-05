function shapeRelease(r) {
  const assets = {};
  for (const a of r.assets ?? []) {
    assets[a.name] = { name: a.name, size: a.size, url: a.browser_download_url };
  }
  return {
    tag: r.tag_name,
    name: r.name || r.tag_name,
    date: r.published_at || r.created_at,
    url: r.html_url,
    prerelease: Boolean(r.prerelease),
    assets,
  };
}

async function fetchJson(url, token) {
  const headers = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    "User-Agent": "internet-monitor-site",
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), 15000);
  try {
    const res = await fetch(url, { headers, signal: ctrl.signal });
    if (!res.ok) throw new Error(`${url} answered ${res.status}`);
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}

export async function loadReleases(repo, { token, offline } = {}) {
  const fallback = {
    ok: false,
    stable: null,
    preview: null,
    list: [],
    page: `https://github.com/${repo}/releases/latest`,
    all: `https://github.com/${repo}/releases`,
  };
  if (offline) return fallback;
  let list;
  try {
    list = await fetchJson(`https://api.github.com/repos/${repo}/releases?per_page=40`, token);
  } catch (err) {
    console.warn(`releases: falling back to static links (${err.message})`);
    return fallback;
  }
  if (!Array.isArray(list)) return fallback;
  const releases = list
    .filter((r) => !r.draft)
    .map(shapeRelease)
    .sort((a, b) => new Date(b.date) - new Date(a.date));
  const stable = releases.find((r) => !r.prerelease) ?? null;
  const newestPre = releases.find((r) => r.prerelease) ?? null;
  const preview = newestPre && (!stable || new Date(newestPre.date) > new Date(stable.date)) ? newestPre : null;

  return { ...fallback, ok: Boolean(stable || preview), stable, preview, list: releases };
}

export const PLATFORMS = [
  {
    id: "windows",
    icon: "windows",
    stepIcons: ["download", "pointer", "clock", "gauge"],
    editions: {
      tray: [{ asset: "internet-monitor-windows.exe" }],
      window: [{ asset: "internet-monitor-gui-windows.exe" }],
    },
  },
  {
    id: "mac",
    icon: "apple",
    stepIcons: ["download", "shield", "pointer", "gauge"],
    editions: {
      tray: [
        { arch: "arm", asset: "internet-monitor-macos-arm64" },
        { arch: "intel", asset: "internet-monitor-macos-intel" },
      ],
      window: [
        { arch: "arm", asset: "internet-monitor-gui-macos-arm64" },
        { arch: "intel", asset: "internet-monitor-gui-macos-intel" },
      ],
    },
  },
  {
    id: "linux",
    icon: "linux",
    stepIcons: ["download", "shield", "pointer", "gauge"],
    editions: {
      tray: [{ asset: "internet-monitor-linux" }],
      window: [{ asset: "internet-monitor-gui-linux" }],
    },
  },
];

export const PI_FILES = [
  { asset: "internet-monitor-headless-linux-arm64", label: "pi_64" },
  { asset: "internet-monitor-headless-linux-armv7", label: "pi_32" },
  { asset: "internet-monitor-headless-linux-amd64", label: "pi_amd64" },
];

export function resolveAsset(releases, name) {
  const fromStable = releases.stable?.assets[name];
  if (fromStable) return { ...fromStable, release: releases.stable, preview: false };
  const fromPreview = releases.preview?.assets[name];
  if (fromPreview) return { ...fromPreview, release: releases.preview, preview: true };
  return { name, url: releases.page, release: null, preview: false };
}
