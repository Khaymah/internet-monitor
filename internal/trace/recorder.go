package trace

import (
	"context"
	"internet-monitor/internal/types"
	"net"
	"sync"
	"time"
)

const (
	Prefix   = "traceroute_"
	cooldown = 10 * time.Minute
)

type Recorder struct {
	Target func() string
	Save   func(at time.Time, t Trace)
	Logf   func(format string, args ...interface{})
	Run    func(ctx context.Context, target string) (Trace, error)
	Now    func() time.Time

	mu      sync.Mutex
	last    time.Time
	running bool
	wg      sync.WaitGroup
}

func (r *Recorder) OnTick(types.CheckResult, types.Status) {}

func (r *Recorder) OnInitialEvent(ev types.Event) { r.OnEvent(ev) }

func (r *Recorder) OnEvent(ev types.Event) {
	if ev.EventType != types.StatusDisconnected.String() {
		return
	}
	now := r.now()
	r.mu.Lock()
	if r.running || (!r.last.IsZero() && now.Sub(r.last) < cooldown) {
		r.mu.Unlock()
		return
	}
	r.running, r.last = true, now
	r.mu.Unlock()

	target := HostOf(r.Target())
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
		}()
		run := r.Run
		if run == nil {
			run = Run
		}
		t, err := run(context.Background(), target)
		if err != nil {
			if r.Logf != nil {
				r.Logf("TRACEROUTE %s: %v", target, err)
			}
			return
		}
		t.Time = r.now()
		t.EventTime = ev.Timestamp
		if r.Save != nil {
			r.Save(ev.Timestamp, t)
		}
	}()
}

func (r *Recorder) Wait() { r.wg.Wait() }

func (r *Recorder) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func HostOf(target string) string {
	if target == "" {
		return "8.8.8.8"
	}
	if h, _, err := net.SplitHostPort(target); err == nil {
		return h
	}
	return target
}
