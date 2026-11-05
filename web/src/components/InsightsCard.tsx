import { CircleAlert, CircleCheck, Info, Lightbulb } from "lucide-react";
import type { CSSProperties } from "react";
import { api } from "../lib/api";
import { formatNumber } from "../lib/format";
import { useAsync } from "../lib/hooks";
import { type MessageKey, useI18n } from "../lib/i18n";
import { navigate } from "../lib/router";
import type { Insight } from "../lib/types";
import { tone } from "../styles/categories";
import { Button, Card, CardHead } from "./ui";

const LEVEL_TONE: Record<Insight["level"], string> = {
  warn: tone.degraded,
  info: tone.checking,
  good: tone.connected,
};

const LEVEL_ICON = { warn: CircleAlert, info: Info, good: CircleCheck };

const ACTION: Record<string, { label: MessageKey; to: "complaints" | "speed" | "settings" }> = {
  file_complaint: { label: "ins_act_complain", to: "complaints" },
  provider_route: { label: "ins_act_complain", to: "complaints" },
  slow_vs_plan: { label: "ins_act_complain", to: "complaints" },
  bufferbloat: { label: "ins_act_speed", to: "speed" },
};

export function InsightsCard({ days = 30, limit = 4 }: { days?: number; limit?: number }) {
  const { t, lang } = useI18n();
  const data = useAsync((signal) => api.insights(days, signal), [days]);
  const list = (data.data ?? []).slice(0, limit);
  if (!list.length) return null;
  return (
    <Card labelledBy="ins-title">
      <CardHead
        id="ins-title"
        title={
          <span className="row" style={{ gap: 8 }}>
            <Lightbulb size={18} aria-hidden />
            {t("ins_title")}
          </span>
        }
        sub={t("ins_sub", { days: formatNumber(lang, days) })}
      />
      <ul className="insights">
        {list.map((ins) => {
          const Icon = LEVEL_ICON[ins.level];
          const vars = Object.fromEntries(
            Object.entries(ins.vars ?? {}).map(([k, v]) => [
              k,
              typeof v === "number" ? formatNumber(lang, v, v % 1 ? 1 : 0) : v,
            ]),
          );
          const action = ACTION[ins.id];
          return (
            <li key={ins.id} className="insight" style={{ "--tone": LEVEL_TONE[ins.level] } as CSSProperties}>
              <Icon size={18} className="insight-icon" aria-hidden />
              <div className="stack" style={{ gap: 4 }}>
                <strong>{t(`ins_${ins.id}_t` as MessageKey, vars)}</strong>
                <p className="secondary">{t(`ins_${ins.id}_d` as MessageKey, vars)}</p>
                {action ? (
                  <div>
                    <Button size="sm" variant="edge" onClick={() => navigate(action.to)}>
                      {t(action.label)}
                    </Button>
                  </div>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
