import { describe, expect, test } from "vitest";
import { LOCALES } from "../lib/i18n";
import { TOURS, tourClock, tourOf, tourPosterUrl, tourSubtitlesUrl, tourVideoUrl } from "./tours";

const SUBTITLES = import.meta.glob<string>("../../public/tutorial/*/*.vtt", {
  query: "?raw",
  import: "default",
  eager: true,
});
const POSTERS = Object.keys(import.meta.glob("../../public/tutorial/*/poster.jpg"));

describe("video tours", () => {
  test("one video per page, in the order of the sidebar", () => {
    expect(TOURS.map((t) => t.id)).toEqual([
      "overview",
      "history",
      "notifications",
      "speed",
      "complaints",
      "report",
      "settings",
    ]);
  });

  test.each(["v2.0.4", "v2.0.4-rc.10", "v10.12.3-rc.15"])("%s streams from its own release", (v) => {
    expect(tourVideoUrl("history", v)).toBe(
      `https://github.com/Khaymah/internet-monitor/releases/download/${v}/internet-monitor-tour-history.mp4`,
    );
  });

  test.each([undefined, null, "", "dev", "2.0.4", "v2.0.4-beta.1", "v2.0.4-rc.8-dirty", "v2.0"])(
    "%s falls back to the latest release",
    (v) => {
      expect(tourVideoUrl("settings", v)).toBe(
        "https://github.com/Khaymah/internet-monitor/releases/latest/download/internet-monitor-tour-settings.mp4",
      );
    },
  );

  test("an unknown id opens the overview tour", () => {
    expect(tourOf("nope").id).toBe("overview");
    expect(tourOf(null).id).toBe("overview");
    expect(tourOf("report").id).toBe("report");
  });

  test("lengths read like a clock", () => {
    expect(tourClock("en", 125.4)).toBe("2:05");
    expect(tourClock("en", 59)).toBe("0:59");
  });

  test.each(TOURS.flatMap((t) => LOCALES.map((l) => [t.id, l.code] as const)))(
    "%s has %s subtitles",
    (id, code) => {
      const text = SUBTITLES[`../../public${tourSubtitlesUrl(id, code)}`];
      expect(text?.startsWith("WEBVTT")).toBe(true);
      expect(text).toMatch(/\d\d:\d\d:\d\d\.\d{3} --> \d\d:\d\d:\d\d\.\d{3}/);
    },
  );

  test.each(TOURS.map((t) => [t.id, t] as const))("%s has a poster and a length", (id, tour) => {
    expect(POSTERS).toContain(`../../public${tourPosterUrl(id)}`);
    expect(tour.seconds).toBeGreaterThan(30);
  });
});
