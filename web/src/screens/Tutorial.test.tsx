import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { TOURS } from "../content/tours";
import { LOCALES } from "../lib/i18n";
import { Tutorial } from "./Tutorial";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  window.history.replaceState(null, "", "/");
});

function open(path: string) {
  window.history.replaceState(null, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
  return render(<Tutorial />);
}

describe("Tutorial", () => {
  test("shows every tour and plays the overview by default, loading nothing until play", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const { container } = open("/tutorial");
    const cards = [...container.querySelectorAll(".tour-card")];
    expect(cards.map((a) => a.getAttribute("href"))).toEqual(TOURS.map((t) => `/tutorial?v=${t.id}`));
    expect(
      cards.filter((a) => a.getAttribute("aria-current") === "page").map((a) => a.getAttribute("href")),
    ).toEqual(["/tutorial?v=overview"]);
    const video = container.querySelector("video");
    expect(video?.getAttribute("preload")).toBe("none");
    expect(video?.hasAttribute("controls")).toBe(true);
    expect(video?.getAttribute("poster")).toBe("/tutorial/overview/poster.jpg");
    expect(video?.getAttribute("src")).toBe(
      "https://github.com/Khaymah/internet-monitor/releases/latest/download/internet-monitor-tour-overview.mp4",
    );
    const tracks = [...container.querySelectorAll("track")];
    expect(
      tracks.map((t) => [t.getAttribute("kind"), t.getAttribute("srclang"), t.getAttribute("src")]),
    ).toEqual(LOCALES.map((l) => ["subtitles", l.code, `/tutorial/overview/${l.code}.vtt`]));
    expect(tracks.filter((t) => t.hasAttribute("default")).map((t) => t.getAttribute("srclang"))).toEqual([
      "en",
    ]);
    expect(screen.getByText(/streams from GitHub only when you press play/)).toBeTruthy();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  test("a deep link opens that page's tour", () => {
    const { container } = open("/tutorial?v=complaints");
    expect(container.querySelector("video")?.getAttribute("src")).toMatch(
      /internet-monitor-tour-complaints\.mp4$/,
    );
    expect(container.querySelector("#tour-now")?.textContent).toBe("Complaints & evidence");
    expect(screen.getByRole("link", { name: "Go to Complaints" }).getAttribute("href")).toBe("/complaints");
  });

  test("explains a failed load and offers to try again", () => {
    const { container } = open("/tutorial?v=settings");
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
