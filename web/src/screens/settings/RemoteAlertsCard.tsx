import { CircleCheck, ExternalLink } from "lucide-react";
import { useState } from "react";
import { Card, CardHead } from "../../components/ui";
import { REMOTE_ALERTS_URL } from "../../content/text";
import { useI18n } from "../../lib/i18n";

const INTEREST_KEY = "im.remoteAlertsInterest";

function readInterest(): boolean {
  try {
    return localStorage.getItem(INTEREST_KEY) === "1";
  } catch {
    return false;
  }
}

function writeInterest() {
  try {
    localStorage.setItem(INTEREST_KEY, "1");
  } catch {}
}

export function RemoteAlertsCard() {
  const { t } = useI18n();
  const [interested, setInterested] = useState(readInterest);

  return (
    <Card
      id="remote-alerts"
      className="settings-section"
      flare="var(--accent-violet)"
      labelledBy="remote-alerts-title"
    >
      <CardHead id="remote-alerts-title" title={t("ra_title")} />
      <p className="secondary">{t("ra_body")}</p>
      <div className="remote-alerts-action" aria-live="polite">
        {interested ? (
          <span className="remote-alerts-thanks">
            <CircleCheck size={17} aria-hidden />
            {t("ra_thanks")}
          </span>
        ) : (
          <>
            <a
              className="btn btn-edge btn-sm"
              href={REMOTE_ALERTS_URL}
              target="_blank"
              rel="noreferrer noopener"
              onClick={() => {
                writeInterest();
                window.setTimeout(() => setInterested(true), 0);
              }}
            >
              {t("ra_want")}
              <ExternalLink size={14} aria-hidden />
              <span className="sr-only"> {t("opens_new_tab")}</span>
            </a>
            <span className="field-hint">{t("ra_hint")}</span>
          </>
        )}
      </div>
    </Card>
  );
}
