package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"internet-monitor/internal/router"
)

const (
	routerReportTimeout = 75 * time.Second
	routerReportMaxBody = 1024
)

func (s *Server) serveRouterReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Deep bool `json:"deep"`
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, routerReportMaxBody))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", "request body too large")
		return
	}
	if len(bytes.TrimSpace(body)) > 0 && json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if s.hooks.RouterReport == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "router reports are not available")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), routerReportTimeout)
	defer cancel()
	rep, err := s.hooks.RouterReport(ctx, req.Deep)
	switch {
	case errors.Is(err, router.ErrReportBusy):
		writeError(w, http.StatusTooManyRequests, "report_in_progress", err.Error())
		return
	case errors.Is(err, router.ErrReportNotPrivate):
		writeError(w, http.StatusBadRequest, "router_not_private", err.Error())
		return
	case err != nil:
		s.logf("ROUTER report failed: %v", err)
		writeError(w, http.StatusBadGateway, "report_failed", "could not build the router report")
		return
	}
	writeJSON(w, http.StatusOK, router.BuildResult(rep))
}
