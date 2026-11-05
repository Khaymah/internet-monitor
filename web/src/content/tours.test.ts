import { describe, expect, test } from "vitest";
import { LOCALES } from "../lib/i18n";
import { tutorialSubtitlesUrl, tutorialVideoUrl } from "./tutorial";

const LATEST =
  "https://github.com/Khaymah/internet-monitor/releases/latest/download/internet-monitor-tutorial.mp4";

const SUBTITLES = import.meta.glob<string>("../../public/tutorial/*.vtt", {
  query: "?raw",
  import: "default",
  eager: true,
});

describe("tutorial video", () => {
  test.each(["v2.0.4", "v2.0.4-rc.8", "v10.12.3-rc.15"])("%s streams from its own release", (v) => {
    expect(tutorialVideoUrl(v)).toBe(
      `https://github.com/Khaymah/internet-monitor/releases/download/${v}/internet-monitor-tutorial.mp4`,
    );
  });

  test.each([undefined, null, "", "dev", "2.0.4", "v2.0.4-beta.1", "v2.0.4-rc.8-dirty", "v2.0"])(
    "%s falls back to the latest release",
    (v) => {
      expect(tutorialVideoUrl(v)).toBe(LATEST);
    },
  );

  test.each(LOCALES.map((l) => [l.code] as const))("%s has subtitles", (code) => {
    const text = SUBTITLES[`../../public${tutorialSubtitlesUrl(code)}`];
    expect(text?.startsWith("WEBVTT")).toBe(true);
    expect(text).toMatch(/\d\d:\d\d:\d\d\.\d{3} --> \d\d:\d\d:\d\d\.\d{3}/);
  });
});
