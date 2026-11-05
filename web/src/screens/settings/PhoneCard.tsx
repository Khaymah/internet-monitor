import { ShieldAlert } from "lucide-react";
import { useEffect, useState } from "react";
import { Card, CardHead, Ltr } from "../../components/ui";
import { api } from "../../lib/api";
import { useI18n } from "../../lib/i18n";
import type { PhoneInfo } from "../../lib/types";

export function PhoneCard() {
  const { t } = useI18n();
  const [info, setInfo] = useState<PhoneInfo | null>(null);

  useEffect(() => {
    api.phone().then(setInfo, () => setInfo(null));
  }, []);

  if (!info?.available) return null;

  return (
    <Card id="phone" className="settings-section" labelledBy="phone-title">
      <CardHead id="phone-title" title={t("phone_title")} sub={t("phone_sub")} />
      <div className="phone-qr">
        <img
          className="phone-qr-code"
          src="/api/phone/qr.png"
          width={196}
          height={196}
          alt={t("phone_qr_alt")}
        />
        <div className="stack">
          <div className="field">
            <span className="field-label">{t("phone_address")}</span>
            <Ltr className="field-hint">{info.address}</Ltr>
          </div>
          <div className="list-row">
            <span className="icon-well" style={{ color: "var(--accent-amber)" }}>
              <ShieldAlert size={17} aria-hidden />
            </span>
            <span className="list-row-main">
              <span className="list-row-title">{t("phone_warning_title")}</span>
              <span className="list-row-sub">{t("phone_warning")}</span>
            </span>
          </div>
        </div>
      </div>
    </Card>
  );
}
