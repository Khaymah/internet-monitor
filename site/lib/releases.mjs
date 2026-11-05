export const TUTORIAL_ASSET = "internet-monitor-tutorial.mp4";

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
    tutorial: null,
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

  let tutorial = null;
  if (stable?.assets[TUTORIAL_ASSET]) {
    tutorial = {
      url: `https://github.com/${repo}/releases/latest/download/${TUTORIAL_ASSET}`,
      release: stable,
    };
  } else {
    const pre = releases.find((r) => r.prerelease && r.assets[TUTORIAL_ASSET]);
    if (pre) tutorial = { url: pre.assets[TUTORIAL_ASSET].url, release: pre };
  }

  return { ...fallback, ok: Boolean(stable || preview), stable, preview, tutorial };
}

export const PLATFORMS = [
  {
    id: "windows",
    icon: "windows",
    groups: [
      {
        files: [
          { asset: "internet-monitor-windows.exe", label: "tray", desc: "tray_desc", main: true },
          { asset: "internet-monitor-gui-windows.exe", label: "window", desc: "window_desc" },
        ],
      },
    ],
  },
  {
    id: "mac",
    icon: "apple",
    note: "mac_which",
    groups: [
      {
        title: "mac_arm",
        arch: "arm",
        files: [
          { asset: "internet-monitor-macos-arm64", label: "tray", desc: "tray_desc", main: true },
          { asset: "internet-monitor-gui-macos-arm64", label: "window", desc: "window_desc" },
        ],
      },
      {
        title: "mac_intel",
        arch: "intel",
        files: [
          { asset: "internet-monitor-macos-intel", label: "tray", desc: "tray_desc", main: true },
          { asset: "internet-monitor-gui-macos-intel", label: "window", desc: "window_desc" },
        ],
      },
    ],
  },
  {
    id: "linux",
    icon: "linux",
    groups: [
      {
        files: [
          { asset: "internet-monitor-linux", label: "tray", desc: "tray_desc", main: true },
          { asset: "internet-monitor-gui-linux", label: "window", desc: "window_desc" },
        ],
      },
    ],
  },
  {
    id: "pi",
    icon: "pi",
    groups: [
      {
        files: [
          { asset: "internet-monitor-headless-linux-arm64", label: "pi_64", main: true },
          { asset: "internet-monitor-headless-linux-armv7", label: "pi_32" },
          { asset: "internet-monitor-headless-linux-amd64", label: "pi_amd64" },
        ],
      },
    ],
  },
];

export function resolveAsset(releases, name) {
  const fromStable = releases.stable?.assets[name];
  if (fromStable) return { ...fromStable, release: releases.stable, preview: false };
  const fromPreview = releases.preview?.assets[name];
  if (fromPreview) return { ...fromPreview, release: releases.preview, preview: true };
  return { name, url: releases.page, release: null, preview: false };
}
