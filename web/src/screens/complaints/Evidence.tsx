import { Copy, Download, Mail, MessageCircle } from "lucide-react";
import { useId, useState } from "react";
import { toast } from "../../components/Toast";
import { Button, Skeleton } from "../../components/ui";
import { api } from "../../lib/api";
import { downloadBlob } from "../../lib/csv";
import { formatDate, formatDuration, formatMoney, formatNumber, formatPct } from "../../lib/format";
import { useAsync } from "../../lib/hooks";
import type { Lang, Translate } from "../../lib/i18n";
import { useI18n } from "../../lib/i18n";
import type { CaseProfile, Complaint, RangeEvidence } from "../../lib/types";
import { availabilityTone, tone } from "../../styles/categories";

export function useEvidence(from: string | null, to?: string) {
  return useAsync(
    (signal) => (from ? api.caseEvidence(from, to, signal) : Promise.resolve(null)),
    [from, to],
    `${from}|${to ?? ""}`,
  );
}

function plainMbps(lang: Lang, v: number): string {
  return `${formatNumber(lang, v, v < 10 ? 1 : 0)} Mbps`;
}

function mbps(lang: Lang, v: number): string {
  return `\u2066${plainMbps(lang, v)}\u2069`;
}

export function EvidenceStats({ ev, title }: { ev: RangeEvidence; title?: string }) {
  const { t, lang } = useI18n();
  const s = ev.summary;
  const sp = ev.speed;
  const rows: [string, string, string][] = [
    [
      t("c_ev_outages"),
      formatNumber(lang, s.disconnections),
      s.disconnections ? tone.disconnected : tone.connected,
    ],
    [t("c_ev_downtime"), formatDuration(t, s.total_downtime_seconds), tone.disconnected],
    [
      t("c_ev_longest"),
      s.longest_outage_seconds ? formatDuration(t, s.longest_outage_seconds) : "—",
      tone.degraded,
    ],
    [
      t("c_ev_uptime"),
      s.window_seconds > 0 ? formatPct(lang, s.uptime_pct) : "—",
      availabilityTone(s.window_seconds > 0 ? s.uptime_pct : null),
    ],
    [t("c_ev_coverage"), s.window_seconds > 0 ? formatPct(lang, s.coverage_pct) : "—", tone.checking],
  ];
  if (sp.tests > 0) {
    rows.push([
      t("c_ev_speed"),
      sp.plan_download_mbps > 0
        ? t("c_ev_speed_plan", {
            speed: mbps(lang, sp.median_download_mbps),
            pct: formatPct(lang, sp.pct_of_plan),
          })
        : mbps(lang, sp.median_download_mbps),
      tone.download,
    ]);
  }
  return (
    <div className="evidence">
      {title ? <h3 className="evidence-title">{title}</h3> : null}
      <dl className="evidence-grid">
        {rows.map(([label, value, color]) => (
          <div key={label} style={{ "--tone": color } as React.CSSProperties}>
            <dt>{label}</dt>
            <dd className="num">{value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

export function complaintText(
  t: Translate,
  lang: Lang,
  profile: CaseProfile,
  ev: RangeEvidence,
  complaint?: Complaint,
): string {
  const s = ev.summary;
  const sp = ev.speed;
  const when = (v: string) => formatDate(lang, v, { dateStyle: "medium", timeStyle: "short" });
  const head = [
    profile.provider ? t("c_tx_provider", { v: profile.provider }) : "",
    profile.line_number ? t("c_tx_line", { v: profile.line_number }) : "",
    profile.account_number ? t("c_tx_account", { v: profile.account_number }) : "",
    profile.plan_name || profile.plan_download_mbps > 0
      ? t("c_tx_plan", {
          v: [
            profile.plan_name,
            profile.plan_download_mbps > 0 ? plainMbps(lang, profile.plan_download_mbps) : "",
          ]
            .filter(Boolean)
            .join(" · "),
        })
      : "",
  ].filter(Boolean);
  const body = [
    t("c_tx_period", { from: when(ev.from), to: when(ev.to) }),
    s.disconnections > 0
      ? t("c_tx_outages", {
          outages: formatNumber(lang, s.disconnections),
          downtime: formatDuration(t, s.total_downtime_seconds),
          longest: formatDuration(t, s.longest_outage_seconds),
          at: s.longest_outage_at ? when(s.longest_outage_at) : "—",
        })
      : t("c_tx_no_outages"),
  ];
  if (sp.tests > 0) {
    body.push(
      sp.plan_download_mbps > 0
        ? t("c_tx_speed", {
            tests: formatNumber(lang, sp.tests),
            speed: plainMbps(lang, sp.median_download_mbps),
            pct: formatPct(lang, sp.pct_of_plan),
            plan: plainMbps(lang, sp.plan_download_mbps),
          })
        : t("c_tx_speed_noplan", {
            tests: formatNumber(lang, sp.tests),
            speed: plainMbps(lang, sp.median_download_mbps),
          }),
    );
  }
  if (s.window_seconds > 0) body.push(t("c_tx_coverage", { coverage: formatPct(lang, s.coverage_pct) }));
  if (s.links?.minutes && s.links.wired_pct >= 95) body.push(t("c_tx_wired"));
  if (complaint?.ticket)
    body.push(t("c_tx_ticket", { ticket: complaint.ticket, date: when(complaint.opened_at) }));
  const comp = ev.compensation;
  if (comp && comp.monthly_price > 0 && comp.pro_rata > 0) {
    body.push(
      t("c_tx_comp", {
        amount: formatMoney(lang, comp.pro_rata, comp.currency),
        price: formatMoney(lang, comp.monthly_price, comp.currency),
      }),
    );
  }
  body.push(t("c_tx_close"));
  return [t("c_tx_subject"), "", ...head, "", ...body].join("\n").replace(/\n{3,}/g, "\n\n");
}

async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

export function EvidenceActions({
  ev,
  profile,
  complaint,
}: {
  ev: RangeEvidence;
  profile: CaseProfile;
  complaint?: Complaint;
}) {
  const { t, lang } = useI18n();
  const id = useId();
  const [text, setText] = useState(() => complaintText(t, lang, profile, ev, complaint));
  const [busy, setBusy] = useState(false);

  const copy = async () => {
    if (await copyText(text)) toast(t("c_copied"), "ok");
    else toast(t("c_copy_failed"), "error");
  };
  const download = async () => {
    setBusy(true);
    try {
      const blob = await api.evidenceRange(ev.from, ev.to);
      downloadBlob(`internet-monitor-evidence-${ev.from.slice(0, 10)}_${ev.to.slice(0, 10)}.zip`, blob);
    } catch {
      toast(t("report_evidence_failed"), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="stack" style={{ gap: 10 }}>
      <label className="field-label" htmlFor={id}>
        {t("c_text_label")}
      </label>
      <textarea
        id={id}
        className="input textarea"
        rows={9}
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <div className="row">
        <Button size="sm" variant="solid" icon={Copy} onClick={copy}>
          {t("c_copy_text")}
        </Button>
        <Button size="sm" variant="edge" icon={Download} busy={busy} onClick={download}>
          {t("c_download_pack")}
        </Button>
        <a
          className="btn btn-edge btn-sm"
          href={`https://wa.me/?text=${encodeURIComponent(text)}`}
          target="_blank"
          rel="noreferrer"
        >
          <MessageCircle size={15} aria-hidden />
          {t("c_share_whatsapp")}
        </a>
        <a
          className="btn btn-edge btn-sm"
          href={`mailto:?subject=${encodeURIComponent(t("c_tx_subject"))}&body=${encodeURIComponent(text)}`}
        >
          <Mail size={15} aria-hidden />
          {t("c_share_email")}
        </a>
      </div>
    </div>
  );
}

export function EvidenceLoading() {
  return (
    <div className="stack" aria-busy="true" style={{ gap: 8 }}>
      <Skeleton height={64} />
    </div>
  );
}
