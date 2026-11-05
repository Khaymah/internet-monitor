import {
  ChevronDown,
  ChevronUp,
  CircleAlert,
  CircleCheck,
  FileText,
  FileWarning,
  Megaphone,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react";
import { type CSSProperties, useEffect, useMemo, useState } from "react";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { InsightsCard } from "../../components/InsightsCard";
import { toast } from "../../components/Toast";
import { TourLink } from "../../components/TourLink";
import { Banner, Button, Card, CardHead, Empty, Skeleton } from "../../components/ui";
import { api } from "../../lib/api";
import { formatDate, formatDuration, formatNumber, monthKey } from "../../lib/format";
import { useAsync, useNow } from "../../lib/hooks";
import { useI18n } from "../../lib/i18n";
import { navigate } from "../../lib/router";
import type { CaseProfile, Complaint, ComplaintStatus } from "../../lib/types";
import { type ComplaintDraft, ComplaintForm, toComplaintDraft } from "./ComplaintForm";
import { EvidenceActions, EvidenceLoading, EvidenceStats, useEvidence } from "./Evidence";
import { ProfileCard } from "./ProfileCard";
import { channelLabel, countryOf, daysText, responseHours, STATUS_LABEL, STATUS_TONE } from "./shared";

const WEEK_MS = 7 * 24 * 3600 * 1000;

function RecentCard({ profile, onLog }: { profile: CaseProfile; onLog: (notes: string) => void }) {
  const { t, lang } = useI18n();
  const from = useMemo(() => new Date(Date.now() - WEEK_MS).toISOString(), []);
  const ev = useEvidence(from);
  const [open, setOpen] = useState(false);
  const s = ev.data?.summary;

  return (
    <Card labelledBy="c-recent">
      <CardHead id="c-recent" title={t("c_recent_title")} sub={t("c_recent_sub")}>
        {ev.data ? (
          <Button
            size="sm"
            variant="ghost"
            icon={open ? ChevronUp : ChevronDown}
            onClick={() => setOpen((o) => !o)}
          >
            {t(open ? "c_ev_hide" : "c_text_show")}
          </Button>
        ) : null}
      </CardHead>
      {ev.error ? (
        <Banner tone="danger" icon={CircleAlert}>
          {t("load_failed")}
        </Banner>
      ) : !ev.data || !s ? (
        <EvidenceLoading />
      ) : (
        <div className="stack" style={{ gap: 16 }}>
          <EvidenceStats ev={ev.data} />
          {s.disconnections > 0 ? (
            <Banner
              tone="warn"
              icon={Megaphone}
              actions={
                <Button
                  size="sm"
                  variant="solid"
                  icon={Plus}
                  onClick={() =>
                    onLog(
                      t("c_notes_prefill", {
                        outages: formatNumber(lang, s.disconnections),
                        downtime: formatDuration(t, s.total_downtime_seconds),
                        longest: formatDuration(t, s.longest_outage_seconds),
                      }),
                    )
                  }
                >
                  {t("c_log_with_evidence")}
                </Button>
              }
            >
              {t("c_recent_suggest", {
                outages: formatNumber(lang, s.disconnections),
                downtime: formatDuration(t, s.total_downtime_seconds),
              })}
            </Banner>
          ) : (
            <p className="muted">{t("c_recent_ok")}</p>
          )}
          {open ? <EvidenceActions ev={ev.data} profile={profile} /> : null}
        </div>
      )}
    </Card>
  );
}

function ComplaintEvidence({ c, profile, open }: { c: Complaint; profile: CaseProfile; open: boolean }) {
  const { t, lang } = useI18n();
  const until = c.closed_at || undefined;
  const since = useEvidence(c.opened_at, until);
  const beforeFrom = useMemo(
    () => new Date(new Date(c.opened_at).getTime() - WEEK_MS).toISOString(),
    [c.opened_at],
  );
  const before = useEvidence(open ? beforeFrom : null, c.opened_at);
  const whole = useEvidence(open ? beforeFrom : null, until);
  const s = since.data?.summary;

  return (
    <div className="stack" style={{ gap: 12 }}>
      {s ? (
        s.disconnections > 0 ? (
          <p className="verdict" style={{ "--tone": "var(--accent-coral)" } as CSSProperties}>
            {t("c_verdict_still", {
              outages: formatNumber(lang, s.disconnections),
              downtime: formatDuration(t, s.total_downtime_seconds),
            })}
          </p>
        ) : (
          <p className="verdict" style={{ "--tone": "var(--accent-green)" } as CSSProperties}>
            {t("c_verdict_clear")}
          </p>
        )
      ) : null}
      {open ? (
        <>
          <div className="grid grid-2">
            {before.data ? <EvidenceStats ev={before.data} title={t("c_ev_before")} /> : <EvidenceLoading />}
            {since.data ? <EvidenceStats ev={since.data} title={t("c_ev_since")} /> : <EvidenceLoading />}
          </div>
          {whole.data ? (
            <EvidenceActions ev={whole.data} profile={profile} complaint={c} />
          ) : (
            <EvidenceLoading />
          )}
        </>
      ) : null}
    </div>
  );
}

function ComplaintItem({
  c,
  profile,
  now,
  onEdit,
  onChanged,
  onDelete,
}: {
  c: Complaint;
  profile: CaseProfile;
  now: number;
  onEdit: () => void;
  onChanged: () => void;
  onDelete: () => void;
}) {
  const { t, lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);
  const elapsed = Math.max(0, (now - new Date(c.opened_at).getTime()) / 1000);
  const left = responseHours(profile) * 3600 - elapsed;
  const channel = channelLabel(c.channel);

  const mark = async (status: ComplaintStatus) => {
    setBusy(true);
    try {
      await api.saveComplaint({ ...c, status });
      onChanged();
    } catch {
      toast(t("load_failed"), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <li className="complaint">
      <div className="complaint-head">
        <span className="pill">
          <span className="dot" style={{ "--tone": STATUS_TONE[c.status] } as CSSProperties} aria-hidden />
          {t(STATUS_LABEL[c.status])}
        </span>
        <strong className="num" dir="ltr">
          {c.ticket || "—"}
        </strong>
        {c.provider ? <span className="secondary">{c.provider}</span> : null}
        <span className="spacer" />
        <span className="muted num">
          {formatDate(lang, c.opened_at, { dateStyle: "medium", timeStyle: "short" })}
        </span>
      </div>
      <p className="muted">
        {[
          t("c_opened_ago", { duration: formatDuration(t, elapsed) }),
          channel ? t(channel) : c.channel,
          c.agent,
          c.regulator_ticket ? `${t("c_regulator_ticket")}: ${c.regulator_ticket}` : "",
        ]
          .filter(Boolean)
          .join(" · ")}
      </p>
      {c.notes ? <p className="secondary wrap">{c.notes}</p> : null}
      {c.status === "open" ? (
        left > 0 ? (
          <p className="muted">{t("c_due_in", { duration: formatDuration(t, left) })}</p>
        ) : (
          <Banner tone="warn" icon={Megaphone}>
            {t("c_overdue", { regulator: t(countryOf(profile).regulator) })}
          </Banner>
        )
      ) : null}
      <ComplaintEvidence c={c} profile={profile} open={open} />
      <div className="row">
        <Button
          size="sm"
          variant="edge"
          icon={open ? ChevronUp : ChevronDown}
          onClick={() => setOpen((o) => !o)}
        >
          {t(open ? "c_ev_hide" : "c_ev_show")}
        </Button>
        <Button size="sm" variant="edge" icon={Pencil} onClick={onEdit}>
          {t("c_edit")}
        </Button>
        {c.status === "open" ? (
          <Button size="sm" variant="edge" icon={Megaphone} busy={busy} onClick={() => mark("escalated")}>
            {t("c_mark_escalated")}
          </Button>
        ) : null}
        {c.status === "open" || c.status === "escalated" ? (
          <Button size="sm" variant="edge" icon={CircleCheck} busy={busy} onClick={() => mark("resolved")}>
            {t("c_mark_resolved")}
          </Button>
        ) : null}
        <Button size="sm" variant="ghost" icon={Trash2} onClick={onDelete}>
          {t("c_delete")}
        </Button>
      </div>
    </li>
  );
}

function GuideCard({ profile }: { profile: CaseProfile }) {
  const { t, lang } = useI18n();
  const rules = countryOf(profile);
  const hours = responseHours(profile);
  const span = hours % 24 === 0 ? daysText(t, lang, hours / 24) : formatDuration(t, hours * 3600);
  return (
    <Card labelledBy="c-guide">
      <CardHead id="c-guide" title={t("c_guide_title", { country: t(rules.name) })} />
      <ol className="guide-steps">
        <li>{t("c_guide_1")}</li>
        <li>{t("c_guide_2", { window: span })}</li>
        <li>{t(rules.escalate)}</li>
        <li>{t("c_guide_4")}</li>
      </ol>
      {rules.code !== "EG" ? (
        <p className="muted" style={{ marginTop: 12 }}>
          {t("c_guide_note")}
        </p>
      ) : null}
    </Card>
  );
}

export function Complaints() {
  const { t } = useI18n();
  const data = useAsync((signal) => api.caseFile(signal), []);
  const [profile, setProfile] = useState<CaseProfile | null>(null);
  const [editing, setEditing] = useState<ComplaintDraft | null>(null);
  const [deleting, setDeleting] = useState<Complaint | null>(null);
  const now = useNow(60_000);

  useEffect(() => {
    if (data.data) setProfile(data.data.profile);
  }, [data.data]);

  const startNew = (notes = "") => {
    setEditing(toComplaintDraft({ notes }, profile?.provider ?? ""));
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const remove = async () => {
    const c = deleting;
    setDeleting(null);
    if (!c) return;
    try {
      await api.deleteComplaint(c.id);
      toast(t("c_deleted"), "ok");
      data.reload();
    } catch {
      toast(t("load_failed"), "error");
    }
  };

  const complaints = data.data?.complaints ?? [];

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("c_page_title")}</h1>
          <p>{t("c_page_sub")}</p>
        </div>
        <div className="row">
          <TourLink id="complaints" />
          <Button
            variant="edge"
            icon={FileText}
            onClick={() => navigate("report", `?month=${monthKey(new Date())}`)}
          >
            {t("c_open_report")}
          </Button>
        </div>
      </div>

      {data.error ? (
        <Banner
          tone="danger"
          icon={CircleAlert}
          actions={
            <Button size="sm" icon={RefreshCw} onClick={data.reload}>
              {t("retry")}
            </Button>
          }
        >
          {t("load_failed")}
        </Banner>
      ) : !data.data || !profile ? (
        <div className="stack" aria-busy="true">
          <Skeleton height={220} />
          <Skeleton height={160} />
        </div>
      ) : (
        <div className="grid grid-overview">
          <div className="stack">
            {editing ? (
              <ComplaintForm
                key={editing.id || "new"}
                initial={editing}
                onCancel={() => setEditing(null)}
                onSaved={() => {
                  setEditing(null);
                  data.reload();
                }}
              />
            ) : null}
            <RecentCard profile={profile} onLog={startNew} />
            <Card labelledBy="c-list">
              <CardHead id="c-list" title={t("c_list_title")}>
                {editing ? null : (
                  <Button size="sm" variant="solid" icon={Plus} onClick={() => startNew()}>
                    {t("c_add")}
                  </Button>
                )}
              </CardHead>
              {complaints.length === 0 ? (
                <Empty icon={FileWarning}>{t("c_empty")}</Empty>
              ) : (
                <ul className="complaints">
                  {complaints.map((c) => (
                    <ComplaintItem
                      key={c.id}
                      c={c}
                      profile={profile}
                      now={now}
                      onEdit={() => {
                        setEditing(toComplaintDraft(c, profile.provider));
                        window.scrollTo({ top: 0, behavior: "smooth" });
                      }}
                      onChanged={data.reload}
                      onDelete={() => setDeleting(c)}
                    />
                  ))}
                </ul>
              )}
            </Card>
            <ProfileCard key={profile.country + profile.provider} profile={profile} onSaved={setProfile} />
          </div>
          <div className="stack">
            <InsightsCard />
            <GuideCard profile={profile} />
          </div>
        </div>
      )}

      <ConfirmDialog
        open={deleting !== null}
        title={t("c_delete_title")}
        body={t("c_delete_body")}
        confirmLabel={t("c_delete")}
        cancelLabel={t("c_cancel")}
        danger
        onConfirm={remove}
        onCancel={() => setDeleting(null)}
      />
    </>
  );
}
