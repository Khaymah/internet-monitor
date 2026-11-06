package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"internet-monitor/internal/config"
	"internet-monitor/internal/logger"
	"internet-monitor/internal/router"
)

func reportEnv(t *testing.T, hook func(ctx context.Context, deep bool) (router.Report, error)) *testEnv {
	t.Helper()
	dir := t.TempDir()
	store, err := config.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := store.Get()
	cfg.LogDir = filepath.Join(dir, "logs")
	lgr, err := logger.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Options{Store: store, Logger: lgr, Version: "v9.9.9", AppDir: dir, Hooks: Hooks{RouterReport: hook}})
	if err := s.Start(0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
		_ = lgr.Close()
	})
	return &testEnv{s: s, dir: dir, base: fmt.Sprintf("http://127.0.0.1:%d", s.Port())}
}

func sampleReport() router.Report {
	return router.Report{
		Schema: router.ReportSchema,
		App:    router.ReportApp{Version: "v9.9.9", OS: "linux", Arch: "amd64"},
		UPnP:   router.UPnPReport{Answered: true, Device: &router.DeviceInfo{Manufacturer: "ZTE", ModelName: "ZXHN H298A"}},
		Deep:   router.DeepReport{Outcome: "not_requested"},
	}
}

func TestRouterReportReturnsReportSummaryAndIssueLink(t *testing.T) {
	var gotDeep atomic.Bool
	e := reportEnv(t, func(_ context.Context, deep bool) (router.Report, error) {
		gotDeep.Store(deep)
		return sampleReport(), nil
	})
	r := e.do(t, "POST", "/api/router/report", `{"deep":true}`, nil)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("status %d", r.StatusCode)
	}
	res := decode[router.ReportResult](t, r.Body)
	if !gotDeep.Load() || res.Filename != "router-report-zxhn-h298a.json" || !strings.Contains(res.Summary, "ZTE ZXHN H298A") ||
		!strings.HasPrefix(res.IssueURL, router.IssueNewURL+"?") {
		t.Fatalf("%+v", res)
	}
	if r := e.do(t, "POST", "/api/router/report", "", nil); r.StatusCode != http.StatusOK || gotDeep.Load() {
		t.Fatalf("an empty body must mean no signed-in probe (status %d)", r.StatusCode)
	}
}

func TestRouterReportIsProtected(t *testing.T) {
	e := reportEnv(t, func(context.Context, bool) (router.Report, error) { return sampleReport(), nil })
	req, _ := http.NewRequest("POST", e.base+"/api/router/report", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("without the session key: %d, want 401", resp.StatusCode)
	}
	if r := e.do(t, "POST", "/api/router/report", `{}`, map[string]string{"Origin": "https://evil.example"}); r.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin: %d, want 403", r.StatusCode)
	}
	if r := e.do(t, "GET", "/api/router/report", "", nil); r.StatusCode == http.StatusOK {
		t.Errorf("GET must not build a report")
	}
}

func TestRouterReportErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"busy", router.ErrReportBusy, http.StatusTooManyRequests},
		{"not private", router.ErrReportNotPrivate, http.StatusBadRequest},
		{"other", errors.New("boom"), http.StatusBadGateway},
	}
	for _, c := range cases {
		e := reportEnv(t, func(context.Context, bool) (router.Report, error) { return router.Report{}, c.err })
		if r := e.do(t, "POST", "/api/router/report", `{}`, nil); r.StatusCode != c.status {
			t.Errorf("%s: %d, want %d", c.name, r.StatusCode, c.status)
		}
	}
	e := reportEnv(t, nil)
	if r := e.do(t, "POST", "/api/router/report", `{}`, nil); r.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("no hook: %d", r.StatusCode)
	}
	e = reportEnv(t, func(context.Context, bool) (router.Report, error) { return sampleReport(), nil })
	if r := e.do(t, "POST", "/api/router/report", `{bad`, nil); r.StatusCode != http.StatusBadRequest {
		t.Errorf("bad json: %d", r.StatusCode)
	}
}
