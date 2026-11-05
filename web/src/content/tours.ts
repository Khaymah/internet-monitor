import type { MessageKey } from "../lib/i18n";
import type { Route } from "../lib/router";
import manifest from "./tours.json";

const RELEASES = "https://github.com/Khaymah/internet-monitor/releases";
const RELEASE_TAG = /^v\d+\.\d+\.\d+(-rc\.\d+)?$/;

export type TourId =
  | "overview"
  | "history"
  | "notifications"
  | "speed"
  | "complaints"
  | "report"
  | "settings";

export interface Tour {
  id: TourId;
  route: Route;
  seconds: number;
  title: MessageKey;
  desc: MessageKey;
  page: MessageKey;
}

const PAGES: Record<TourId, Omit<Tour, "id" | "seconds">> = {
  overview: {
    route: "overview",
    page: "nav_overview",
    title: "tour_overview_title",
    desc: "tour_overview_desc",
  },
  history: { route: "history", page: "nav_history", title: "tour_history_title", desc: "tour_history_desc" },
  notifications: {
    route: "notifications",
    page: "nav_notifications",
    title: "tour_notifications_title",
    desc: "tour_notifications_desc",
  },
  speed: { route: "speed", page: "nav_speed", title: "tour_speed_title", desc: "tour_speed_desc" },
  complaints: {
    route: "complaints",
    page: "nav_complaints",
    title: "tour_complaints_title",
    desc: "tour_complaints_desc",
  },
  report: { route: "report", page: "report_open", title: "tour_report_title", desc: "tour_report_desc" },
  settings: {
    route: "settings",
    page: "nav_settings",
    title: "tour_settings_title",
    desc: "tour_settings_desc",
  },
};

export const TOURS: Tour[] = manifest
  .filter((m): m is { id: TourId; seconds: number } => Object.hasOwn(PAGES, m.id))
  .map((m) => ({ id: m.id, seconds: m.seconds, ...PAGES[m.id] }));

export const DEFAULT_TOUR: TourId = "overview";

export function tourOf(id: string | null | undefined): Tour {
  return TOURS.find((t) => t.id === id) ?? TOURS.find((t) => t.id === DEFAULT_TOUR) ?? (TOURS[0] as Tour);
}

export function tourFile(id: TourId): string {
  return `internet-monitor-tour-${id}.mp4`;
}

export function tourVideoUrl(id: TourId, version: string | null | undefined): string {
  if (version && RELEASE_TAG.test(version)) return `${RELEASES}/download/${version}/${tourFile(id)}`;
  return `${RELEASES}/latest/download/${tourFile(id)}`;
}

export function tourSubtitlesUrl(id: TourId, lang: string): string {
  return `/tutorial/${id}/${lang}.vtt`;
}

export function tourPosterUrl(id: TourId): string {
  return `/tutorial/${id}/poster.jpg`;
}

export function tourClock(locale: string, seconds: number): string {
  const total = Math.max(0, Math.round(seconds));
  const minutes = new Intl.NumberFormat(locale).format(Math.floor(total / 60));
  const rest = new Intl.NumberFormat(locale, { minimumIntegerDigits: 2 }).format(total % 60);
  return `${minutes}:${rest}`;
}
