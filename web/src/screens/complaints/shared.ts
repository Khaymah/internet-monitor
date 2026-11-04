import { formatNumber } from "../../lib/format";
import { type Lang, localeOf, type MessageKey, type Translate } from "../../lib/i18n";
import type { CaseProfile, ComplaintStatus } from "../../lib/types";
import { tone } from "../../styles/categories";

export const CHANNELS = ["hotline", "app", "web", "branch", "social", "other"] as const;
export const CHANNEL_LABEL: Record<(typeof CHANNELS)[number], MessageKey> = {
  hotline: "c_ch_hotline",
  app: "c_ch_app",
  web: "c_ch_web",
  branch: "c_ch_branch",
  social: "c_ch_social",
  other: "c_ch_other",
};

export const STATUSES: ComplaintStatus[] = ["open", "escalated", "resolved", "rejected"];
export const STATUS_LABEL: Record<ComplaintStatus, MessageKey> = {
  open: "c_st_open",
  escalated: "c_st_escalated",
  resolved: "c_st_resolved",
  rejected: "c_st_rejected",
};
export const STATUS_TONE: Record<ComplaintStatus, string> = {
  open: tone.degraded,
  escalated: tone.checking,
  resolved: tone.connected,
  rejected: tone.disconnected,
};

export function channelLabel(channel: string): MessageKey | null {
  return (CHANNELS as readonly string[]).includes(channel)
    ? CHANNEL_LABEL[channel as (typeof CHANNELS)[number]]
    : null;
}

export type CountryCode = "EG" | "SA" | "GB" | "US" | "";

export interface CountryRules {
  code: CountryCode;
  name: MessageKey;
  regulator: MessageKey;
  escalate: MessageKey;
  hours: number;
}

export const COUNTRIES: CountryRules[] = [
  { code: "EG", name: "c_cn_eg", regulator: "c_reg_eg", escalate: "c_g3_eg", hours: 72 },
  { code: "SA", name: "c_cn_sa", regulator: "c_reg_sa", escalate: "c_g3_sa", hours: 120 },
  { code: "GB", name: "c_cn_gb", regulator: "c_reg_gb", escalate: "c_g3_gb", hours: 56 * 24 },
  { code: "US", name: "c_cn_us", regulator: "c_reg_us", escalate: "c_g3_us", hours: 72 },
  { code: "", name: "c_cn_other", regulator: "c_reg_other", escalate: "c_g3_other", hours: 72 },
];

export function countryOf(profile: Pick<CaseProfile, "country"> | null | undefined): CountryRules {
  return (
    COUNTRIES.find((c) => c.code === (profile?.country ?? "")) ??
    (COUNTRIES[COUNTRIES.length - 1] as CountryRules)
  );
}

export function responseHours(
  profile: Pick<CaseProfile, "country" | "response_hours"> | null | undefined,
): number {
  return profile && profile.response_hours > 0 ? profile.response_hours : countryOf(profile).hours;
}

export function toNumber(v: string): number {
  const n = Number(v.replace(",", ".").trim());
  return Number.isFinite(n) && n > 0 ? n : 0;
}

export function toLocalInput(value: string | undefined): string {
  const d = value ? new Date(value) : new Date();
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const DAY_FORMS: Record<Intl.LDMLPluralRule, MessageKey> = {
  zero: "c_days_zero",
  one: "c_days_one",
  two: "c_days_two",
  few: "c_days_few",
  many: "c_days_many",
  other: "c_days_other",
};

export function daysText(t: Translate, lang: Lang, days: number): string {
  const form = new Intl.PluralRules(localeOf(lang)).select(days);
  return t(DAY_FORMS[form], { n: formatNumber(lang, days, Number.isInteger(days) ? 0 : 1) });
}
