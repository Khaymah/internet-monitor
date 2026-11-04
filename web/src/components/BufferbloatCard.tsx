import { Gamepad2 } from "lucide-react";
import type { CSSProperties } from "react";
import { latencyParts } from "../lib/format";
import { type MessageKey, useI18n } from "../lib/i18n";
import type { SpeedTestResult } from "../lib/types";
import { tone } from "../styles/categories";
import { Card } from "./ui";

export const GRADE_TONE: Record<string, string> = {
  "A+": tone.connected,
  A: tone.connected,
  B: tone.degraded,
  C: tone.degraded,
  D: tone.disconnected,
  F: tone.disconnected,
};

const GRADE_TEXT: Record<string, MessageKey> = {
  "A+": "bb_good",
  A: "bb_good",
  B: "bb_ok",
  C: "bb_bad",
  D: "bb_worse",
  F: "bb_worse",
};

export function BufferbloatCard({ result }: { result: SpeedTestResult }) {
  const { t, lang } = useI18n();
  const grade = result.bufferbloat;
  if (!grade || !result.idle_latency_ms) return null;
  const loaded = Math.max(result.loaded_latency_down_ms ?? 0, result.loaded_latency_up_ms ?? 0);
  const ms = (v: number) => {
    const p = latencyParts(t, lang, v);
    return `${p.value} ${p.unit}`;
  };
  return (
    <Card className="bufferbloat" flare={GRADE_TONE[grade]} labelledBy="bb-title">
      <div className="bb-grade" style={{ "--tone": GRADE_TONE[grade] } as CSSProperties} aria-hidden>
        {grade}
      </div>
      <div className="stack" style={{ gap: 6 }}>
        <h2 id="bb-title" className="row" style={{ gap: 8 }}>
          <Gamepad2 size={18} aria-hidden />
          {t("bb_title", { grade })}
        </h2>
        <p className="secondary">{t(GRADE_TEXT[grade] ?? "bb_ok")}</p>
        <p className="muted num">
          {[
            t("bb_idle", { ms: ms(result.idle_latency_ms) }),
            result.loaded_latency_down_ms ? t("bb_down", { ms: ms(result.loaded_latency_down_ms) }) : "",
            result.loaded_latency_up_ms ? t("bb_up", { ms: ms(result.loaded_latency_up_ms) }) : "",
            t("bb_added", { ms: ms(Math.max(0, loaded - result.idle_latency_ms)) }),
          ]
            .filter(Boolean)
            .join(" · ")}
        </p>
      </div>
    </Card>
  );
}
