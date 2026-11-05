package weekly

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQuotaWatcherSendsEachStepOncePerPeriod(t *testing.T) {
	period := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	reading := QuotaReading{Period: period, QuotaGB: 140}
	var sent []int
	q := &QuotaWatcher{
		State: filepath.Join(t.TempDir(), quotaFile),
		Read:  func(time.Time) (QuotaReading, bool) { return reading, true },
		Send:  func(step int, _ QuotaReading) { sent = append(sent, step) },
		Now:   time.Now,
	}
	for _, pct := range []float64{50, 81, 85, 99, 100, 120} {
		reading.UsedPct = pct
		q.Tick()
	}
	if len(sent) != 2 || sent[0] != 80 || sent[1] != 100 {
		t.Fatalf("sent %v", sent)
	}
	reading.Period = period.AddDate(0, 1, 0)
	reading.UsedPct = 130
	q.Tick()
	if len(sent) != 3 || sent[2] != 100 {
		t.Fatalf("a new period must alert again, once, at the highest step: %v", sent)
	}
	reading.QuotaGB = 0
	reading.Period = period.AddDate(0, 2, 0)
	if q.Tick() != 0 {
		t.Fatal("no alert without a quota")
	}
}
