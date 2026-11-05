package weekly

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	nudgeFile  = "nudge-state.json"
	NudgeEvery = 7 * 24 * time.Hour
	nudgeDays  = 7
)

type Nudger struct {
	State string
	Check func(days int) (outages int, due bool)
	Send  func(outages, days int)
	Now   func() time.Time
	Every time.Duration
	mu    sync.Mutex
}

func NudgePath(appDir string) string { return filepath.Join(appDir, nudgeFile) }

func (n *Nudger) last() time.Time {
	var st struct {
		Last time.Time `json:"last"`
	}
	if data, err := os.ReadFile(n.State); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	return st.Last
}

func (n *Nudger) Tick() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := n.Now()
	if last := n.last(); !last.IsZero() && now.Sub(last) < NudgeEvery {
		return false
	}
	outages, due := n.Check(nudgeDays)
	if !due {
		return false
	}
	data, _ := json.Marshal(map[string]time.Time{"last": now})
	_ = os.WriteFile(n.State, data, 0o600)
	n.Send(outages, nudgeDays)
	return true
}

func (n *Nudger) Run(stop <-chan struct{}) {
	every := n.Every
	if every <= 0 {
		every = time.Hour
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			n.Tick()
		case <-stop:
			return
		}
	}
}
