import { Save } from "lucide-react";
import { useState } from "react";
import { Field, TextField } from "../../components/Field";
import { SmartSelect } from "../../components/SmartSelect";
import { toast } from "../../components/Toast";
import { Button, Card, CardHead } from "../../components/ui";
import { ApiError, api } from "../../lib/api";
import { useI18n } from "../../lib/i18n";
import type { Complaint, ComplaintStatus } from "../../lib/types";
import { CHANNEL_LABEL, CHANNELS, STATUS_LABEL, STATUSES, toLocalInput } from "./shared";

export interface ComplaintDraft {
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

export function toComplaintDraft(c: Partial<Complaint>, provider: string): ComplaintDraft {
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

export function ComplaintForm({
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
