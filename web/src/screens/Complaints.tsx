import {
  CircleAlert,
  CircleCheck,
  FileText,
  FileWarning,
  Megaphone,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Trash2,
} from "lucide-react";
import { type CSSProperties, useEffect, useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Field, TextField } from "../components/Field";
import { SmartSelect } from "../components/SmartSelect";
import { toast } from "../components/Toast";
import { Banner, Button, Card, CardHead, Empty, Skeleton } from "../components/ui";
import { ApiError, api } from "../lib/api";
import { formatDate, formatDuration, monthKey } from "../lib/format";
import { useAsync, useNow } from "../lib/hooks";
import { type MessageKey, useI18n } from "../lib/i18n";
import { navigate } from "../lib/router";
import type { CaseProfile, Complaint, ComplaintStatus } from "../lib/types";
import { tone } from "../styles/categories";

export const RESPONSE_HOURS = 72;

const CHANNELS = ["hotline", "app", "web", "branch", "social", "other"] as const;
const CHANNEL_LABEL: Record<(typeof CHANNELS)[number], MessageKey> = {
  hotline: "c_ch_hotline",
  app: "c_ch_app",
  web: "c_ch_web",
  branch: "c_ch_branch",
  social: "c_ch_social",
  other: "c_ch_other",
};

const STATUSES: ComplaintStatus[] = ["open", "escalated", "resolved", "rejected"];
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

interface ProfileDraft {
  provider: string;
  plan_name: string;
  plan_download_mbps: string;
  plan_upload_mbps: string;
  line_number: string;
  account_number: string;
}

function toProfileDraft(p: CaseProfile): ProfileDraft {
  const n = (v: number) => (v > 0 ? String(v) : "");
  return {
    provider: p.provider,
    plan_name: p.plan_name,
    plan_download_mbps: n(p.plan_download_mbps),
    plan_upload_mbps: n(p.plan_upload_mbps),
    line_number: p.line_number,
    account_number: p.account_number,
  };
}

function toNumber(v: string): number {
  const n = Number(v.replace(",", ".").trim());
  return Number.isFinite(n) && n > 0 ? n : 0;
}

/** Converts an RFC3339 value to the "YYYY-MM-DDTHH:mm" a datetime-local input expects. */
function toLocalInput(value: string | undefined): string {
  const d = value ? new Date(value) : new Date();
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function ProfileCard({ profile, onSaved }: { profile: CaseProfile; onSaved: (p: CaseProfile) => void }) {
  const { t } = useI18n();
  const [draft, setDraft] = useState(() => toProfileDraft(profile));
  const [busy, setBusy] = useState(false);
  const set = (k: keyof ProfileDraft, v: string) => setDraft((d) => ({ ...d, [k]: v }));

  const save = async () => {
    setBusy(true);
    try {
      const f = await api.saveProfile({
        provider: draft.provider,
        plan_name: draft.plan_name,
        plan_download_mbps: toNumber(draft.plan_download_mbps),
        plan_upload_mbps: toNumber(draft.plan_upload_mbps),
        line_number: draft.line_number,
        account_number: draft.account_number,
      });
      onSaved(f.profile);
      setDraft(toProfileDraft(f.profile));
      toast(t("c_profile_saved"), "ok");
    } catch {
      toast(t("load_failed"), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card labelledBy="c-profile">
      <CardHead id="c-profile" title={t("c_profile_title")} sub={t("c_profile_sub")} />
      <div className="form-grid">
        <TextField
          label={t("c_provider")}
          value={draft.provider}
          onChange={(v) => set("provider", v)}
          placeholder={t("c_provider_ph")}
        />
        <TextField
          label={t("c_plan_name")}
          value={draft.plan_name}
          onChange={(v) => set("plan_name", v)}
          placeholder={t("c_plan_name_ph")}
        />
        <TextField
          label={t("c_plan_down")}
          hint={t("c_plan_hint")}
          value={draft.plan_download_mbps}
          onChange={(v) => set("plan_download_mbps", v)}
          inputMode="decimal"
          suffix="Mbps"
          narrow
        />
        <TextField
          label={t("c_plan_up")}
          value={draft.plan_upload_mbps}
          onChange={(v) => set("plan_upload_mbps", v)}
          inputMode="decimal"
          suffix="Mbps"
          narrow
        />
        <TextField
          label={t("c_line")}
          value={draft.line_number}
          onChange={(v) => set("line_number", v)}
          inputMode="numeric"
          ltr
        />
        <TextField
          label={t("c_account")}
          value={draft.account_number}
          onChange={(v) => set("account_number", v)}
          ltr
        />
      </div>
      <div className="row" style={{ marginTop: 20 }}>
        <Button variant="solid" icon={Save} busy={busy} onClick={save}>
          {t("c_save_profile")}
        </Button>
      </div>
    </Card>
  );
}

interface ComplaintDraft {
  id: string;
  provider: string;
  ticket: string;
  opened_at: string;
  channel: string;
  agent: string;
  status: ComplaintStatus;
  regulator_ticket: string;
  notes: string;
}

function toComplaintDraft(c: Partial<Complaint>, provider: string): ComplaintDraft {
  return {
    id: c.id ?? "",
    provider: c.provider ?? provider,
    ticket: c.ticket ?? "",
    opened_at: toLocalInput(c.opened_at),
    channel: c.channel || "hotline",
    agent: c.agent ?? "",
    status: c.status ?? "open",
    regulator_ticket: c.regulator_ticket ?? "",
    notes: c.notes ?? "",
  };
}

function ComplaintForm({
  initial,
  onSaved,
  onCancel,
}: {
  initial: ComplaintDraft;
  onSaved: () => void;
  onCancel: () => void;
}) {
  const { t } = useI18n();
  const [draft, setDraft] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const set = <K extends keyof ComplaintDraft>(k: K, v: ComplaintDraft[K]) =>
    setDraft((d) => ({ ...d, [k]: v }));

  const save = async () => {
    if (!draft.ticket.trim() && !draft.notes.trim()) {
      setError(t("c_err_ticket"));
      return;
    }
    setBusy(true);
    try {
      await api.saveComplaint({ ...draft, opened_at: draft.opened_at });
      toast(t("c_saved"), "ok");
      onSaved();
    } catch (e) {
      if (e instanceof ApiError && e.field === "ticket") setError(t("c_err_ticket"));
      else toast(t("load_failed"), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card labelledBy="c-form">
      <CardHead id="c-form" title={t(initial.id ? "c_form_edit" : "c_form_new")} />
      <div className="form-grid">
        <TextField
          label={t("c_ticket")}
          value={draft.ticket}
          onChange={(v) => {
            set("ticket", v);
            setError(undefined);
          }}
          placeholder={t("c_ticket_ph")}
          error={error}
          ltr
        />
        <TextField label={t("c_provider")} value={draft.provider} onChange={(v) => set("provider", v)} />
        <Field label={t("c_opened")}>
          {({ inputId, describedBy }) => (
            <input
              id={inputId}
              className="input ltr"
              dir="ltr"
              type="datetime-local"
              value={draft.opened_at}
              aria-describedby={describedBy}
              onChange={(e) => set("opened_at", e.target.value)}
            />
          )}
        </Field>
        <Field label={t("c_channel")}>
          {({ inputId, describedBy }) => (
            <SmartSelect
              id={inputId}
              describedBy={describedBy}
              label={t("c_channel")}
              value={draft.channel}
              onChange={(v) => set("channel", v)}
              options={CHANNELS.map((c) => ({ value: c, label: t(CHANNEL_LABEL[c]) }))}
            />
          )}
        </Field>
        <TextField label={t("c_agent")} value={draft.agent} onChange={(v) => set("agent", v)} />
        <Field label={t("c_status")}>
          {({ inputId, describedBy }) => (
            <SmartSelect
              id={inputId}
              describedBy={describedBy}
              label={t("c_status")}
              value={draft.status}
              onChange={(v) => set("status", v)}
              options={STATUSES.map((s) => ({ value: s, label: t(STATUS_LABEL[s]) }))}
            />
          )}
        </Field>
        {draft.status !== "open" ? (
          <TextField
            label={t("c_regulator_ticket")}
            value={draft.regulator_ticket}
            onChange={(v) => set("regulator_ticket", v)}
            ltr
          />
        ) : null}
        <Field label={t("c_notes")} className="full">
          {({ inputId, describedBy }) => (
            <textarea
              id={inputId}
              className="input textarea"
              rows={4}
              maxLength={2000}
              value={draft.notes}
              placeholder={t("c_notes_ph")}
              aria-describedby={describedBy}
              onChange={(e) => {
                set("notes", e.target.value);
                setError(undefined);
              }}
            />
          )}
        </Field>
      </div>
      <div className="row" style={{ marginTop: 20 }}>
        <Button variant="solid" icon={Save} busy={busy} onClick={save}>
          {t("c_save")}
        </Button>
        <Button variant="ghost" onClick={onCancel} disabled={busy}>
          {t("c_cancel")}
        </Button>
      </div>
    </Card>
  );
}

function ComplaintItem({
  c,
  now,
  onEdit,
  onChanged,
  onDelete,
}: {
  c: Complaint;
  now: number;
  onEdit: () => void;
  onChanged: () => void;
  onDelete: () => void;
}) {
  const { t, lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const opened = new Date(c.opened_at).getTime();
  const elapsed = Math.max(0, (now - opened) / 1000);
  const left = RESPONSE_HOURS * 3600 - elapsed;
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
        <span className="pill" style={{ "--tone": STATUS_TONE[c.status] } as CSSProperties}>
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
            {t("c_overdue")}
          </Banner>
        )
      ) : null}
      <div className="row">
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
        <Button
          variant="edge"
          icon={FileText}
          onClick={() => navigate("report", `?month=${monthKey(new Date())}`)}
        >
          {t("c_open_report")}
        </Button>
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
            <Card labelledBy="c-list">
              <CardHead id="c-list" title={t("c_list_title")}>
                {editing ? null : (
                  <Button
                    size="sm"
                    variant="solid"
                    icon={Plus}
                    onClick={() => setEditing(toComplaintDraft({}, profile.provider))}
                  >
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
            <ProfileCard profile={profile} onSaved={setProfile} />
          </div>
          <Card labelledBy="c-guide">
            <CardHead id="c-guide" title={t("c_guide_title")} />
            <ol className="guide-steps">
              <li>{t("c_guide_1")}</li>
              <li>{t("c_guide_2")}</li>
              <li>{t("c_guide_3")}</li>
              <li>{t("c_guide_4")}</li>
            </ol>
            <p className="muted" style={{ marginTop: 12 }}>
              {t("c_guide_note")}
            </p>
          </Card>
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
