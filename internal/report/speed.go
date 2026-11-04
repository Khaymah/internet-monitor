package report

import (
	"sort"
	"time"
)

// SpeedSample is the subset of a speedtest_*.jsonl line the report needs.
type SpeedSample struct {
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   *float64  `json:"upload_mbps"`
	LatencyMs    int64     `json:"latency_ms"`
}

type SpeedSummary struct {
	Tests            int     `json:"tests"`
	AvgDownloadMbps  float64 `json:"avg_download_mbps"`
	MedianDownload   float64 `json:"median_download_mbps"`
	MinDownloadMbps  float64 `json:"min_download_mbps"`
	MaxDownloadMbps  float64 `json:"max_download_mbps"`
	AvgUploadMbps    float64 `json:"avg_upload_mbps"`
	AvgLatencyMs     int64   `json:"avg_latency_ms"`
	PlanDownloadMbps float64 `json:"plan_download_mbps"`
	// PctOfPlan is the median download as a share of the plan speed (0 when no plan is set).
	PctOfPlan float64 `json:"pct_of_plan"`
	// BelowHalfPlan counts tests that measured less than half the plan speed.
	BelowHalfPlan int `json:"below_half_plan"`
}

// SummarizeSpeed aggregates the month's speed tests and compares them with
// the plan speed the user entered (planMbps <= 0 skips the comparison).
func SummarizeSpeed(tests []SpeedSample, month string, planMbps float64) SpeedSummary {
	out := SpeedSummary{PlanDownloadMbps: max(planMbps, 0)}
	var downs []float64
	var upSum, latSum float64
	ups := 0
	for _, t := range tests {
		if month != "" && t.Timestamp.Format("2006-01") != month {
			continue
		}
		if t.DownloadMbps <= 0 {
			continue
		}
		downs = append(downs, t.DownloadMbps)
		latSum += float64(t.LatencyMs)
		if t.UploadMbps != nil && *t.UploadMbps > 0 {
			upSum += *t.UploadMbps
			ups++
		}
		if planMbps > 0 && t.DownloadMbps < planMbps/2 {
			out.BelowHalfPlan++
		}
	}
	out.Tests = len(downs)
	if out.Tests == 0 {
		return out
	}
	sort.Float64s(downs)
	var sum float64
	for _, d := range downs {
		sum += d
	}
	out.AvgDownloadMbps = sum / float64(len(downs))
	out.MinDownloadMbps = downs[0]
	out.MaxDownloadMbps = downs[len(downs)-1]
	mid := len(downs) / 2
	if len(downs)%2 == 0 {
		out.MedianDownload = (downs[mid-1] + downs[mid]) / 2
	} else {
		out.MedianDownload = downs[mid]
	}
	out.AvgLatencyMs = int64(latSum / float64(len(downs)))
	if ups > 0 {
		out.AvgUploadMbps = upSum / float64(ups)
	}
	if planMbps > 0 {
		out.PctOfPlan = out.MedianDownload / planMbps * 100
	}
	return out
}
