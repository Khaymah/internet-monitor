import { existsSync, readFileSync, statSync } from "node:fs";
import { createServer } from "node:http";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const DIST = fileURLToPath(new URL("./dist/", import.meta.url));
const BASE = `/${(process.env.SITE_BASE ?? "/internet-monitor/").replace(/^\/+|\/+$/g, "")}/`.replace(/^\/\/$/, "/");
const PORT = Number(process.env.PORT || 4173);

const TYPES = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".png": "image/png",
  ".webp": "image/webp",
  ".woff2": "font/woff2",
  ".xml": "application/xml",
  ".txt": "text/plain; charset=utf-8",
  ".vtt": "text/vtt; charset=utf-8",
};

export function resolve(urlPath) {
  const path = decodeURIComponent(urlPath.split(/[?#]/)[0]);
  if (!path.startsWith(BASE)) return null;
  const rel = normalize(path.slice(BASE.length)).replace(/^(\.\.[/\\])+/, "");
  let file = join(DIST, rel);
  if (existsSync(file) && statSync(file).isDirectory()) file = join(file, "index.html");
  return existsSync(file) ? file : null;
}

if (process.argv[1]?.endsWith("serve.mjs")) {
  createServer((req, res) => {
    if (req.url === "/") {
      res.writeHead(302, { Location: BASE }).end();
      return;
    }
    const file = resolve(req.url);
    if (!file) {
      res.writeHead(404, { "Content-Type": TYPES[".html"] }).end(readFileSync(join(DIST, "404.html")));
      return;
    }
    res.writeHead(200, { "Content-Type": TYPES[extname(file)] ?? "application/octet-stream" }).end(readFileSync(file));
  }).listen(PORT, () => console.log(`serving site/dist at http://localhost:${PORT}${BASE}`));
}
