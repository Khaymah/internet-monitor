package weekly

import (
	"encoding/json"
	"fmt"
	"internet-monitor/internal/jsonl"
	"internet-monitor/internal/report"
	"internet-monitor/internal/types"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	Week      = 7 * 24 * time.Hour
	sendHour  = 9
	stateFile = "weekly-summary.json"
)

func Compute(dir string, from, to, now time.Time, planMbps float64) (types.WeeklySummary, bool) {
	events := jsonl.DecodeDays[types.Event](dir, "connectivity_", from.AddDate(0, 0, -1), to.AddDate(0, 0, 1))
	samples := jsonl.DecodeDays[types.MetricSample](dir, "metrics_", from, to)
	sum := report.SummarizeRange(events, samples, from, to, now)
	if sum.WindowSeconds <= 0 {
		return types.WeeklySummary{}, false
	}
	var speeds []report.SpeedSample
	for _, sp := range jsonl.DecodeDays[report.SpeedSample](dir, "speedtest_", from, to) {
		if !sp.Timestamp.Before(from) && sp.Timestamp.Before(to) {
			speeds = append(speeds, sp)
		}
	}
	speed := report.SummarizeSpeed(speeds, "", planMbps)
	out := types.WeeklySummary{
		From:            from,
		To:              to,
		Outages:         sum.Disconnections,
		DowntimeSeconds: sum.TotalDowntimeSecs,
		LongestSeconds:  sum.LongestOutageSecs,
		UptimePct:       sum.UptimePct,
		CoveragePct:     sum.CoveragePct,
		SpeedTests:      speed.Tests,
		MedianMbps:      speed.MedianDownload,
		PlanMbps:        speed.PlanDownloadMbps,
		PctOfPlan:       speed.PctOfPlan,
	}
	prevEvents := jsonl.DecodeDays[types.Event](dir, "connectivity_", from.Add(-Week).AddDate(0, 0, -1), from)
	prevSamples := jsonl.DecodeDays[types.MetricSample](dir, "metrics_", from.Add(-Week), from)
	if prev := report.SummarizeRange(prevEvents, prevSamples, from.Add(-Week), from, now); prev.WindowSeconds > 0 {
		n := prev.Disconnections
		out.PrevOutages = &n
	}
	return out, true
}

type Scheduler struct {
	Dir     func() string
	State   string
	Plan    func() float64
	Send    func(types.WeeklySummary)
	Logf    func(format string, args ...interface{})
	Now     func() time.Time
	Every   time.Duration
	mu      sync.Mutex
	lastKey string
}

type state struct {
	LastWeek string `json:"last_week"`
}

func weekKey(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func StatePath(appDir string) string { return filepath.Join(appDir, stateFile) }

func (s *Scheduler) load() string {
	data, err := os.ReadFile(s.State)
	if err != nil {
		return ""
	}
	var st state
	if json.Unmarshal(data, &st) != nil {
		return ""
	}
	return st.LastWeek
}

func (s *Scheduler) save(key string) {
	data, _ := json.Marshal(state{LastWeek: key})
	if err := os.WriteFile(s.State, data, 0o600); err != nil && s.Logf != nil {
		s.Logf("weekly summary state: %v", err)
	}
}

func (s *Scheduler) Due(now time.Time) bool {
	return now.Weekday() == time.Monday && now.Hour() >= sendHour
}

func (s *Scheduler) Tick() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.Now()
	if !s.Due(now) {
		return false
	}
	key := weekKey(now)
	if s.lastKey == "" {
		s.lastKey = s.load()
	}
	if s.lastKey == key {
		return false
	}
	s.lastKey = key
	s.save(key)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sum, ok := Compute(s.Dir(), to.AddDate(0, 0, -7), to, now, s.plan())
	if !ok {
		return false
	}
	s.Send(sum)
	return true
}

func (s *Scheduler) SendNow() bool {
	now := s.Now()
	sum, ok := Compute(s.Dir(), now.Add(-Week), now, now, s.plan())
	if ok {
		s.Send(sum)
	}
	return ok
}

func (s *Scheduler) plan() float64 {
	if s.Plan == nil {
		return 0
	}
	return s.Plan()
}

func (s *Scheduler) Run(stop <-chan struct{}) {
	every := s.Every
	if every <= 0 {
		every = 10 * time.Minute
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		s.Tick()
		select {
		case <-t.C:
		case <-stop:
			return
		}
	}
}
