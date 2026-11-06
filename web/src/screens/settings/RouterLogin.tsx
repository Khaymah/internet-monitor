import { Plug } from "lucide-react";
import { useState } from "react";
import { TextField } from "../../components/Field";
import { toast } from "../../components/Toast";
import { Button } from "../../components/ui";
import { ApiError, api } from "../../lib/api";
import { errorKey } from "../../lib/errors";
import { formatNumber } from "../../lib/format";
import { useI18n } from "../../lib/i18n";
import type { ConfigView } from "../../lib/types";
import { RouterReport } from "./RouterReport";
import { SecretField } from "./SecretField";
import type { SectionProps } from "./Sections";

export function RouterLogin({ draft, set, err, base }: SectionProps & { base: ConfigView }) {
  const { t, lang } = useI18n();
  const [busy, setBusy] = useState(false);
  const typed = draft.router_password.mode === "set" ? draft.router_password.value : "";
  const canTest =
    typed !== "" || (draft.router_password.mode === "keep" && !!base.secret_set.router_password);

  const test = async () => {
    setBusy(true);
    try {
      const r = await api.testRouter(draft.router_user.trim() || undefined, typed || undefined);
      toast(
        t(r.has_counters ? "rt_test_ok_usage" : "rt_test_ok", {
          speed: formatNumber(lang, r.down_kbps / 1000, 1),
          snr: formatNumber(lang, r.snr_down_db, 1),
        }),
        "ok",
      );
    } catch (e) {
      const key = errorKey(e, "rt_web_unreachable");
      const detail =
        e instanceof ApiError && key === "rt_web_unreachable" ? e.message.replace(/\n/g, " · ") : "";
      toast(detail ? `${t(key)} ${detail}` : t(key), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <p className="field-hint full">{t("f_router_login_hint")}</p>
      <TextField
        label={t("f_router_address")}
        hint={t("f_router_address_hint")}
        value={draft.router_address}
        onChange={(v) => set("router_address", v)}
        placeholder="192.168.1.1"
        error={err("router_address")}
        ltr
        narrow
      />
      <TextField
        label={t("f_router_user")}
        value={draft.router_user}
        onChange={(v) => set("router_user", v)}
        placeholder="admin"
        error={err("router_user")}
        ltr
        narrow
      />
      <SecretField
        t={t}
        label={t("f_router_password")}
        hint={t("f_router_password_hint")}
        error={err("router_password")}
        saved={!!base.secret_set.router_password}
        value={draft.router_password}
        onChange={(v) => set("router_password", v)}
        placeholder="••••••••"
        masked
      />
      <div className="full row">
        <Button variant="ghost" size="sm" icon={Plug} busy={busy} disabled={!canTest} onClick={test}>
          {t("rt_test")}
        </Button>
      </div>
      <hr className="divider full" />
      <RouterReport hasPassword={!!base.secret_set.router_password} />
    </>
  );
}
