package report

import (
	"sort"
	"time"
)

type SpeedSample struct {
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   *float64  `json:"upload_mbps"`
	LatencyMs    int64     `json:"latency_ms"`
	Bufferbloat  string    `json:"bufferbloat,omitempty"`
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
	PctOfPlan        float64 `json:"pct_of_plan"`
	BelowHalfPlan    int     `json:"below_half_plan"`
	PeakMedian       float64 `json:"peak_median_mbps"`
	OffPeakMedian    float64 `json:"off_peak_median_mbps"`
	PeakTests        int     `json:"peak_tests"`
}

func Median(v []float64) float64 { return medianOf(v) }

func medianOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 0 {
		return (s[m-1] + s[m]) / 2
	}
	return s[m]
}

func SummarizeSpeed(tests []SpeedSample, month string, planMbps float64) SpeedSummary {
	out := SpeedSummary{PlanDownloadMbps: max(planMbps, 0)}
	var downs, peak, offPeak []float64
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
		if h := t.Timestamp.Local().Hour(); h >= PeakFrom && h < PeakTo {
			peak = append(peak, t.DownloadMbps)
		} else {
			offPeak = append(offPeak, t.DownloadMbps)
		}
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
	out.PeakTests = len(peak)
	out.PeakMedian = medianOf(peak)
	out.OffPeakMedian = medianOf(offPeak)
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
