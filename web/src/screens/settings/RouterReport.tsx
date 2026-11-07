import { Download, ExternalLink, FileSearch } from "lucide-react";
import { useState } from "react";
import { toast } from "../../components/Toast";
import { Button } from "../../components/ui";
import { api } from "../../lib/api";
import { downloadText } from "../../lib/csv";
import { errorKey } from "../../lib/errors";
import { useI18n } from "../../lib/i18n";
import type { RouterReportResult } from "../../lib/types";

const ISSUE_PREFIX = "https://github.com/Khaymah/internet-monitor/issues/new?";

export function RouterReport({ hasPassword }: { hasPassword: boolean }) {
  const { t } = useI18n();
  const [deep, setDeep] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<RouterReportResult | null>(null);

  const create = async () => {
    setBusy(true);
    setResult(null);
    try {
      setResult(await api.routerReport(deep && hasPassword));
      toast(t("rr_ready"), "ok");
    } catch (e) {
      toast(t(errorKey(e, "rr_failed")), "error");
    } finally {
      setBusy(false);
    }
  };

  const download = () => {
    if (!result) return;
    downloadText(result.filename, JSON.stringify(result.report, null, 2), "application/json");
    toast(t("rr_downloaded"), "ok");
  };

  return (
    <div id="router-report" className="full stack" style={{ gap: 10 }}>
      <span className="field-label">{t("rr_title")}</span>
      <p className="field-hint">{t("rr_intro")}</p>
      <p className="field-hint">{t("rr_collects")}</p>
      <p className="field-hint">{t("rr_removes")}</p>
      <label className="row">
        <input
          type="checkbox"
          checked={deep && hasPassword}
          disabled={!hasPassword || busy}
          onChange={(e) => setDeep(e.target.checked)}
        />
        <span>{t("rr_deep")}</span>
      </label>
      <p className="field-hint">{hasPassword ? t("rr_deep_hint") : t("rr_deep_none")}</p>
      <div className="row">
        <Button variant="ghost" size="sm" icon={FileSearch} busy={busy} onClick={create}>
          {t("rr_create")}
        </Button>
      </div>
      {busy ? (
        <p className="secondary" role="status">
          {t("rr_working")}
        </p>
      ) : null}
      {result ? (
        <div className="stack" style={{ gap: 10 }}>
          <span className="field-label">{t("rr_summary")}</span>
          <pre className="report-box">{result.summary}</pre>
          <details>
            <summary>{t("rr_show_json")}</summary>
            <pre className="report-box">{JSON.stringify(result.report, null, 2)}</pre>
          </details>
          <p className="field-hint">{t("rr_next")}</p>
          <div className="row">
            <Button variant="edge" size="sm" icon={Download} onClick={download}>
              {t("rr_download")}
            </Button>
            {result.issue_url.startsWith(ISSUE_PREFIX) ? (
              <a
                className="btn btn-edge btn-sm"
                href={result.issue_url}
                target="_blank"
                rel="noreferrer noopener"
              >
                <ExternalLink size={15} aria-hidden />
                {t("rr_issue")}
              </a>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}
