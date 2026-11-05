package router

import (
	"context"
	"sync"
	"time"
)

const (
	pollEvery     = time.Minute
	rediscover    = 10 * time.Minute
	maxFailures   = 3
	discoverWait  = 3 * time.Second
	statusTimeout = 10 * time.Second
)

type Poller struct {
	Enabled  func() bool
	Save     func(at time.Time, s Status)
	Logf     func(format string, args ...any)
	Discover func(ctx context.Context) (string, error)
	Open     func(ctx context.Context, location string) (*Gateway, error)
	Now      func() time.Time

	mu        sync.Mutex
	gw        *Gateway
	last      Status
	found     bool
	failures  int
	nextProbe time.Time
	lastErr   string
}

type Snapshot struct {
	Found  bool    `json:"found"`
	Status *Status `json:"status,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func (p *Poller) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func (p *Poller) logf(format string, args ...any) {
	if p.Logf != nil {
		p.Logf(format, args...)
	}
}

func (p *Poller) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := Snapshot{Found: p.gw != nil, Error: p.lastErr}
	if p.found {
		s := p.last
		out.Status = &s
	}
	return out
}

func (p *Poller) Tick(ctx context.Context) {
	if p.Enabled != nil && !p.Enabled() {
		p.mu.Lock()
		p.gw, p.found, p.lastErr = nil, false, ""
		p.mu.Unlock()
		return
	}
	now := p.now()
	p.mu.Lock()
	gw := p.gw
	if gw == nil && now.Before(p.nextProbe) {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	if gw == nil {
		gw = p.connect(ctx, now)
		if gw == nil {
			return
		}
	}
	sctx, cancel := context.WithTimeout(ctx, statusTimeout)
	st, err := gw.Status(sctx)
	cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.failures++
		p.lastErr = err.Error()
		if p.failures >= maxFailures {
			p.logf("ROUTER lost after %d failed reads: %v", p.failures, err)
			p.gw, p.failures, p.nextProbe = nil, 0, now.Add(rediscover)
		}
		return
	}
	p.failures, p.lastErr = 0, ""
	st.Time = now
	p.last, p.found = st, true
	if p.Save != nil {
		p.Save(now, st)
	}
}

func (p *Poller) connect(ctx context.Context, now time.Time) *Gateway {
	discover := p.Discover
	if discover == nil {
		discover = func(ctx context.Context) (string, error) { return Discover(ctx, discoverWait) }
	}
	open := p.Open
	if open == nil {
		open = func(ctx context.Context, loc string) (*Gateway, error) { return Open(ctx, loc, nil) }
	}
	loc, err := discover(ctx)
	var gw *Gateway
	if err == nil {
		gw, err = open(ctx, loc)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.lastErr = err.Error()
		p.nextProbe = now.Add(rediscover)
		return nil
	}
	p.gw, p.lastErr = gw, ""
	p.logf("ROUTER found %q at %s", gw.Model, loc)
	return gw
}

func (p *Poller) Run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-stop
		cancel()
	}()
	p.Tick(ctx)
	t := time.NewTicker(pollEvery)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.Tick(ctx)
		case <-stop:
			return
		}
	}
}
