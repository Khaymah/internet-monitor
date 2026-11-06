package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"internet-monitor/internal/jsonl"
	"internet-monitor/internal/report"
	"internet-monitor/internal/router"
)

const gigabyte = 1e9

type PlanUsage struct {
	From           time.Time `json:"from"`
	To             time.Time `json:"to"`
	UsedBytes      uint64    `json:"used_bytes"`
	QuotaBytes     uint64    `json:"quota_bytes"`
	UsedPct        float64   `json:"used_pct"`
	ElapsedPct     float64   `json:"elapsed_pct"`
	ProjectedBytes uint64    `json:"projected_bytes"`
	DaysLeft       int       `json:"days_left"`
}

type routerView struct {
	Enabled  bool           `json:"enabled"`
	Found    bool           `json:"found"`
	Status   *router.Status `json:"status,omitempty"`
	Error    string         `json:"error,omitempty"`
	Usage    *PlanUsage     `json:"usage,omitempty"`
	WebOK    bool           `json:"web_ok,omitempty"`
	WebError string         `json:"web_error,omitempty"`
	Driver   string         `json:"driver,omitempty"`
}

func (s *Server) PlanUsage(now time.Time) (PlanUsage, bool) {
	f, _ := s.caseStore().Load()
	period := router.PeriodAt(now, f.Profile.QuotaResetDay)
	samples := jsonl.DecodeDays[router.Status](s.logDir(), router.Prefix, period.From, now)
	counted := false
	for _, st := range samples {
		counted = counted || st.HasCounters
	}
	if !counted {
		return PlanUsage{}, false
	}
	u := PlanUsage{
		From:       period.From,
		To:         period.To,
		UsedBytes:  router.Usage(samples, period.From, period.To),
		QuotaBytes: uint64(f.Profile.QuotaGB * gigabyte),
		ElapsedPct: period.Elapsed(now) * 100,
		DaysLeft:   max(0, int(period.To.Sub(now).Hours()/24)),
	}
	if u.QuotaBytes > 0 {
		u.UsedPct = float64(u.UsedBytes) / float64(u.QuotaBytes) * 100
	}
	if e := period.Elapsed(now); e > 0.05 {
		u.ProjectedBytes = uint64(float64(u.UsedBytes) / e)
	}
	return u, true
}

func (s *Server) serveRouter(w http.ResponseWriter, _ *http.Request) {
	view := routerView{Enabled: s.store.Get().RouterStatus}
	if s.hooks.Router != nil {
		snap := s.hooks.Router()
		view.Found, view.Status, view.Error = snap.Found, snap.Status, snap.Error
		view.WebOK, view.WebError, view.Driver = snap.WebOK, snap.WebError, snap.Driver
	}
	if u, ok := s.PlanUsage(time.Now()); ok {
		view.Usage = &u
	}
	writeJSON(w, http.StatusOK, view)
}

func attachRouter(rows []report.EventRow, samples []router.Status) {
	if len(samples) == 0 {
		return
	}
	for i := range rows {
		if rows[i].Type != "disconnected" {
			continue
		}
		start, err := time.Parse(time.RFC3339, rows[i].Time)
		if err != nil {
			continue
		}
		end := start.Add(time.Duration(rows[i].DurationSecs * float64(time.Second)))
		rows[i].Router = router.Classify(samples, start, end)
	}
}

func (s *Server) serveTestRouter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", "request body too large")
		return
	}
	if len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if s.hooks.TestRouter == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "router reading is not available")
		return
	}
	if !s.tester.allow("test-router") {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "please wait a minute before testing again")
		return
	}
	stats, err := s.hooks.TestRouter(strings.TrimSpace(req.User), req.Password)
	if err != nil {
		s.tester.forget("test-router")
		code := "router_unreachable"
		switch {
		case errors.Is(err, router.ErrWrongPassword):
			code = "router_wrong_password"
		case errors.Is(err, router.ErrRouterBusy):
			code = "router_busy"
		case errors.Is(err, router.ErrWebUnknown):
			code = "router_unsupported"
		}
		writeError(w, http.StatusBadGateway, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"down_kbps":    stats.DownKbps,
		"up_kbps":      stats.UpKbps,
		"snr_down_db":  stats.SNRDown,
		"has_counters": stats.HasCounters,
		"driver":       stats.Driver,
		"has_line":     stats.HasLineStats,
	})
}
