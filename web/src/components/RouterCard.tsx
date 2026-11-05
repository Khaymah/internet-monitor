import { Router } from "lucide-react";
import type { CSSProperties } from "react";
import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { formatDate, formatDuration, formatNumber } from "../lib/format";
import { useAsync } from "../lib/hooks";
import { type Lang, useI18n } from "../lib/i18n";
import { navigate } from "../lib/router";
import type { PlanUsage, RouterStatus } from "../lib/types";
import { tone } from "../styles/categories";
import { Button, Card, CardHead, Meter } from "./ui";

function gb(lang: Lang, bytes: number): string {
  const v = bytes / 1e9;
  return `\u2066${formatNumber(lang, v, v < 10 ? 1 : 0)} GB\u2069`;
}

function speed(lang: Lang, kbps: number): string {
  const v = kbps / 1000;
  return formatNumber(lang, v, v < 10 ? 1 : 0);
}

function useMinuteTick(): number {
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const id = window.setInterval(() => setTick((n) => n + 1), 60_000);
    return () => window.clearInterval(id);
  }, []);
  return tick;
}

function StatusGrid({ st }: { st: RouterStatus }) {
  const { t, lang } = useI18n();
  const rows: [string, string, string][] = [];
  if (st.link_up !== undefined) {
    rows.push([
      t("rt_line"),
      [st.link_up ? t("rt_line_up") : t("rt_line_down"), st.access_type].filter(Boolean).join(" · "),
      st.link_up ? tone.connected : tone.disconnected,
    ]);
  }
  if (st.sync_down_kbps) {
    rows.push([
      t("rt_sync"),
      `⁦↓ ${speed(lang, st.sync_down_kbps)}${st.sync_up_kbps ? ` · ↑ ${speed(lang, st.sync_up_kbps)}` : ""} Mbps⁩`,
      tone.download,
    ]);
  }
  if (st.connected !== undefined) {
    rows.push([
      t("rt_session"),
      st.connected ? t("rt_session_up") : t("rt_session_down"),
      st.connected ? tone.connected : tone.disconnected,
    ]);
  }
  if (st.uptime_seconds) rows.push([t("rt_uptime"), formatDuration(t, st.uptime_seconds), tone.checking]);
  if (!rows.length) return null;
  return (
    <dl className="evidence-grid">
      {rows.map(([label, value, color]) => (
        <div key={label} style={{ "--tone": color } as CSSProperties}>
          <dt>{label}</dt>
          <dd className="num">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function Usage({ u }: { u: PlanUsage }) {
  const { t, lang } = useI18n();
  const since = formatDate(lang, u.from, { day: "numeric", month: "short" });
  if (!u.quota_bytes) {
    return (
      <div className="stack" style={{ gap: 8 }}>
        <p>{t("rt_used_noquota", { used: gb(lang, u.used_bytes), since })}</p>
        <div className="row">
          <Button size="sm" variant="edge" onClick={() => navigate("complaints")}>
            {t("rt_set_quota")}
          </Button>
        </div>
      </div>
    );
  }
  const color = u.used_pct >= 100 ? tone.disconnected : u.used_pct >= 80 ? tone.degraded : tone.connected;
  return (
    <div className="stack" style={{ gap: 6 }}>
      <Meter
        label={t("rt_used_label", { since })}
        value={u.used_pct}
        display={t("rt_used_of", { used: gb(lang, u.used_bytes), quota: gb(lang, u.quota_bytes) })}
        tone={color}
      />
      <p className="secondary">
        {u.projected_bytes > u.quota_bytes
          ? t("rt_pace_over", {
              days: formatNumber(lang, u.days_left),
              projected: gb(lang, u.projected_bytes),
            })
          : t("rt_pace_ok", { days: formatNumber(lang, u.days_left) })}
      </p>
    </div>
  );
}

export function RouterCard() {
  const { t } = useI18n();
  const tick = useMinuteTick();
  const data = useAsync((signal) => api.router(signal), [tick]);
  const v = data.data;
  if (!v?.enabled) return null;
  const st = v.status;
  return (
    <Card labelledBy="rt-title">
      <CardHead
        id="rt-title"
        title={
          <span className="row" style={{ gap: 8 }}>
            <Router size={18} aria-hidden />
            {t("rt_title")}
          </span>
        }
        sub={st?.model ? t("rt_sub_model", { model: st.model }) : t("rt_sub")}
      />
      <div className="stack" style={{ gap: 16 }}>
        {st ? <StatusGrid st={st} /> : <p className="secondary">{t("rt_not_found")}</p>}
        {v.usage ? (
          <Usage u={v.usage} />
        ) : st && !st.has_counters ? (
          <p className="muted">{t("rt_no_counters")}</p>
        ) : null}
      </div>
    </Card>
  );
}
