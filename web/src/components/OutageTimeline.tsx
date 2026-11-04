import { CircleAlert } from "lucide-react";
import { type CSSProperties, useId, useMemo, useRef, useState } from "react";
import { api } from "../lib/api";
import { formatDate, formatDuration, formatNumber, formatTime } from "../lib/format";
import { useAsync, useNow } from "../lib/hooks";
import { type MessageKey, useI18n } from "../lib/i18n";
import type { TimelineSegment, TimelineStatus } from "../lib/types";
import { tone } from "../styles/categories";
import { ChartTooltip } from "./ChartTooltip";
import { Banner, Card, CardHead, Segmented, Skeleton } from "./ui";

type Range = "24" | "168";

const STATUS_TONE: Record<TimelineStatus, string> = {
  connected: tone.connected,
  degraded: tone.degraded,
  disconnected: tone.disconnected,
  off: "var(--color-track)",
};

const STATUS_LABEL: Record<TimelineStatus, MessageKey> = {
  connected: "status_connected",
  degraded: "status_degraded",
  disconnected: "status_disconnected",
  off: "tl_off",
};

const MIN_WIDTH_PCT = 0.35;

export function OutageTimeline() {
  const { t, lang } = useI18n();
  const [range, setRange] = useState<Range>("24");
  const minute = Math.floor(useNow(60_000) / 60_000);
  const data = useAsync((signal) => api.timeline(Number(range), signal), [range, minute], range);
  const wrap = useRef<HTMLDivElement>(null);
  const tipId = useId();
  const [active, setActive] = useState<{ seg: TimelineSegment; x: number; y: number; w: number } | null>(
    null,
  );

  const view = useMemo(() => {
    const d = data.data;
    if (!d) return null;
    const from = new Date(d.from).getTime();
    const span = Math.max(1, new Date(d.to).getTime() - from);
    let outages = 0;
    let downtime = 0;
    const segs = d.segments.map((s) => {
      const start = new Date(s.start).getTime();
      const end = new Date(s.end).getTime();
      if (s.status === "disconnected") {
        outages++;
        downtime += (end - start) / 1000;
      }
      return {
        ...s,
        left: ((start - from) / span) * 100,
        width: ((end - start) / span) * 100,
        secs: (end - start) / 1000,
      };
    });
    return { segs, outages, downtime };
  }, [data.data]);

  const show = (seg: TimelineSegment, el: HTMLElement) => {
    const box = wrap.current?.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    if (!box) return;
    setActive({ seg, x: r.left + r.width / 2 - box.left, y: r.top - box.top, w: box.width });
  };

  const when = (v: string) =>
    range === "24"
      ? formatTime(lang, v, false)
      : formatDate(lang, v, { weekday: "short", hour: "numeric", minute: "2-digit" });

  return (
    <Card labelledBy="tl-title">
      <CardHead
        id="tl-title"
        title={t("tl_title")}
        sub={
          view
            ? view.outages
              ? t("tl_sub", {
                  outages: formatNumber(lang, view.outages),
                  downtime: formatDuration(t, view.downtime),
                })
              : t("tl_sub_none")
            : undefined
        }
      >
        <Segmented
          label={t("tl_range")}
          value={range}
          onChange={(v) => setRange(v)}
          options={[
            { value: "24", label: t("tl_24h") },
            { value: "168", label: t("tl_7d") },
          ]}
        />
      </CardHead>
      {data.error && !data.data ? (
        <Banner tone="danger" icon={CircleAlert}>
          {t("load_failed")}
        </Banner>
      ) : !view || !data.data ? (
        <Skeleton height={40} />
      ) : (
        <div className="chart" ref={wrap}>
          <div className="timeline" onPointerLeave={() => setActive(null)}>
            {view.segs.map((s) => {
              const style = {
                insetInlineStart: `${s.left}%`,
                width: `${Math.max(s.width, s.status === "connected" ? 0 : MIN_WIDTH_PCT)}%`,
                "--tone": STATUS_TONE[s.status],
              } as CSSProperties;
              const label = `${t(STATUS_LABEL[s.status])}: ${when(s.start)} – ${when(s.end)} (${formatDuration(t, s.secs)})`;
              return s.status === "connected" ? (
                <span
                  key={s.start}
                  className="timeline-seg"
                  style={style}
                  onPointerEnter={(e) => show(s, e.currentTarget)}
                />
              ) : (
                <button
                  type="button"
                  key={s.start}
                  className="timeline-seg"
                  data-status={s.status}
                  style={style}
                  aria-label={label}
                  aria-describedby={active?.seg.start === s.start ? tipId : undefined}
                  onPointerEnter={(e) => show(s, e.currentTarget)}
                  onFocus={(e) => show(s, e.currentTarget)}
                  onBlur={() => setActive(null)}
                />
              );
            })}
          </div>
          {active ? (
            <ChartTooltip
              id={tipId}
              x={active.x}
              y={active.y}
              bounds={active.w}
              title={t(STATUS_LABEL[active.seg.status as TimelineStatus])}
              rows={[
                { label: t("tl_from"), value: when(active.seg.start) },
                { label: t("tl_to"), value: when(active.seg.end) },
                {
                  label: t("col_duration"),
                  value: formatDuration(
                    t,
                    (new Date(active.seg.end).getTime() - new Date(active.seg.start).getTime()) / 1000,
                  ),
                  tone: STATUS_TONE[active.seg.status as TimelineStatus],
                },
              ]}
            />
          ) : null}
          <div className="row" style={{ justifyContent: "space-between", marginTop: 8 }}>
            <span className="chart-axis num muted">{when(data.data.from)}</span>
            <span className="chart-axis muted">{t("tl_now")}</span>
          </div>
          <div className="legend" style={{ marginTop: 8 }}>
            {(["connected", "degraded", "disconnected", "off"] as TimelineStatus[]).map((st) => (
              <span key={st}>
                <span className="dot" style={{ "--tone": STATUS_TONE[st] } as CSSProperties} />
                {t(STATUS_LABEL[st])}
              </span>
            ))}
          </div>
        </div>
      )}
    </Card>
  );
}
