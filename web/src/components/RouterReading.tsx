import { ExternalLink } from "lucide-react";
import { type MessageKey, useI18n } from "../lib/i18n";
import type { RouterView } from "../lib/types";

export const ROUTER_SUPPORT_URL =
  "https://github.com/Khaymah/internet-monitor/issues/new?labels=router-support";

const DRIVER_LABEL: Record<string, MessageKey> = {
  "huawei-hg": "rt_driver_huawei_hg",
  tr064: "rt_driver_tr064",
};

export function driverKey(driver: string): MessageKey {
  return DRIVER_LABEL[driver] ?? "rt_driver_other";
}

export function RouterReading({ view }: { view: Pick<RouterView, "web_ok" | "web_error" | "driver"> }) {
  const { t } = useI18n();
  if (view.web_error === "unsupported") {
    return (
      <p className="secondary">
        {t("rt_web_unsupported")}{" "}
        <a href={ROUTER_SUPPORT_URL} target="_blank" rel="noreferrer noopener">
          {t("rt_support_link")}
          <ExternalLink size={13} aria-hidden style={{ marginInlineStart: 4 }} />
          <span className="sr-only"> {t("opens_new_tab")}</span>
        </a>
      </p>
    );
  }
  if (view.web_ok && view.driver) return <p className="secondary">{t(driverKey(view.driver))}</p>;
  return null;
}
