import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { LOCALES } from "../lib/i18n";
import { Tutorial } from "./Tutorial";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("Tutorial", () => {
  test("loads nothing until play and offers subtitles in every language", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const { container } = render(<Tutorial />);
    const video = container.querySelector("video");
    expect(video?.getAttribute("preload")).toBe("none");
    expect(video?.hasAttribute("controls")).toBe(true);
    expect(video?.getAttribute("src")).toBe(
      "https://github.com/Khaymah/internet-monitor/releases/latest/download/internet-monitor-tutorial.mp4",
    );
    const tracks = [...container.querySelectorAll("track")];
    expect(
      tracks.map((t) => [t.getAttribute("kind"), t.getAttribute("srclang"), t.getAttribute("src")]),
    ).toEqual(LOCALES.map((l) => ["subtitles", l.code, `/tutorial/${l.code}.vtt`]));
    expect(tracks.filter((t) => t.hasAttribute("default")).map((t) => t.getAttribute("srclang"))).toEqual([
      "en",
    ]);
    expect(screen.getByText(/streams from GitHub only when you press play/)).toBeTruthy();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  test("explains a failed load and offers to try again", () => {
    const { container } = render(<Tutorial />);
    const video = container.querySelector("video") as HTMLVideoElement;
    vi.spyOn(video, "load").mockImplementation(() => undefined);
    vi.spyOn(video, "play").mockImplementation(() => Promise.resolve());
    fireEvent.error(video);
    expect(screen.getByText(/couldn't be loaded from GitHub/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Try again/ }));
    expect(video.load).toHaveBeenCalled();
    expect(screen.queryByText(/couldn't be loaded from GitHub/)).toBeNull();
  });
});
