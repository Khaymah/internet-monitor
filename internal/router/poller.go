package router

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"
)

const (
	pollEvery     = time.Minute
	rediscover    = 10 * time.Minute
	maxFailures   = 3
	discoverWait  = 3 * time.Second
	statusTimeout = 10 * time.Second
	webEvery      = 5 * time.Minute
	webMinEvery   = time.Minute
	DefaultHost   = "192.168.1.1"
)

type Poller struct {
	Enabled  func() bool
	Web      func() (WebReader, bool)
	Save     func(at time.Time, s Status)
	Logf     func(format string, args ...any)
	Discover func(ctx context.Context) (string, error)
	Open     func(ctx context.Context, location string) (*Gateway, error)
	ReadWeb  func(ctx context.Context, r WebReader) (WebStats, error)
	Now      func() time.Time

	mu         sync.Mutex
	gw         *Gateway
	last       Status
	found      bool
	failures   int
	nextProbe  time.Time
	lastErr    string
	webBlocked string
	webErr     string
	webOK      bool
	nextWeb    time.Time
}

type Snapshot struct {
	Found    bool    `json:"found"`
	Status   *Status `json:"status,omitempty"`
	Error    string  `json:"error,omitempty"`
	WebOK    bool    `json:"web_ok,omitempty"`
	WebError string  `json:"web_error,omitempty"`
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
	out := Snapshot{Found: p.gw != nil, Error: p.lastErr, WebOK: p.webOK, WebError: p.webErr}
	if p.found {
		s := p.last
		out.Status = &s
	}
	return out
}

func (p *Poller) Tick(ctx context.Context) {
	if p.Enabled != nil && !p.Enabled() {
		p.mu.Lock()
		p.gw, p.found, p.lastErr, p.webOK, p.webErr = nil, false, "", false, ""
		p.mu.Unlock()
		return
	}
	now := p.now()
	st, upnp := p.readUPnP(ctx, now)
	web := p.readWeb(ctx, now, &st)
	if !upnp && !web {
		return
	}
	st.Time = now
	p.mu.Lock()
	p.last, p.found = st, true
	p.mu.Unlock()
	if p.Save != nil {
		p.Save(now, st)
	}
}

func (p *Poller) readUPnP(ctx context.Context, now time.Time) (Status, bool) {
	p.mu.Lock()
	gw := p.gw
	waiting := gw == nil && now.Before(p.nextProbe)
	p.mu.Unlock()
	if waiting {
		return Status{}, false
	}
	if gw == nil {
		if gw = p.connect(ctx, now); gw == nil {
			return Status{}, false
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
		return Status{Model: gw.Model}, false
	}
	p.failures, p.lastErr = 0, ""
	return st, true
}

func webKey(r WebReader) string { return r.Host + "\x00" + r.User + "\x00" + r.Password }

func (p *Poller) webTarget() (WebReader, bool) {
	if p.Web == nil {
		return WebReader{}, false
	}
	r, ok := p.Web()
	if !ok || r.Password == "" {
		return WebReader{}, false
	}
	if r.Host == "" {
		r.Host = p.gatewayHost()
	}
	return r, true
}

func (p *Poller) gatewayHost() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gw != nil {
		if u, err := url.Parse(p.gw.Location); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	return DefaultHost
}

func webCode(err error) string {
	switch {
	case errors.Is(err, ErrWrongPassword):
		return "wrong_password"
	case errors.Is(err, ErrRouterBusy):
		return "busy"
	case errors.Is(err, ErrWebUnknown):
		return "unsupported"
	default:
		return "unreachable"
	}
}

func webInterval(downKbps int64) time.Duration {
	if downKbps <= 0 {
		return webEvery
	}
	wrap := time.Duration(float64(counterWrap*8) / float64(downKbps*1000) * float64(time.Second))
	return min(webEvery, max(webMinEvery, wrap/2))
}

func (p *Poller) readWeb(ctx context.Context, now time.Time, st *Status) bool {
	r, ok := p.webTarget()
	p.mu.Lock()
	if !ok {
		p.webOK, p.webErr = false, ""
		p.mu.Unlock()
		return false
	}
	if p.webBlocked == webKey(r) || now.Before(p.nextWeb) {
		p.mu.Unlock()
		return false
	}
	p.nextWeb = now.Add(webEvery)
	p.mu.Unlock()
	stats, err := p.fetch(ctx, r)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		code := webCode(err)
		if code != p.webErr {
			p.logf("ROUTER page: %v", err)
		}
		p.webOK, p.webErr = false, code
		if code == "wrong_password" || code == "unsupported" {
			p.webBlocked = webKey(r)
		}
		return false
	}
	p.webOK, p.webErr = true, ""
	p.nextWeb = now.Add(webInterval(stats.DownKbps))
	merge(st, stats)
	return true
}

func (p *Poller) fetch(ctx context.Context, r WebReader) (WebStats, error) {
	read := p.ReadWeb
	if read == nil {
		read = func(ctx context.Context, r WebReader) (WebStats, error) { return r.Read(ctx) }
	}
	wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return read(wctx, r)
}

func (p *Poller) TestWeb(ctx context.Context, r WebReader) (WebStats, error) {
	if r.Password == "" {
		t, ok := p.webTarget()
		if !ok {
			return WebStats{}, ErrWrongPassword
		}
		r.Password, r.User = t.Password, t.User
	}
	if r.Host == "" {
		r.Host = p.gatewayHost()
	}
	stats, err := p.fetch(ctx, r)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		p.webBlocked, p.webErr, p.nextWeb = "", "", time.Time{}
	}
	return stats, err
}

func f64(v float64) *float64 { return &v }

func merge(st *Status, w WebStats) {
	if w.HasLineStats {
		if w.DownKbps > 0 {
			st.SyncDownKbps, st.SyncUpKbps = w.DownKbps, w.UpKbps
		}
		st.MaxDownKbps, st.MaxUpKbps = w.MaxDownKbps, w.MaxUpKbps
		st.SNRDown, st.SNRUp = f64(w.SNRDown), f64(w.SNRUp)
		st.AttnDown, st.AttnUp = f64(w.AttnDown), f64(w.AttnUp)
		st.LineUpSecs = w.LineUpSecs
		if st.LinkUp == nil && w.Status != "" {
			st.LinkUp = boolPtr(w.Status == "Up")
		}
	}
	if w.HasCounters {
		st.RxBytes, st.TxBytes, st.HasCounters = w.RxBytes, w.TxBytes, true
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
