package weekly

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const quotaFile = "quota-state.json"

var quotaSteps = []int{80, 100}

type QuotaReading struct {
	Period   time.Time
	UsedPct  float64
	UsedGB   float64
	QuotaGB  float64
	DaysLeft int
}

type QuotaWatcher struct {
	State string
	Read  func(now time.Time) (QuotaReading, bool)
	Send  func(step int, r QuotaReading)
	Now   func() time.Time
	Every time.Duration
	mu    sync.Mutex
}

type quotaState struct {
	Period time.Time `json:"period"`
	Sent   int       `json:"sent"`
}

func QuotaPath(appDir string) string { return filepath.Join(appDir, quotaFile) }

func (q *QuotaWatcher) load() quotaState {
	var st quotaState
	if data, err := os.ReadFile(q.State); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	return st
}

func (q *QuotaWatcher) Tick() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	r, ok := q.Read(q.Now())
	if !ok || r.QuotaGB <= 0 {
		return 0
	}
	st := q.load()
	if !st.Period.Equal(r.Period) {
		st = quotaState{Period: r.Period}
	}
	step := 0
	for _, s := range quotaSteps {
		if r.UsedPct >= float64(s) && s > st.Sent {
			step = s
		}
	}
	if step == 0 {
		return 0
	}
	st.Sent = step
	data, _ := json.Marshal(st)
	_ = os.WriteFile(q.State, data, 0o600)
	q.Send(step, r)
	return step
}

func (q *QuotaWatcher) Run(stop <-chan struct{}) {
	every := q.Every
	if every <= 0 {
		every = 10 * time.Minute
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			q.Tick()
		case <-stop:
			return
		}
	}
}
