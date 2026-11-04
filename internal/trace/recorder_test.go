package trace

import (
	"context"
	"internet-monitor/internal/types"
	"sync"
	"testing"
	"time"
)

func TestRecorderTracesOutagesWithCooldown(t *testing.T) {
	var mu sync.Mutex
	var saved []Trace
	var targets []string
	now := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	r := &Recorder{
		Target: func() string { return "1.1.1.1:53" },
		Now:    func() time.Time { return now },
		Run: func(_ context.Context, target string) (Trace, error) {
			mu.Lock()
			targets = append(targets, target)
			mu.Unlock()
			return Parse(" 1  192.168.1.1  0.5 ms\n 2  *\n", target), nil
		},
		Save: func(_ time.Time, tr Trace) {
			mu.Lock()
			saved = append(saved, tr)
			mu.Unlock()
		},
	}
	down := types.Event{Timestamp: now, EventType: "disconnected"}

	r.OnEvent(types.Event{Timestamp: now, EventType: "connected"})
	r.OnEvent(down)
	r.Wait()
	now = now.Add(5 * time.Minute)
	r.OnEvent(down)
	r.Wait()
	now = now.Add(6 * time.Minute)
	r.OnEvent(types.Event{Timestamp: now, EventType: "disconnected"})
	r.Wait()

	if len(saved) != 2 {
		t.Fatalf("saved %d traces, want 2 (one skipped by the 10-minute cooldown)", len(saved))
	}
	if targets[0] != "1.1.1.1" || saved[0].LastAddr != "192.168.1.1" || saved[0].EventTime.IsZero() {
		t.Fatalf("first trace = %+v (target %q)", saved[0], targets[0])
	}
}

func TestHostOf(t *testing.T) {
	for in, want := range map[string]string{"8.8.8.8:53": "8.8.8.8", "1.1.1.1": "1.1.1.1", "": "8.8.8.8", "[2606:4700::1111]:53": "2606:4700::1111"} {
		if got := HostOf(in); got != want {
			t.Errorf("HostOf(%q) = %q, want %q", in, got, want)
		}
	}
}
