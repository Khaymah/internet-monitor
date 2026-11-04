package report

import (
	"internet-monitor/internal/types"
	"math"
	"testing"
	"time"
)

func minutes(from time.Time, n int) []types.MetricSample {
	out := make([]types.MetricSample, n)
	for i := range out {
		out[i] = types.MetricSample{Timestamp: from.Add(time.Duration(i) * time.Minute), Samples: 12, UpSamples: 12}
	}
	return out
}

func TestCoverageFindsGapAndPowerCut(t *testing.T) {
	base := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	resume := base.Add(3 * time.Hour)
	samples := append(minutes(base, 60), minutes(resume, 60)...)
	now := resume.Add(time.Hour)
	events := []types.Event{
		ev(resume.Add(20*time.Second), "disconnected", 0, true, true, true),
		ev(resume.Add(3*time.Minute), "connected", 160, false, false, false),
	}

	s := Summarize(events, samples, "2026-05", now)

	if len(s.Gaps) != 1 {
		t.Fatalf("Gaps = %+v, want one", s.Gaps)
	}
	g := s.Gaps[0]
	if g.DurationSecs != 2*3600 || g.Kind != GapPower {
		t.Errorf("gap = %+v, want 2h of kind power", g)
	}
	if len(s.Events) != 1 || !s.Events[0].AfterGap {
		t.Errorf("Events = %+v, want the outage flagged after_gap", s.Events)
	}
	if s.WindowSeconds != 4*3600 || s.UnmonitoredSecs != 2*3600 {
		t.Errorf("window=%v unmonitored=%v", s.WindowSeconds, s.UnmonitoredSecs)
	}
	if math.Abs(s.CoveragePct-50) > 0.01 {
		t.Errorf("CoveragePct = %v, want 50", s.CoveragePct)
	}
}

func TestCoverageIgnoresShortRestartAndKeepsRealOutage(t *testing.T) {
	base := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	samples := append(minutes(base, 30), minutes(base.Add(32*time.Minute), 30)...)
	now := base.Add(62 * time.Minute)
	events := []types.Event{
		ev(base.Add(40*time.Minute), "disconnected", 0, true, false, false),
		ev(base.Add(45*time.Minute), "connected", 300, false, false, false),
	}
	s := Summarize(events, samples, "2026-05", now)
	if len(s.Gaps) != 0 {
		t.Errorf("Gaps = %+v, want none", s.Gaps)
	}
	if len(s.Events) != 1 || s.Events[0].AfterGap {
		t.Errorf("Events = %+v, want one regular outage", s.Events)
	}
	if s.CoveragePct != 100 {
		t.Errorf("CoveragePct = %v, want 100", s.CoveragePct)
	}
}

func TestCoverageTrailingGapUntilMonthEnd(t *testing.T) {
	last := time.Date(2026, 5, 31, 23, 0, 0, 0, time.UTC)
	samples := minutes(last, 10)
	s := Summarize(nil, samples, "2026-05", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	if len(s.Gaps) != 1 || s.Gaps[0].Kind != GapOff {
		t.Fatalf("Gaps = %+v, want one trailing off gap", s.Gaps)
	}
	if s.Gaps[0].DurationSecs != 50*60 {
		t.Errorf("trailing gap = %v, want 3000s", s.Gaps[0].DurationSecs)
	}
}

func TestSummarizeSpeed(t *testing.T) {
	up := 9.0
	at := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	tests := []SpeedSample{
		{Timestamp: at, DownloadMbps: 40, UploadMbps: &up, LatencyMs: 20},
		{Timestamp: at, DownloadMbps: 20, LatencyMs: 40},
		{Timestamp: at, DownloadMbps: 60, UploadMbps: &up, LatencyMs: 30},
		{Timestamp: at.AddDate(0, 1, 0), DownloadMbps: 100},
		{Timestamp: at, DownloadMbps: 0},
	}
	s := SummarizeSpeed(tests, "2026-05", 100)
	if s.Tests != 3 || s.MedianDownload != 40 || s.MinDownloadMbps != 20 || s.MaxDownloadMbps != 60 {
		t.Fatalf("summary = %+v", s)
	}
	if s.AvgDownloadMbps != 40 || s.AvgUploadMbps != 9 || s.AvgLatencyMs != 30 {
		t.Errorf("averages = %+v", s)
	}
	if s.PctOfPlan != 40 || s.BelowHalfPlan != 2 {
		t.Errorf("plan comparison = %+v", s)
	}
	if none := SummarizeSpeed(tests, "2026-05", 0); none.PctOfPlan != 0 || none.BelowHalfPlan != 0 {
		t.Errorf("no plan: %+v", none)
	}
}
