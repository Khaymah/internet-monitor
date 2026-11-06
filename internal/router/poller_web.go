package router

import (
	"context"
	"errors"
	"net/url"
	"time"
)

func webKey(r WebReader) string { return r.Host + "\x00" + r.User + "\x00" + r.Password }

func (p *Poller) webTarget() (WebReader, bool) {
	if p.Web == nil {
		return WebReader{}, false
	}
	r, ok := p.Web()
	if !ok {
		return WebReader{}, false
	}
	if r.Host == "" {
		host, found := p.gatewayAddr()
		if !found && r.Password == "" {
			return WebReader{}, false
		}
		r.Host = host
	}
	return r, true
}

func (p *Poller) gatewayAddr() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gw != nil {
		if u, err := url.Parse(p.gw.Location); err == nil && u.Hostname() != "" {
			return u.Hostname(), true
		}
	}
	return DefaultHost, false
}

func (p *Poller) gatewayHost() string {
	host, _ := p.gatewayAddr()
	return host
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
		p.failWeb(r, err)
		if p.webCounts {
			st.RxBytes, st.TxBytes, st.HasCounters = 0, 0, false
		}
		return false
	}
	p.webOK, p.webErr, p.webCounts = true, "", stats.HasCounters
	p.nextWeb = now.Add(webInterval(stats.DownKbps))
	merge(st, stats)
	return true
}

func (p *Poller) failWeb(r WebReader, err error) {
	p.webOK = false
	if r.Password == "" {
		p.webErr = ""
		if errors.Is(err, ErrPasswordRequired) {
			p.webBlocked = webKey(r)
		}
		return
	}
	code := webCode(err)
	if code != p.webErr {
		p.logf("ROUTER page: %v", err)
	}
	p.webErr = code
	if code == "wrong_password" || code == "unsupported" {
		p.webBlocked = webKey(r)
	}
}

func (p *Poller) drivers() []Driver {
	if p.Drivers != nil {
		return p.Drivers
	}
	return Drivers()
}

func (p *Poller) cachedDriver(host string, anon bool) Driver {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.drv != nil && (p.drvHost != host || p.drvAnon != anon) {
		p.drv, p.drvFails = nil, 0
	}
	return p.drv
}

func (p *Poller) noteDriver(d Driver, host string, anon bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.drv == nil {
		p.drv, p.drvHost, p.drvAnon, p.drvFails = d, host, anon, 0
		p.logf("ROUTER page driver: %s", d.Name())
	}
	switch {
	case err == nil:
		p.drvFails = 0
	case errors.Is(err, ErrWrongPassword), errors.Is(err, ErrRouterBusy), errors.Is(err, ErrPasswordRequired):
	default:
		p.drvFails++
		if p.drvFails >= maxFailures {
			p.logf("ROUTER page driver %s dropped after %d failed reads", d.Name(), p.drvFails)
			p.drv, p.drvFails = nil, 0
		}
	}
}

func (p *Poller) fetch(ctx context.Context, r WebReader) (WebStats, error) {
	wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if p.ReadWeb != nil {
		return p.ReadWeb(wctx, r)
	}
	probe, err := NewProbe(r.Host, r.Timeout)
	if err != nil {
		return WebStats{}, err
	}
	anon := r.Password == ""
	d := p.cachedDriver(r.Host, anon)
	if d == nil {
		if d, err = Detect(wctx, probe, p.drivers(), anon); err != nil {
			return WebStats{}, err
		}
	}
	stats, err := d.Read(wctx, probe, Creds{User: r.User, Password: r.Password})
	stats.Driver = d.Name()
	p.noteDriver(d, r.Host, anon, err)
	return stats, err
}

func (p *Poller) TestWeb(ctx context.Context, r WebReader) (WebStats, error) {
	if r.Password == "" {
		t, ok := p.webTarget()
		if !ok || t.Password == "" {
			return WebStats{}, ErrWrongPassword
		}
		r.Password, r.User = t.Password, t.User
	}
	if r.Host == "" {
		r.Host = p.gatewayHost()
	}
	p.mu.Lock()
	p.drv, p.drvFails = nil, 0
	p.mu.Unlock()
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
		if w.LineUpSecs > 0 {
			st.LineUpSecs = w.LineUpSecs
		}
		if st.LinkUp == nil && w.Status != "" {
			st.LinkUp = boolPtr(w.Status == "Up")
		}
	}
	if w.HasCounters {
		st.RxBytes, st.TxBytes, st.HasCounters = w.RxBytes, w.TxBytes, true
	}
	if st.Model == "" {
		st.Model = w.Model
	}
	if st.Connected == nil && w.Connected != nil {
		st.Connected = w.Connected
	}
	if st.UptimeSecs == 0 {
		st.UptimeSecs = w.UptimeSecs
	}
	if st.LastError == "" {
		st.LastError = w.LastError
	}
}
