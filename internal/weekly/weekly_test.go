package weekly

import (
	"encoding/json"
	"internet-monitor/internal/types"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLines(t *testing.T, path string, vs ...any) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, v := range vs {
		b, _ := json.Marshal(v)
		if _, err := f.Write(append(b, '\n')); err != nil {
			t.Fatal(err)
		}
	}
}

func seed(t *testing.T, dir string, from, to time.Time) {
	t.Helper()
	for m := from; m.Before(to); m = m.Add(time.Minute) {
		writeLines(t, filepath.Join(dir, "metrics_"+m.Format("2006-01-02")+".jsonl"),
			types.MetricSample{Timestamp: m, Samples: 12, UpSamples: 12})
	}
	down := from.Add(26 * time.Hour)
	writeLines(t, filepath.Join(dir, "connectivity_"+down.Format("2006-01-02")+".jsonl"),
		types.Event{Timestamp: down, EventType: "disconnected", Reason: types.EventReason{TCPPingFailed: true}},
		types.Event{Timestamp: down.Add(5 * time.Minute), EventType: "connected", DurationSeconds: 300},
	)
}

func TestComputeAndScheduleOncePerWeek(t *testing.T) {
	dir := t.TempDir()
	monday := time.Date(2026, 11, 9, 9, 30, 0, 0, time.Local)
	seed(t, dir, time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local), time.Date(2026, 11, 9, 0, 0, 0, 0, time.Local))

	var sent []types.WeeklySummary
	now := monday
	s := &Scheduler{
		Dir:   func() string { return dir },
		State: filepath.Join(dir, "state.json"),
		Plan:  func() float64 { return 70 },
		Send:  func(w types.WeeklySummary) { sent = append(sent, w) },
		Now:   func() time.Time { return now },
	}

	if !s.Tick() {
		t.Fatal("Monday 09:30 should send the summary")
	}
	if s.Tick() {
		t.Fatal("a second tick in the same week must not send again")
	}
	if len(sent) != 1 || sent[0].Outages != 1 || sent[0].DowntimeSeconds != 300 || sent[0].PlanMbps != 70 {
		t.Fatalf("sent = %+v", sent)
	}

	restarted := &Scheduler{Dir: s.Dir, State: s.State, Send: s.Send, Now: s.Now}
	if restarted.Tick() {
		t.Fatal("after a restart the same week must not be sent again")
	}

	now = monday.Add(-2 * time.Hour)
	other := &Scheduler{Dir: s.Dir, State: filepath.Join(dir, "other.json"), Send: s.Send, Now: func() time.Time { return now }}
	if other.Tick() {
		t.Fatal("before 09:00 on Monday nothing is sent")
	}
}

func TestComputeSkipsWhenNothingWasMonitored(t *testing.T) {
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.Local)
	if _, ok := Compute(t.TempDir(), now.Add(-Week), now, now, 0); ok {
		t.Fatal("no data must not produce a summary")
	}
}
