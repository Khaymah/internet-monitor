import { describe, expect, test } from "vitest";
import { type MessageKey, translate, type Vars } from "../../lib/i18n";
import type { CaseProfile, RangeEvidence } from "../../lib/types";
import { complaintText } from "./Evidence";
import { daysText, responseHours } from "./shared";

const t = (key: MessageKey, vars?: Vars) => translate("en", key, vars);

const profile: CaseProfile = {
  country: "EG",
  response_hours: 0,
  provider: "WE",
  plan_name: "Super 70",
  plan_download_mbps: 70,
  plan_upload_mbps: 10,
  line_number: "0233334444",
  account_number: "",
};

function evidence(outages: number): RangeEvidence {
  return {
    from: "2026-11-01T00:00:00Z",
    to: "2026-11-08T00:00:00Z",
    summary: {
      disconnections: outages,
      total_downtime_seconds: outages * 600,
      longest_outage_seconds: outages ? 900 : 0,
      longest_outage_at: outages ? "2026-11-03T10:00:00Z" : "",
      window_seconds: 7 * 86400,
      coverage_pct: 95,
    } as RangeEvidence["summary"],
    speed: {
      tests: 3,
      median_download_mbps: 31,
      plan_download_mbps: 70,
      pct_of_plan: 44.3,
    } as RangeEvidence["speed"],
  };
}

describe("complaintText", () => {
  test("states outages, speed against the plan, coverage and the earlier ticket", () => {
    const text = complaintText(t, "en", profile, evidence(3), {
      id: "a",
      provider: "WE",
      ticket: "98765",
      opened_at: "2026-11-02T09:00:00Z",
      channel: "hotline",
      agent: "",
      status: "open",
      notes: "",
    });
    expect(text).toContain("Provider: WE");
    expect(text).toContain("Line number: 0233334444");
    expect(text).not.toContain("Account number");
    expect(text).toContain("dropped 3 times");
    expect(text).toContain("31 Mbps");
    expect(text).toContain("70 Mbps");
    expect(text).toContain("complaint number 98765");
    expect(text).not.toMatch(/[⁦⁩]/);
  });

  test("says so when no outage was recorded", () => {
    expect(complaintText(t, "en", profile, evidence(0))).toContain("no complete outage");
  });
});

describe("country rules", () => {
  test("a custom response window wins over the country default", () => {
    expect(responseHours(profile)).toBe(72);
    expect(responseHours({ ...profile, country: "GB" })).toBe(56 * 24);
    expect(responseHours({ ...profile, response_hours: 48 })).toBe(48);
  });

  test("Arabic day counts use the right plural form", () => {
    const ar = (key: MessageKey, vars?: Vars) => translate("ar", key, vars);
    expect(daysText(ar, "ar", 2)).toBe("يومين");
    expect(daysText(ar, "ar", 3)).toContain("أيام");
    expect(daysText(t, "en", 1)).toBe("1 day");
    expect(daysText(t, "en", 5)).toBe("5 days");
  });

  test("a fixed-number form is only used for its own number", () => {
    const fr = (key: MessageKey, vars?: Vars) => translate("fr", key, vars);
    expect(daysText(fr, "fr", 1)).toBe(translate("fr", "c_days_one"));
    expect(daysText(fr, "fr", 0)).not.toBe(translate("fr", "c_days_one"));
    expect(daysText(fr, "fr", 0.5)).toContain("0,5");
  });
});
