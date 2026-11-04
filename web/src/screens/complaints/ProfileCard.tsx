import { Save } from "lucide-react";
import { useState } from "react";
import { Field, TextField } from "../../components/Field";
import { SmartSelect } from "../../components/SmartSelect";
import { toast } from "../../components/Toast";
import { Button, Card, CardHead } from "../../components/ui";
import { api } from "../../lib/api";
import { useI18n } from "../../lib/i18n";
import type { CaseProfile } from "../../lib/types";
import { COUNTRIES, type CountryCode, countryOf, daysText, toNumber } from "./shared";

interface ProfileDraft {
  country: CountryCode;
  response_days: string;
  provider: string;
  plan_name: string;
  plan_download_mbps: string;
  plan_upload_mbps: string;
  line_number: string;
  account_number: string;
}

function toDraft(p: CaseProfile): ProfileDraft {
  const n = (v: number) => (v > 0 ? String(v) : "");
  return {
    country: p.country,
    response_days: p.response_hours > 0 ? String(Math.round((p.response_hours / 24) * 10) / 10) : "",
    provider: p.provider,
    plan_name: p.plan_name,
    plan_download_mbps: n(p.plan_download_mbps),
    plan_upload_mbps: n(p.plan_upload_mbps),
    line_number: p.line_number,
    account_number: p.account_number,
  };
}

export function ProfileCard({
  profile,
  onSaved,
}: {
  profile: CaseProfile;
  onSaved: (p: CaseProfile) => void;
}) {
  const { t, lang } = useI18n();
  const [draft, setDraft] = useState(() => toDraft(profile));
  const [busy, setBusy] = useState(false);
  const set = <K extends keyof ProfileDraft>(k: K, v: ProfileDraft[K]) => setDraft((d) => ({ ...d, [k]: v }));
  const rules = countryOf({ country: draft.country });

  const save = async () => {
    setBusy(true);
    try {
      const f = await api.saveProfile({
        country: draft.country,
        response_hours: Math.round(toNumber(draft.response_days) * 24),
        provider: draft.provider,
        plan_name: draft.plan_name,
        plan_download_mbps: toNumber(draft.plan_download_mbps),
        plan_upload_mbps: toNumber(draft.plan_upload_mbps),
        line_number: draft.line_number,
        account_number: draft.account_number,
      });
      onSaved(f.profile);
      setDraft(toDraft(f.profile));
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
        <Field label={t("c_country")}>
          {({ inputId, describedBy }) => (
            <SmartSelect
              id={inputId}
              describedBy={describedBy}
              label={t("c_country")}
              value={draft.country}
              onChange={(v) => set("country", v)}
              options={COUNTRIES.map((c) => ({ value: c.code, label: t(c.name) }))}
            />
          )}
        </Field>
        <TextField
          label={t("c_response")}
          hint={t("c_response_hint", { window: daysText(t, lang, rules.hours / 24), country: t(rules.name) })}
          value={draft.response_days}
          onChange={(v) => set("response_days", v)}
          inputMode="decimal"
          suffix={t("c_days_unit")}
          placeholder={String(rules.hours / 24)}
          narrow
        />
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
