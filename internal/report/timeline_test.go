package report

import (
	"internet-monitor/internal/types"
	"testing"
	"time"
)

func TestTimelineSegments(t *testing.T) {
	from := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	to := from.Add(4 * time.Hour)
	samples := append(minutes(from, 120), minutes(from.Add(3*time.Hour), 60)...)
	events := []types.Event{
		ev(from.Add(-time.Hour), "connected", 0, false, false, false),
		ev(from.Add(30*time.Minute), "disconnected", 0, true, false, false),
		ev(from.Add(40*time.Minute), "connected", 600, false, false, false),
		ev(from.Add(3*time.Hour+30*time.Minute), "degraded", 0, false, false, false),
	}

	got := Timeline(events, samples, from, to)
	want := []struct {
		status     string
		start, end time.Duration
	}{
		{"connected", 0, 30 * time.Minute},
		{"disconnected", 30 * time.Minute, 40 * time.Minute},
		{"connected", 40 * time.Minute, 2 * time.Hour},
		{SegmentOff, 2 * time.Hour, 3 * time.Hour},
		{"connected", 3 * time.Hour, 3*time.Hour + 30*time.Minute},
		{"degraded", 3*time.Hour + 30*time.Minute, 4 * time.Hour},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d segments: %+v", len(got), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Status != w.status || !g.Start.Equal(from.Add(w.start)) || !g.End.Equal(from.Add(w.end)) {
			t.Errorf("segment %d = %s %s–%s, want %s %v–%v", i, g.Status, g.Start.Sub(from), g.End.Sub(from), w.status, w.start, w.end)
		}
	}
}

func TestTimelineWithoutDataIsAllOff(t *testing.T) {
	from := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	got := Timeline(nil, nil, from, from.Add(time.Hour))
	if len(got) != 1 || got[0].Status != SegmentOff {
		t.Fatalf("got %+v", got)
	}
}
