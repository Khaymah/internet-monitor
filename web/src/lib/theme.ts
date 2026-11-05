import { useSyncExternalStore } from "react";

export type Theme = "dark" | "light";
const KEY = "im.theme";
const listeners = new Set<() => void>();

function read(): Theme {
  if (typeof document === "undefined") return "dark";
  return document.documentElement.getAttribute("data-theme") === "light" ? "light" : "dark";
}

const THEME_COLOR: Record<Theme, string> = { dark: "#0c1015", light: "#f3f5f9" };

export function setTheme(theme: Theme) {
  document.documentElement.setAttribute("data-theme", theme);
  document.querySelector('meta[name="theme-color"]')?.setAttribute("content", THEME_COLOR[theme]);
  try {
    localStorage.setItem(KEY, theme);
  } catch {}
  for (const fn of listeners) fn();
}

export function useTheme(): Theme {
  return useSyncExternalStore(
    (fn) => {
      listeners.add(fn);
      return () => {
        listeners.delete(fn);
      };
    },
    read,
    () => "dark",
  );
}
