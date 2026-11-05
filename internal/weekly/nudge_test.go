package weekly

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNudgeOncePerWeek(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	sent := 0
	due := false
	n := &Nudger{
		State: filepath.Join(t.TempDir(), "nudge.json"),
		Check: func(int) (int, bool) { return 4, due },
		Send:  func(int, int) { sent++ },
		Now:   func() time.Time { return now },
	}
	if n.Tick() {
		t.Fatal("nothing to nudge about yet")
	}
	due = true
	if !n.Tick() || sent != 1 {
		t.Fatal("first due check must nudge")
	}
	now = now.Add(3 * 24 * time.Hour)
	if n.Tick() {
		t.Fatal("must wait a week before the next nudge")
	}
	now = now.Add(5 * 24 * time.Hour)
	if !n.Tick() || sent != 2 {
		t.Fatal("a week later it may nudge again")
	}
}
