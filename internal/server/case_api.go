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
	"internet-monitor/internal/jsonl"
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
on %s for %s.

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
	now := time.Now()
	q := r.URL.Query()
	month := q.Get("month")
	var period, label string
	var reportData interface{}
	var wantFile func(name string) bool
	if q.Get("from") != "" {
		from, ok1 := parseRangeTime(q.Get("from"), now)
		to, ok2 := parseRangeTime(q.Get("to"), now)
		if to.After(now) {
			to = now
		}
		if !ok1 || !ok2 || !from.Before(to) {
			writeError(w, http.StatusBadRequest, "invalid_range", "from and to must be RFC3339 with from before to")
			return
		}
		if to.Sub(from) > maxEvidenceRange {
			from = to.Add(-maxEvidenceRange)
		}
		days := map[string]bool{}
		for d := from.AddDate(0, 0, -1); !d.After(to); d = d.AddDate(0, 0, 1) {
			days[d.Format("2006-01-02")] = true
		}
		wantFile = func(name string) bool {
			i := strings.LastIndex(name, "_")
			return i >= 0 && days[strings.TrimSuffix(name[i+1:], ".jsonl")]
		}
		period = from.Format("2006-01-02 15:04") + " – " + to.Format("2006-01-02 15:04")
		label = from.Format("20060102") + "-" + to.Format("20060102")
		reportData = s.evidenceFor(from, to, now)
	} else {
		if !validMonth(month) {
			writeError(w, http.StatusBadRequest, "invalid_month", "month must be YYYY-MM")
			return
		}
		wantFile = func(name string) bool { return strings.Contains(name, "_"+month) }
		period, label = "the month "+month, month
		reportData = s.buildReport(month, now)
	}
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

	rep, err := json.MarshalIndent(reportData, "", "  ")
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
			if strings.HasPrefix(n, prefix) && wantFile(n) {
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

	if err := add("README.txt", fmt.Appendf(nil, evidenceReadme, now.Format(time.RFC1123), period)); err != nil {
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
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="internet-monitor-evidence-%s.zip"`, label))
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

const maxEvidenceRange = 92 * 24 * time.Hour

type rangeEvidence struct {
	From    string                `json:"from"`
	To      string                `json:"to"`
	Summary report.MonthlySummary `json:"summary"`
	Speed   report.SpeedSummary   `json:"speed"`
}

func (s *Server) evidenceFor(from, to, now time.Time) rangeEvidence {
	dir := s.logDir()
	events := jsonl.DecodeDays[types.Event](dir, "connectivity_", from.AddDate(0, 0, -1), to.AddDate(0, 0, 1))
	samples := jsonl.DecodeDays[types.MetricSample](dir, "metrics_", from, to)
	var speeds []report.SpeedSample
	for _, sp := range jsonl.DecodeDays[report.SpeedSample](dir, "speedtest_", from, to) {
		if !sp.Timestamp.Before(from) && sp.Timestamp.Before(to) {
			speeds = append(speeds, sp)
		}
	}
	f, _ := s.caseStore().Load()
	return rangeEvidence{
		From:    from.Format(time.RFC3339),
		To:      to.Format(time.RFC3339),
		Summary: report.SummarizeRange(events, samples, from, to, now),
		Speed:   report.SummarizeSpeed(speeds, "", f.Profile.PlanDownloadMbps),
	}
}

func parseRangeTime(v string, def time.Time) (time.Time, bool) {
	if v == "" {
		return def, true
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, false
	}
	return t.In(time.Local), true
}

func (s *Server) serveCaseEvidence(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	to, ok := parseRangeTime(r.URL.Query().Get("to"), now)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_range", "to must be RFC3339")
		return
	}
	from, ok := parseRangeTime(r.URL.Query().Get("from"), to.Add(-7*24*time.Hour))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_range", "from must be RFC3339")
		return
	}
	if to.After(now) {
		to = now
	}
	if !from.Before(to) {
		writeError(w, http.StatusBadRequest, "invalid_range", "from must be before to")
		return
	}
	if to.Sub(from) > maxEvidenceRange {
		from = to.Add(-maxEvidenceRange)
	}
	writeJSON(w, http.StatusOK, s.evidenceFor(from, to, now))
}

func (s *Server) serveWeeklySend(w http.ResponseWriter, _ *http.Request) {
	if s.hooks.SendWeekly == nil {
		writeError(w, http.StatusNotImplemented, "unavailable", "weekly summary is not available in this build")
		return
	}
	if !s.tester.allow("weekly") {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "wait a moment and try again")
		return
	}
	if !s.hooks.SendWeekly() {
		writeError(w, http.StatusConflict, "no_data", "no monitoring data in the last 7 days")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) serveTimeline(w http.ResponseWriter, r *http.Request) {
	hours := parseLimit(r.URL.Query().Get("hours"), 24, 24*7)
	now := time.Now()
	from := now.Add(-time.Duration(hours) * time.Hour)
	dir := s.logDir()
	events := jsonl.DecodeDays[types.Event](dir, "connectivity_", from.AddDate(0, 0, -1), now)
	samples := jsonl.DecodeDays[types.MetricSample](dir, "metrics_", from, now)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"from":     from.Format(time.RFC3339),
		"to":       now.Format(time.RFC3339),
		"segments": report.Timeline(events, samples, from, now),
	})
}
