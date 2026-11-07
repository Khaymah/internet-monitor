package server

import (
	"net/http"
	"testing"

	"internet-monitor/internal/router"
)

func TestRouterViewNamesTheDriver(t *testing.T) {
	e := newTestEnv(t)
	e.s.hooks.Router = func() router.Snapshot {
		return router.Snapshot{Found: true, WebOK: true, Driver: router.DriverTR064}
	}
	r := e.do(t, "GET", "/api/router", "", nil)
	got := decode[map[string]any](t, r.Body)
	if got["driver"] != "tr064" || got["web_ok"] != true {
		t.Fatalf("view %v", got)
	}
}

func TestTestRouterReportsDriver(t *testing.T) {
	e := newTestEnv(t)
	e.s.hooks.TestRouter = func(user, password string) (router.WebStats, error) {
		return router.WebStats{Driver: router.DriverHuaweiHG, HasLineStats: true, HasCounters: true}, nil
	}
	r := e.do(t, "POST", "/api/test-router", `{"password":"x"}`, nil)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("status %d", r.StatusCode)
	}
	got := decode[map[string]any](t, r.Body)
	if got["driver"] != "huawei-hg" || got["has_line"] != true {
		t.Fatalf("result %v", got)
	}
}

func TestTestRouterMapsUnsupported(t *testing.T) {
	e := newTestEnv(t)
	e.s.hooks.TestRouter = func(user, password string) (router.WebStats, error) {
		return router.WebStats{}, router.ErrWebUnknown
	}
	r := e.do(t, "POST", "/api/test-router", `{"password":"x"}`, nil)
	if r.StatusCode != http.StatusBadGateway {
		t.Fatalf("status %d", r.StatusCode)
	}
}
