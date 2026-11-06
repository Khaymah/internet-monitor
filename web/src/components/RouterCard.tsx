import { LifeBuoy, Router } from "lucide-react";
import type { CSSProperties } from "react";
import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { formatDate, formatDuration, formatNumber } from "../lib/format";
import { useAsync } from "../lib/hooks";
import { type Lang, type MessageKey, useI18n } from "../lib/i18n";
import { navigate } from "../lib/router";
import { goToTarget } from "../lib/spotlight";
import type { PlanUsage, RouterStatus, RouterView } from "../lib/types";
import { tone } from "../styles/categories";
import { Button, Card, CardHead, Meter } from "./ui";

const ROUTER_REPORT_ANCHOR = "router-report";

function gb(lang: Lang, bytes: number): string {
  const v = bytes / 1e9;
  return `\u2066${formatNumber(lang, v, v < 10 ? 1 : 0)} GB\u2069`;
}

const WEB_ERROR: Record<NonNullable<RouterView["web_error"]>, MessageKey> = {
  wrong_password: "rt_web_wrong_password",
  busy: "rt_web_busy",
  unsupported: "rt_web_unsupported",
  unreachable: "rt_web_unreachable",
};

const LAST_ERROR: Record<string, MessageKey> = {
  ERROR_ISP_DISCONNECT: "rt_err_isp",
  ERROR_ISP_TIME_OUT: "rt_err_isp_timeout",
  ERROR_USER_DISCONNECT: "rt_err_user",
  ERROR_IDLE_DISCONNECT: "rt_err_idle",
  ERROR_AUTHENTICATION_FAILURE: "rt_err_auth",
};

function speed(lang: Lang, kbps: number): string {
  const v = kbps / 1000;
  return formatNumber(lang, v, v < 10 ? 1 : 0);
}

function pair(down: number, up: number | undefined, fmt: (v: number) => string): string {
  const line = (arrow: string, v: number) => `\u2066${arrow} ${fmt(v)}\u2069`;
  return up === undefined || up === 0 ? line("↓", down) : `${line("↓", down)}\n${line("↑", up)}`;
}

function db(lang: Lang, v: number): string {
  return `${formatNumber(lang, v, 1)} dB`;
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
      pair(st.sync_down_kbps, st.sync_up_kbps, (v) => `${speed(lang, v)} Mbps`),
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
  if (st.max_down_kbps) {
    rows.push([
      t("rt_max"),
      pair(st.max_down_kbps, st.max_up_kbps, (v) => `${speed(lang, v)} Mbps`),
      tone.checking,
    ]);
  }
  if (st.snr_down_db !== undefined) {
    rows.push([
      t("rt_snr"),
      pair(st.snr_down_db, st.snr_up_db, (v) => db(lang, v)),
      st.snr_down_db < 6 ? tone.disconnected : st.snr_down_db < 9 ? tone.degraded : tone.connected,
    ]);
  }
  if (st.attn_down_db !== undefined) {
    rows.push([t("rt_attn"), pair(st.attn_down_db, st.attn_up_db, (v) => db(lang, v)), tone.checking]);
  }
  if (st.line_up_seconds)
    rows.push([t("rt_line_up_for"), formatDuration(t, st.line_up_seconds), tone.checking]);
  if (st.uptime_seconds) rows.push([t("rt_uptime"), formatDuration(t, st.uptime_seconds), tone.checking]);
  const reason = st.last_error ? LAST_ERROR[st.last_error] : undefined;
  if (reason) rows.push([t("rt_last_drop"), t(reason), tone.degraded]);
  if (!rows.length) return null;
  return (
    <dl className="evidence-grid">
      {rows.map(([label, value, color]) => (
        <div key={label} style={{ "--tone": color } as CSSProperties}>
          <dt>{label}</dt>
          <dd className="num" style={{ whiteSpace: "pre-line" }}>
            {value}
          </dd>
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
  const unreadable = !st || v.web_error === "unsupported" || (!st.has_counters && !v.usage);
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
        {v.web_error ? <p className="secondary">{t(WEB_ERROR[v.web_error])}</p> : null}
        {v.usage ? (
          <Usage u={v.usage} />
        ) : st && !st.has_counters ? (
          <p className="muted">{t("rt_no_counters")}</p>
        ) : null}
        {unreadable ? (
          <div className="row">
            <Button
              size="sm"
              variant="edge"
              icon={LifeBuoy}
              onClick={() => goToTarget("settings", ROUTER_REPORT_ANCHOR)}
            >
              {t("rt_help")}
            </Button>
          </div>
        ) : null}
      </div>
    </Card>
  );
}
