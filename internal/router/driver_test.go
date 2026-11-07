package router

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

type stubDriver struct {
	name     string
	matches  bool
	anon     bool
	detects  int
	reads    int
	readFunc func() (WebStats, error)
}

func (s *stubDriver) Name() string { return s.name }

func (s *stubDriver) Anonymous() bool { return s.anon }

func (s *stubDriver) Detect(context.Context, *Probe) (bool, error) {
	s.detects++
	return s.matches, nil
}

func (s *stubDriver) Read(context.Context, *Probe, Creds) (WebStats, error) {
	s.reads++
	return s.readFunc()
}

func okStats() (WebStats, error) {
	return WebStats{Status: "Up", DownKbps: 1000, HasLineStats: true, HasCounters: true, RxBytes: 1, TxBytes: 2}, nil
}

func stubPoller(clock *time.Time, host, password *string, drivers ...Driver) *Poller {
	return &Poller{
		Discover: func(context.Context) (string, error) { return "", ErrNotFound },
		Web: func() (WebReader, bool) {
			return WebReader{Host: *host, Password: *password}, true
		},
		Drivers: drivers,
		Save:    func(time.Time, Status) {},
		Now:     func() time.Time { return *clock },
	}
}

func advance(p *Poller, clock *time.Time, times int) {
	for range times {
		p.Tick(context.Background())
		*clock = clock.Add(10 * time.Minute)
	}
}

func TestPollerDetectsOnceAndShowsDriver(t *testing.T) {
	a := &stubDriver{name: "a"}
	b := &stubDriver{name: "b", matches: true, readFunc: okStats}
	clock, host, pw := at(0), "127.0.0.1", "x"
	p := stubPoller(&clock, &host, &pw, a, b)
	advance(p, &clock, 3)
	if a.detects != 1 || b.detects != 1 || b.reads != 3 {
		t.Fatalf("detection is cached: a %d b %d reads %d", a.detects, b.detects, b.reads)
	}
	if snap := p.Snapshot(); snap.Driver != "b" || !snap.WebOK {
		t.Fatalf("snapshot %+v", snap)
	}
}

func TestPollerRedetectsAfterRepeatedFailures(t *testing.T) {
	failing := true
	a := &stubDriver{name: "a", matches: true, readFunc: func() (WebStats, error) {
		if failing {
			return WebStats{}, errors.New("timeout")
		}
		return okStats()
	}}
	clock, host, pw := at(0), "127.0.0.1", "x"
	p := stubPoller(&clock, &host, &pw, a)
	advance(p, &clock, maxFailures)
	if a.detects != 1 {
		t.Fatalf("detects %d", a.detects)
	}
	if snap := p.Snapshot(); snap.Driver != "" {
		t.Fatalf("the driver must be forgotten after %d failures: %+v", maxFailures, snap)
	}
	failing = false
	advance(p, &clock, 1)
	if a.detects != 2 || p.Snapshot().Driver != "a" {
		t.Fatalf("detects %d", a.detects)
	}
}

func TestPollerKeepsDriverWhenPasswordRejected(t *testing.T) {
	a := &stubDriver{name: "a", matches: true, readFunc: func() (WebStats, error) { return WebStats{}, ErrWrongPassword }}
	clock, host, pw := at(0), "127.0.0.1", "bad"
	p := stubPoller(&clock, &host, &pw, a)
	advance(p, &clock, 5)
	if a.reads != 1 || a.detects != 1 {
		t.Fatalf("a rejected password is tried once: reads %d detects %d", a.reads, a.detects)
	}
	if snap := p.Snapshot(); snap.WebError != "wrong_password" || snap.Driver != "a" {
		t.Fatalf("snapshot %+v", snap)
	}
}

func TestPollerRedetectsWhenAddressChanges(t *testing.T) {
	a := &stubDriver{name: "a", matches: true, readFunc: okStats}
	clock, host, pw := at(0), "127.0.0.1", "x"
	p := stubPoller(&clock, &host, &pw, a)
	advance(p, &clock, 2)
	host = "127.0.0.2"
	advance(p, &clock, 1)
	if a.detects != 2 {
		t.Fatalf("a new address must be detected again: %d", a.detects)
	}
}

func TestPollerWithoutPasswordOnlyUsesAnonymousDrivers(t *testing.T) {
	web := &stubDriver{name: "web", matches: true, readFunc: okStats}
	open := &stubDriver{name: "open", matches: true, anon: true, readFunc: okStats}
	clock, host, pw := at(0), "127.0.0.1", ""
	p := stubPoller(&clock, &host, &pw, web, open)
	advance(p, &clock, 2)
	if web.detects != 0 || open.reads != 2 {
		t.Fatalf("web %d open reads %d", web.detects, open.reads)
	}
	pw = "now-set"
	advance(p, &clock, 1)
	if web.detects != 1 || p.Snapshot().Driver != "web" {
		t.Fatalf("adding a password must re-detect with every driver: %+v", p.Snapshot())
	}
}

func TestPollerSilentWhenNoPasswordAndRouterNeedsOne(t *testing.T) {
	open := &stubDriver{name: "open", matches: true, anon: true, readFunc: func() (WebStats, error) {
		return WebStats{}, ErrPasswordRequired
	}}
	clock, host, pw := at(0), "127.0.0.1", ""
	p := stubPoller(&clock, &host, &pw, open)
	advance(p, &clock, 4)
	if open.reads != 1 {
		t.Fatalf("must not ask again: %d", open.reads)
	}
	if snap := p.Snapshot(); snap.WebError != "" || snap.WebOK {
		t.Fatalf("no error for a password the user never set: %+v", snap)
	}
}

func TestPollerSkipsGuessedAddressWithoutPassword(t *testing.T) {
	open := &stubDriver{name: "open", matches: true, anon: true, readFunc: okStats}
	clock, host, pw := at(0), "", ""
	p := stubPoller(&clock, &host, &pw, open)
	advance(p, &clock, 2)
	if open.detects != 0 {
		t.Fatal("without a password the app must not probe an address it only guessed")
	}
}

func TestPollerUnsupportedWithPassword(t *testing.T) {
	a := &stubDriver{name: "a"}
	clock, host, pw := at(0), "127.0.0.1", "x"
	p := stubPoller(&clock, &host, &pw, a)
	advance(p, &clock, 3)
	if a.detects != 1 {
		t.Fatalf("an unsupported router is not asked again until the settings change: %d", a.detects)
	}
	if snap := p.Snapshot(); snap.WebError != "unsupported" || snap.Driver != "" {
		t.Fatalf("snapshot %+v", snap)
	}
}

func TestPollerDropsTrafficSampleWhenPageReadFails(t *testing.T) {
	igd := &fakeIGD{link: "Up", conn: "Connected", rx: 10, tx: 20, uptime: 99}
	srv := httptest.NewServer(igd.handler())
	defer srv.Close()
	failing := false
	a := &stubDriver{name: "a", matches: true, readFunc: func() (WebStats, error) {
		if failing {
			return WebStats{}, errors.New("timeout")
		}
		return okStats()
	}}
	clock, host, pw := at(0), "127.0.0.1", "x"
	var saved []Status
	p := stubPoller(&clock, &host, &pw, a)
	p.Discover = func(context.Context) (string, error) { return srv.URL + "/desc.xml", nil }
	p.Save = func(_ time.Time, s Status) { saved = append(saved, s) }
	advance(p, &clock, 1)
	failing = true
	advance(p, &clock, 1)
	if len(saved) != 2 || !saved[0].HasCounters || saved[0].RxBytes != 1 {
		t.Fatalf("saved %+v", saved)
	}
	if saved[1].HasCounters {
		t.Fatalf("UPnP counters must not be mixed with the page's counters: %+v", saved[1])
	}
}

func TestDetectReportsUnreachableRouter(t *testing.T) {
	p, err := NewProbe("127.0.0.1:1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Detect(context.Background(), p, Drivers(), false)
	if err == nil || errors.Is(err, ErrWebUnknown) {
		t.Fatalf("an unreachable router is not an unsupported one: %v", err)
	}
}
