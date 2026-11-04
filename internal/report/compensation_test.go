package report

import (
	"internet-monitor/internal/types"
	"testing"
	"time"
)

func TestHoursAndPeakShare(t *testing.T) {
	base := time.Date(2026, 5, 10, 20, 30, 0, 0, time.UTC)
	events := []types.Event{
		ev(base, "disconnected", 0, true, false, false),
		ev(base.Add(time.Hour), "connected", 3600, false, false, false),
		ev(base.Add(-12*time.Hour), "disconnected", 0, true, false, false),
		ev(base.Add(-12*time.Hour+20*time.Minute), "connected", 1200, false, false, false),
	}
	s := Summarize(events, nil, "2026-05", base.Add(2*time.Hour))
	if s.Hours[20].DowntimeSecs != 1800 || s.Hours[21].DowntimeSecs != 1800 || s.Hours[8].DowntimeSecs != 1200 {
		t.Fatalf("hours = %+v", s.Hours)
	}
	if s.Hours[20].Outages != 1 || s.Hours[8].Outages != 1 || s.Hours[21].Outages != 0 {
		t.Fatalf("outage starts by hour wrong: %+v", s.Hours)
	}
	if got := s.PeakSharePct; got < 74.9 || got > 75.1 {
		t.Fatalf("peak share = %v, want 75", got)
	}
}

func TestCompensate(t *testing.T) {
	sum := MonthlySummary{
		Month:             "2026-05",
		TotalDowntimeSecs: 3*86400 + 3600,
		Events: []EventRow{
			{Type: "disconnected", DurationSecs: 3 * 86400},
			{Type: "disconnected", DurationSecs: 1800, OnBattery: true},
			{Type: "disconnected", DurationSecs: 1800},
		},
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	c := Compensate(sum, 310, "EGP", "EG", now)
	if c.ExcludedSeconds != 1800 || c.ProviderDowntime != 3*86400+1800 {
		t.Fatalf("downtime split = %+v", c)
	}
	want := round2(310 * float64(3*86400+1800) / (31 * 86400))
	if c.ProRata != want || c.OfcomDays != 0 {
		t.Fatalf("pro-rata = %v (want %v), ofcom days %d", c.ProRata, want, c.OfcomDays)
	}
	uk := Compensate(sum, 40, "GBP", "GB", now)
	if uk.OfcomDays != 1 || uk.OfcomAmountGBP != round2(OfcomDailyGBP) {
		t.Fatalf("ofcom = %d days / %v", uk.OfcomDays, uk.OfcomAmountGBP)
	}
	if none := Compensate(sum, 0, "", "EG", now); none.ProRata != 0 {
		t.Fatal("no price means no amount")
	}
}

func TestSpeedPeakSplit(t *testing.T) {
	day := time.Date(2026, 5, 3, 0, 0, 0, 0, time.Local)
	tests := []SpeedSample{
		{Timestamp: day.Add(10 * time.Hour), DownloadMbps: 60},
		{Timestamp: day.Add(11 * time.Hour), DownloadMbps: 50},
		{Timestamp: day.Add(21 * time.Hour), DownloadMbps: 12},
		{Timestamp: day.Add(22 * time.Hour), DownloadMbps: 18},
	}
	s := SummarizeSpeed(tests, "2026-05", 70)
	if s.PeakTests != 2 || s.PeakMedian != 15 || s.OffPeakMedian != 55 {
		t.Fatalf("peak split = %+v", s)
	}
}
