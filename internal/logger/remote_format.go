package logger

import (
	"fmt"
	"internet-monitor/internal/i18n"
	"internet-monitor/internal/monitor"
	"internet-monitor/internal/notifytext"
	"internet-monitor/internal/types"
	"strings"
	"time"
)

type remoteStyle struct {
	color int
	emoji string
	slack string
	title func(lang string) string
}

var remoteStyles = map[string]remoteStyle{
	"down":     {0xEF4444, "❌", ":x:", func(l string) string { return i18n.T(l, "remote.title.down", nil) }},
	"degraded": {0xEAB308, "⚠️", ":warning:", func(l string) string { return i18n.T(l, "remote.title.degraded", nil) }},
	"restored": {0x22C55E, "✅", ":white_check_mark:", func(l string) string { return i18n.T(l, "remote.title.restored", nil) }},
	"reminder": {0xEF4444, "⏰", ":alarm_clock:", func(l string) string { return i18n.T(l, "remote.title.reminder", nil) }},
}

func styleOf(kind string) remoteStyle {
	if s, ok := remoteStyles[kind]; ok {
		return s
	}
	return remoteStyles["down"]
}

func EventTitle(lang, kind string) string { return styleOf(kind).title(lang) }

type remoteLine struct{ name, value string }

func eventLines(lang string, ev monitor.Event) (checks, metrics []remoteLine) {
	mark := func(failed bool) string {
		if failed {
			return "❌ " + i18n.T(lang, "remote.failed", nil)
		}
		return "✅ " + i18n.T(lang, "remote.ok", nil)
	}
	checks = []remoteLine{
		{"🔌 " + i18n.T(lang, "remote.tcp", nil), mark(ev.Reason.TCPPingFailed)},
		{"🌐 " + i18n.T(lang, "remote.http", nil), mark(ev.Reason.HTTPFailed)},
		{"🔍 " + i18n.T(lang, "remote.dns", nil), mark(ev.Reason.DNSFailed)},
	}
	if ev.Reason.PacketLossPct > 0 {
		metrics = append(metrics, remoteLine{"📉 " + i18n.T(lang, "remote.loss", nil), fmt.Sprintf("%.1f%%", ev.Reason.PacketLossPct)})
	}
	if ev.Reason.AvgLatencyMs > 0 {
		metrics = append(metrics, remoteLine{"⚡ " + i18n.T(lang, "remote.latency", nil), notifytext.Latency(lang, ev.Reason.AvgLatencyMs)})
	}
	if ev.DurationSeconds >= 1 {
		metrics = append(metrics, remoteLine{"⏱️ " + i18n.T(lang, "remote.duration", nil), notifytext.Duration(lang, ev.DurationSeconds)})
	}
	return checks, metrics
}

func BuildEventPayload(lang string, ev monitor.Event, kind, url string) interface{} {
	st := styleOf(kind)
	title := EventTitle(lang, kind)
	checks, metrics := eventLines(lang, ev)
	if IsDiscord(url) {
		fields := make([]map[string]interface{}, 0, len(checks)+len(metrics))
		for _, l := range append(checks, metrics...) {
			fields = append(fields, field(l.name, l.value, true))
		}
		return map[string]interface{}{
			"username": "Internet Monitor",
			"embeds": []map[string]interface{}{{
				"title":     st.emoji + " " + title,
				"color":     st.color,
				"fields":    fields,
				"timestamp": ev.Timestamp.UTC().Format(time.RFC3339),
				"footer":    map[string]string{"text": "Internet Monitor • " + i18n.T(lang, "remote.footer", nil)},
			}},
		}
	}
	lines := []string{fmt.Sprintf("%s *%s*", st.slack, title), ""}
	parts := make([]string, 0, len(checks))
	for _, l := range checks {
		parts = append(parts, fmt.Sprintf("*%s:* %s", l.name, l.value))
	}
	lines = append(lines, strings.Join(parts, "  "))
	for _, l := range metrics {
		lines = append(lines, fmt.Sprintf("*%s:* %s", l.name, l.value))
	}
	return map[string]interface{}{"text": strings.Join(lines, "\n")}
}

func TelegramEventText(lang string, ev monitor.Event, kind string) string {
	checks, metrics := eventLines(lang, ev)
	var b strings.Builder
	fmt.Fprintf(&b, "%s *%s*", styleOf(kind).emoji, EventTitle(lang, kind))
	for _, l := range append(checks, metrics...) {
		fmt.Fprintf(&b, "\n%s: %s", l.name, l.value)
	}
	return b.String()
}

func speedLines(lang string, ev SpeedTestEvent, thresholdMbps float64, below bool) []remoteLine {
	lines := []remoteLine{{"📥 " + i18n.T(lang, "remote.download", nil), fmt.Sprintf("%.1f Mbps", ev.DownloadMbps)}}
	if ev.UploadMbps != nil {
		lines = append(lines, remoteLine{"📤 " + i18n.T(lang, "remote.upload", nil), fmt.Sprintf("%.1f Mbps", *ev.UploadMbps)})
	}
	lines = append(lines, remoteLine{"📡 " + i18n.T(lang, "remote.latency", nil), notifytext.Latency(lang, ev.LatencyMs)})
	if ev.DurationSeconds > 0 {
		lines = append(lines, remoteLine{"⏱️ " + i18n.T(lang, "remote.duration", nil), fmt.Sprintf("%.1f s", ev.DurationSeconds)})
	}
	if below && thresholdMbps > 0 {
		lines = append(lines, remoteLine{"⚠️ " + i18n.T(lang, "remote.threshold", nil), fmt.Sprintf("%.1f Mbps", thresholdMbps)})
	}
	return lines
}

func speedTitle(lang string, below bool) (string, string, int) {
	if below {
		return "⚠️", i18n.T(lang, "remote.speed.drop", nil), 0xEAB308
	}
	return "🚀", i18n.T(lang, "remote.speed.done", nil), 0x22C55E
}

func BuildSpeedResultPayload(lang string, ev SpeedTestEvent, thresholdMbps float64, below bool, url string) interface{} {
	icon, title, color := speedTitle(lang, below)
	lines := speedLines(lang, ev, thresholdMbps, below)
	if IsDiscord(url) {
		fields := make([]map[string]interface{}, 0, len(lines))
		for _, l := range lines {
			fields = append(fields, field(l.name, l.value, true))
		}
		return map[string]interface{}{
			"username": "Internet Monitor",
			"embeds": []map[string]interface{}{{
				"title":     icon + " " + title,
				"color":     color,
				"fields":    fields,
				"timestamp": ev.Timestamp.UTC().Format(time.RFC3339),
			}},
		}
	}
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, fmt.Sprintf("%s: *%s*", l.name, l.value))
	}
	return map[string]interface{}{
		"blocks": []map[string]interface{}{{
			"type": "section",
			"text": map[string]interface{}{"type": "mrkdwn", "text": fmt.Sprintf("*%s %s*\n%s", icon, title, strings.Join(parts, " | "))},
		}},
	}
}

func TelegramSpeedText(lang string, ev SpeedTestEvent, thresholdMbps float64, below bool) string {
	icon, title, _ := speedTitle(lang, below)
	var b strings.Builder
	fmt.Fprintf(&b, "%s *%s*", icon, title)
	for _, l := range speedLines(lang, ev, thresholdMbps, below) {
		fmt.Fprintf(&b, "\n%s: %s", l.name, l.value)
	}
	return b.String()
}

func weeklyLines(lang string, w types.WeeklySummary) []remoteLine {
	outages := fmt.Sprintf("%d", w.Outages)
	if w.PrevOutages != nil {
		outages = i18n.T(lang, "remote.weekly.vs_prev", i18n.Vars{"n": w.Outages, "prev": *w.PrevOutages})
	}
	lines := []remoteLine{
		{"🔌 " + i18n.T(lang, "remote.weekly.outages", nil), outages},
		{"⏱️ " + i18n.T(lang, "remote.weekly.downtime", nil), notifytext.Duration(lang, w.DowntimeSeconds)},
	}
	if w.LongestSeconds > 0 {
		lines = append(lines, remoteLine{"📏 " + i18n.T(lang, "remote.weekly.longest", nil), notifytext.Duration(lang, w.LongestSeconds)})
	}
	lines = append(lines,
		remoteLine{"📶 " + i18n.T(lang, "remote.weekly.uptime", nil), fmt.Sprintf("%.2f%%", w.UptimePct)},
		remoteLine{"👁️ " + i18n.T(lang, "remote.weekly.coverage", nil), fmt.Sprintf("%.0f%%", w.CoveragePct)},
	)
	if w.SpeedTests > 0 {
		v := fmt.Sprintf("%.1f Mbps", w.MedianMbps)
		if w.PlanMbps > 0 {
			v = i18n.T(lang, "remote.weekly.speed_plan", i18n.Vars{"speed": v, "pct": fmt.Sprintf("%.0f%%", w.PctOfPlan)})
		}
		lines = append(lines, remoteLine{"📥 " + i18n.T(lang, "remote.weekly.speed", nil), v})
	}
	return lines
}

func weeklyTitle(lang string, w types.WeeklySummary) (string, string, int) {
	period := w.From.Format("2006-01-02") + " → " + w.To.Format("2006-01-02")
	title := i18n.T(lang, "remote.weekly.title", i18n.Vars{"period": period})
	if w.Outages > 0 {
		return "📊", title, 0xEAB308
	}
	return "📊", title, 0x22C55E
}

func BuildWeeklyPayload(lang string, w types.WeeklySummary, url string) interface{} {
	icon, title, color := weeklyTitle(lang, w)
	lines := weeklyLines(lang, w)
	if IsDiscord(url) {
		fields := make([]map[string]interface{}, 0, len(lines))
		for _, l := range lines {
			fields = append(fields, field(l.name, l.value, true))
		}
		return map[string]interface{}{
			"username": "Internet Monitor",
			"embeds": []map[string]interface{}{{
				"title":     icon + " " + title,
				"color":     color,
				"fields":    fields,
				"timestamp": w.To.UTC().Format(time.RFC3339),
			}},
		}
	}
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, fmt.Sprintf("%s: *%s*", l.name, l.value))
	}
	return map[string]interface{}{
		"blocks": []map[string]interface{}{{
			"type": "section",
			"text": map[string]interface{}{"type": "mrkdwn", "text": fmt.Sprintf("*%s %s*\n%s", icon, title, strings.Join(parts, "\n"))},
		}},
	}
}

func TelegramWeeklyText(lang string, w types.WeeklySummary) string {
	icon, title, _ := weeklyTitle(lang, w)
	var b strings.Builder
	fmt.Fprintf(&b, "%s *%s*", icon, title)
	for _, l := range weeklyLines(lang, w) {
		fmt.Fprintf(&b, "\n%s: %s", l.name, l.value)
	}
	return b.String()
}

func GenericPayload(kind, title, text string, at time.Time, data interface{}) map[string]interface{} {
	out := map[string]interface{}{
		"source": "internet-monitor",
		"type":   kind,
		"title":  title,
		"text":   text,
		"time":   at.UTC().Format(time.RFC3339),
	}
	if data != nil {
		out["data"] = data
	}
	return out
}

func PlainText(s string) string { return strings.ReplaceAll(s, "*", "") }

func PlainBody(telegram string) string {
	_, body, ok := strings.Cut(PlainText(telegram), "\n")
	if !ok {
		return PlainText(telegram)
	}
	return body
}
