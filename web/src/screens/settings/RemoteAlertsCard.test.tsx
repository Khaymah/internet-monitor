import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { REMOTE_ALERTS_URL } from "../../content/text";
import { RemoteAlertsCard } from "./RemoteAlertsCard";

afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("RemoteAlertsCard", () => {
  test("opens the issue page in the browser, sends nothing and remembers the click", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    render(<RemoteAlertsCard />);
    const link = screen.getByRole("link", { name: /I want this/ });
    expect(link.getAttribute("href")).toBe(REMOTE_ALERTS_URL);
    expect(link.getAttribute("target")).toBe("_blank");
    expect(link.getAttribute("rel")).toContain("noreferrer");

    link.addEventListener("click", (e) => e.preventDefault());
    fireEvent.click(link);
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(fetchSpy).not.toHaveBeenCalled();
    expect(localStorage.getItem("im.remoteAlertsInterest")).toBe("1");
    expect(screen.queryByRole("link", { name: /I want this/ })).toBeNull();
    expect(screen.getByText(/we'll mention it in What's new/)).toBeTruthy();
  });

  test("shows the thanks right away after a reload", () => {
    localStorage.setItem("im.remoteAlertsInterest", "1");
    render(<RemoteAlertsCard />);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText(/we'll mention it in What's new/)).toBeTruthy();
  });

  test("still works when storage is blocked", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    render(<RemoteAlertsCard />);
    const link = screen.getByRole("link", { name: /I want this/ });
    link.addEventListener("click", (e) => e.preventDefault());
    expect(() => fireEvent.click(link)).not.toThrow();
  });
});
