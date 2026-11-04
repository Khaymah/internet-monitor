package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"internet-monitor/internal/casefile"
	"internet-monitor/internal/report"
	"internet-monitor/internal/types"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const caseBodyLimit = 64 << 10

func (s *Server) caseStore() *casefile.Store {
	dir := s.appDir
	if dir == "" {
		dir = s.logDir()
	}
	return casefile.Open(dir)
}

func (s *Server) writeCaseError(w http.ResponseWriter, err error) {
	var fe *casefile.FieldError
	switch {
	case errors.As(err, &fe):
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "code": "invalid", "field": fe.Field, "error": fe.Message})
	case errors.Is(err, casefile.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		s.logf("case file: %v", err)
		writeError(w, http.StatusInternalServerError, "case_failed", "could not read or save case.json")
	}
}

func (s *Server) serveCaseGet(w http.ResponseWriter, _ *http.Request) {
	f, err := s.caseStore().Load()
	if err != nil {
		s.writeCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) serveCaseProfile(w http.ResponseWriter, r *http.Request) {
	var p casefile.Profile
	if !decodeJSON(w, r, caseBodyLimit, &p) {
		return
	}
	f, err := s.caseStore().SetProfile(p)
	if err != nil {
		s.writeCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) serveComplaintSave(w http.ResponseWriter, r *http.Request) {
	var c casefile.Complaint
	if !decodeJSON(w, r, caseBodyLimit, &c) {
		return
	}
	saved, err := s.caseStore().Upsert(c, time.Now())
	if err != nil {
		s.writeCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) serveComplaintDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.caseStore().Delete(r.PathValue("id")); err != nil {
		s.writeCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// reportView is the monthly report plus the dispute details that belong on it.
type reportView struct {
	report.MonthlySummary
	Speed      report.SpeedSummary  `json:"speed"`
	Profile    casefile.Profile     `json:"profile"`
	Complaints []casefile.Complaint `json:"complaints"`
}

func (s *Server) buildReport(month string, now time.Time) reportView {
	dir := s.logDir()
	prev, next := adjacentMonths(month)
	events := readMonthly[types.Event](dir, "connectivity_", month)
	events = append(events, readMonthly[types.Event](dir, "connectivity_", prev)...)
	events = append(events, readMonthly[types.Event](dir, "connectivity_", next)...)
	samples := readMonthly[types.MetricSample](dir, "metrics_", month)
	speeds := readMonthly[report.SpeedSample](dir, "speedtest_", month)

	f, err := s.caseStore().Load()
	if err != nil {
		s.logf("case file: %v", err)
	}
	return reportView{
		MonthlySummary: report.Summarize(events, samples, month, now),
		Speed:          report.SummarizeSpeed(speeds, month, f.Profile.PlanDownloadMbps),
		Profile:        f.Profile,
		Complaints:     casefile.InMonth(f.Complaints, month, now.Location()),
	}
}

func (s *Server) serveReport(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if !validMonth(month) {
		month = time.Now().Format("2006-01")
	}
	writeJSON(w, http.StatusOK, s.buildReport(month, time.Now()))
}

const evidenceReadme = `Internet Monitor — evidence pack
================================

This archive was exported by Internet Monitor (https://github.com/Khaymah/internet-monitor)
on %s for the month %s.

report.json            The monthly summary shown on the report page, including the
                       provider details, speed tests and complaints you entered.
case.json              Your provider details and every complaint you logged.
logs/connectivity_*    One line per status change (connected / unstable / disconnected),
                       with the layer that failed (tcp, http, dns).
logs/metrics_*         One line per monitored minute: checks run, checks up, latency,
                       jitter and packet loss. Minutes with no line were not monitored
                       (computer off or asleep, or the app closed).
logs/speedtest_*       Every speed test with download, upload and latency.
SHA256SUMS.txt         The SHA-256 checksum of every file above. Anyone can recompute
                       them (for example "sha256sum -c SHA256SUMS.txt", or
                       "certutil -hashfile <file> SHA256" on Windows) to confirm that
                       nothing was changed after export.
`

func (s *Server) serveEvidence(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if !validMonth(month) {
		writeError(w, http.StatusBadRequest, "invalid_month", "month must be YYYY-MM")
		return
	}
	now := time.Now()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var sums []string
	add := func(name string, data []byte) error {
		fw, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: now})
		if err != nil {
			return err
		}
		if _, err := fw.Write(data); err != nil {
			return err
		}
		h := sha256.Sum256(data)
		sums = append(sums, hex.EncodeToString(h[:])+"  "+name)
		return nil
	}

	fail := func(err error) {
		s.logf("evidence pack: %v", err)
		writeError(w, http.StatusInternalServerError, "export_failed", "could not build the evidence pack")
	}

	rep, err := json.MarshalIndent(s.buildReport(month, now), "", "  ")
	if err != nil {
		fail(err)
		return
	}
	if err := add("report.json", rep); err != nil {
		fail(err)
		return
	}
	caseData, err := json.MarshalIndent(mustCase(s.caseStore()), "", "  ")
	if err == nil {
		err = add("case.json", caseData)
	}
	if err != nil {
		fail(err)
		return
	}

	dir := s.logDir()
	entries, _ := os.ReadDir(dir)
	names := []string{}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".jsonl") {
			continue
		}
		for _, prefix := range []string{"connectivity_", "metrics_", "speedtest_"} {
			if strings.HasPrefix(n, prefix+month) {
				names = append(names, n)
			}
		}
	}
	sort.Strings(names)
	for _, n := range names {
		data, err := readCapped(filepath.Join(dir, n), 64<<20)
		if err != nil {
			fail(err)
			return
		}
		if err := add("logs/"+n, data); err != nil {
			fail(err)
			return
		}
	}

	if err := add("README.txt", fmt.Appendf(nil, evidenceReadme, now.Format(time.RFC1123), month)); err != nil {
		fail(err)
		return
	}
	fw, err := zw.Create("SHA256SUMS.txt")
	if err == nil {
		_, err = io.WriteString(fw, strings.Join(sums, "\n")+"\n")
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		fail(err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="internet-monitor-evidence-%s.zip"`, month))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func mustCase(st *casefile.Store) casefile.File {
	f, _ := st.Load()
	return f
}

func readCapped(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxBytes))
}
