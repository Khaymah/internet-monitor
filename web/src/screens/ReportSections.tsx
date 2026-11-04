import type { CSSProperties } from "react";
import { Card, CardHead, Meter } from "../components/ui";
import { formatDate, formatDuration, formatNumber, formatPct } from "../lib/format";
import { type MessageKey, useI18n } from "../lib/i18n";
import type { MonthlyReport } from "../lib/types";
import { availabilityTone } from "../styles/categories";
import { channelLabel, STATUS_LABEL, STATUS_TONE } from "./Complaints";

/** Speed with its unit, isolated as left-to-right so "31 Mbps" never flips to "Mbps 31" in RTL text. */
export function formatMbps(lang: string, v: number): string {
  return `\u2066${formatNumber(lang, v, v < 10 ? 1 : 0)} Mbps\u2069`;
}

export function SpeedCard({ report: r }: { report: MonthlyReport }) {
  const { t, lang } = useI18n();
  const sp = r.speed;
  const rows: [MessageKey, string][] = [
    ["report_speed_tests", formatNumber(lang, sp.tests)],
    ["report_speed_median", formatMbps(lang, sp.median_download_mbps)],
    [
      "report_speed_range",
      `${formatMbps(lang, sp.min_download_mbps)} – ${formatMbps(lang, sp.max_download_mbps)}`,
    ],
  ];
  if (sp.avg_upload_mbps > 0) rows.push(["report_speed_up", formatMbps(lang, sp.avg_upload_mbps)]);
  if (sp.plan_download_mbps > 0) {
    rows.push(["report_speed_plan", formatMbps(lang, sp.plan_download_mbps)]);
    rows.push(["report_speed_below", formatNumber(lang, sp.below_half_plan)]);
  }
  return (
    <Card labelledBy="report-speed">
      <CardHead id="report-speed" title={t("report_speed")} />
      <div className="stack" style={{ gap: 16 }}>
        {sp.plan_download_mbps > 0 ? (
          <Meter
            label={t("report_speed_pct")}
            value={Math.min(100, sp.pct_of_plan)}
            display={formatPct(lang, sp.pct_of_plan)}
            tone={availabilityTone(sp.pct_of_plan >= 80 ? 99 : sp.pct_of_plan >= 50 ? 96 : 0)}
          />
        ) : null}
        <div className="table-wrap">
          <table className="table">
            <tbody>
              {rows.map(([k, v]) => (
                <tr key={k}>
                  <th scope="row">{t(k)}</th>
                  <td className="num">{v}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </Card>
  );
}

export function GapsCard({ report: r }: { report: MonthlyReport }) {
  const { t, lang } = useI18n();
  return (
    <Card labelledBy="report-gaps">
      <CardHead
        id="report-gaps"
        title={t("report_gaps")}
        sub={`${formatDuration(t, r.unmonitored_seconds)} · ${t("report_gaps_sub")}`}
      />
      <p className="secondary" style={{ marginBottom: 12 }}>
        {t("report_gaps_explain")}
      </p>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th scope="col">{t("report_gap_from")}</th>
              <th scope="col">{t("report_gap_to")}</th>
              <th scope="col">{t("col_duration")}</th>
              <th scope="col">{t("col_cause")}</th>
            </tr>
          </thead>
          <tbody>
            {r.gaps.map((g) => (
              <tr key={g.start}>
                <td className="num">
                  {formatDate(lang, g.start, { dateStyle: "medium", timeStyle: "short" })}
                </td>
                <td className="num">
                  {formatDate(lang, g.end, { dateStyle: "medium", timeStyle: "short" })}
                </td>
                <td className="num">{formatDuration(t, g.duration_seconds)}</td>
                <td className="wrap secondary">
                  {t(g.kind === "power" ? "report_gap_power" : "report_gap_off")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

export function ComplaintsCard({ report: r }: { report: MonthlyReport }) {
  const { t, lang } = useI18n();
  return (
    <Card labelledBy="report-complaints">
      <CardHead
        id="report-complaints"
        title={t("report_complaints")}
        sub={formatNumber(lang, r.complaints.length)}
      />
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th scope="col">{t("c_opened")}</th>
              <th scope="col">{t("c_ticket")}</th>
              <th scope="col">{t("c_provider")}</th>
              <th scope="col">{t("c_status")}</th>
              <th scope="col">{t("c_notes")}</th>
            </tr>
          </thead>
          <tbody>
            {r.complaints.map((c) => {
              const ch = channelLabel(c.channel);
              return (
                <tr key={c.id}>
                  <td className="num">
                    {formatDate(lang, c.opened_at, { dateStyle: "medium", timeStyle: "short" })}
                  </td>
                  <td className="num" dir="ltr">
                    {c.ticket || "—"}
                  </td>
                  <td>{[c.provider, ch ? t(ch) : c.channel, c.agent].filter(Boolean).join(" · ")}</td>
                  <td>
                    <span className="row" style={{ gap: 8, flexWrap: "nowrap" }}>
                      <span
                        className="dot"
                        style={{ "--tone": STATUS_TONE[c.status] } as CSSProperties}
                        aria-hidden
                      />
                      {t(STATUS_LABEL[c.status])}
                    </span>
                    {c.regulator_ticket ? (
                      <div className="muted num" dir="ltr">
                        {c.regulator_ticket}
                      </div>
                    ) : null}
                  </td>
                  <td className="wrap secondary">{c.notes}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
