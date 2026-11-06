import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { api } from "../../lib/api";
import type { RouterReportResult } from "../../lib/types";
import { RouterReport } from "./RouterReport";

const RESULT: RouterReportResult = {
  report: { schema: 1, app: { version: "v1" } },
  summary: "Router: ZTE ZXHN H298A",
  filename: "router-report-zxhn-h298a.json",
  issue_url: "https://github.com/Khaymah/internet-monitor/issues/new?labels=router-support",
};

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("RouterReport", () => {
  test("creates the report, shows it and links to the issue without sending anything", async () => {
    const spy = vi.spyOn(api, "routerReport").mockResolvedValue(RESULT);
    render(<RouterReport hasPassword={false} />);
    expect((screen.getByRole("checkbox") as HTMLInputElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: /Create report/ }));
    await waitFor(() => expect(screen.getByText("Router: ZTE ZXHN H298A")).toBeTruthy());
    expect(spy).toHaveBeenCalledWith(false);
    const link = screen.getByRole("link", { name: /Open a GitHub issue/ });
    expect(link.getAttribute("href")).toBe(RESULT.issue_url);
    expect(link.getAttribute("rel")).toContain("noreferrer");
    expect(screen.getByRole("button", { name: /Download report/ })).toBeTruthy();
    expect(screen.getByText(/"schema": 1/)).toBeTruthy();
  });

  test("asks for the signed-in probe only when it is ticked and a password is saved", async () => {
    const spy = vi.spyOn(api, "routerReport").mockResolvedValue(RESULT);
    render(<RouterReport hasPassword />);
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: /Create report/ }));
    await waitFor(() => expect(spy).toHaveBeenCalledWith(true));
  });

  test("refuses a link that does not point to the project's issues", async () => {
    vi.spyOn(api, "routerReport").mockResolvedValue({ ...RESULT, issue_url: "https://evil.example/x" });
    render(<RouterReport hasPassword={false} />);
    fireEvent.click(screen.getByRole("button", { name: /Create report/ }));
    await waitFor(() => expect(screen.getByText("Router: ZTE ZXHN H298A")).toBeTruthy());
    expect(screen.queryByRole("link", { name: /Open a GitHub issue/ })).toBeNull();
  });
});
