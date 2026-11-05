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
	"internet-monitor/internal/insights"
	"internet-monitor/internal/jsonl"
	"internet-monitor/internal/report"
	"internet-monitor/internal/router"
	"internet-monitor/internal/seal"
	"internet-monitor/internal/trace"
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
	Integrity  seal.Status          `json:"integrity"`
	Comp       report.Compensation  `json:"compensation"`
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
	sum := report.Summarize(events, samples, month, now)
	report.MarkRestarts(&sum, readMonthly[report.Startup](dir, report.SystemPrefix, month))
	integrity, _ := seal.Verify(dir, now)
	attachRoutes(sum.Events, readMonthly[trace.Trace](dir, trace.Prefix, month))
	attachRouter(sum.Events, readMonthly[router.Status](dir, router.Prefix, month))
	return reportView{
		MonthlySummary: sum,
		Speed:          report.SummarizeSpeed(speeds, month, f.Profile.PlanDownloadMbps),
		Profile:        f.Profile,
		Complaints:     casefile.InMonth(f.Complaints, month, now.Location()),
		Integrity:      integrity,
		Comp:           report.Compensate(sum, f.Profile.MonthlyPrice, f.Profile.Currency, f.Profile.Country, now),
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
logs/speedtest_*       Every speed test with download, upload, latency and lag under load.
logs/traceroute_*      The route to the internet, traced when each outage started:
                       which hops answered and where it stopped.
logs/router_*          Present when the router shares its status (UPnP): once a minute,
                       the line state, sync speed, connection state, router uptime
                       and traffic counters, as the router reported them.
seals.jsonl            One line per finished day: the SHA-256 of each of that day's log
                       files and the hash of the previous day's seal. Changing any old
                       file, or any older seal, breaks the chain from that day on.
seals/DATE.tsr         Present when timestamping was turned on: an RFC 3161 timestamp
seals/DATE.seal.json   from an independent authority over that day's seal. Check it with
                       openssl ts -verify -data seals/DATE.seal.json -in seals/DATE.tsr \
                         -CAfile cacert.pem -untrusted tsa.crt
                       (cacert.pem and tsa.crt from https://freetsa.org/files/ when the
                       default authority was used). "Verification: OK" proves the logs
                       of that day existed in this exact form at the time shown by
                         openssl ts -reply -in seals/DATE.tsr -text
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
		days := logDays(from, to, now.Location())
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
		for _, prefix := range []string{"connectivity_", "metrics_", "speedtest_", trace.Prefix, report.SystemPrefix, router.Prefix} {
			if strings.HasPrefix(n, prefix) && wantFile(n) {
				names = append(names, n)
			}
		}
	}
	sort.Strings(names)
	sealed := map[string]bool{}
	for _, n := range names {
		if i := strings.LastIndex(n, "_"); i >= 0 {
			sealed[strings.TrimSuffix(n[i+1:], ".jsonl")] = true
		}
	}
	if data, err := readCapped(filepath.Join(dir, seal.ChainFile), 16<<20); err == nil {
		if err := add(seal.ChainFile, data); err != nil {
			fail(err)
			return
		}
	}
	for day := range sealed {
		for _, ext := range []string{".seal.json", ".tsr"} {
			if data, err := readCapped(filepath.Join(dir, seal.StampDir, day+ext), 1<<20); err == nil {
				if err := add(seal.StampDir+"/"+day+ext, data); err != nil {
					fail(err)
					return
				}
			}
		}
	}
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
	Comp    report.Compensation   `json:"compensation"`
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
	sum := report.SummarizeRange(events, samples, from, to, now)
	report.MarkRestarts(&sum, jsonl.DecodeDays[report.Startup](dir, report.SystemPrefix, from, to))
	attachRouter(sum.Events, jsonl.DecodeDays[router.Status](dir, router.Prefix, from, to.AddDate(0, 0, 1)))
	return rangeEvidence{
		From:    from.Format(time.RFC3339),
		To:      to.Format(time.RFC3339),
		Summary: sum,
		Speed:   report.SummarizeSpeed(speeds, "", f.Profile.PlanDownloadMbps),
		Comp:    report.Compensate(sum, f.Profile.MonthlyPrice, f.Profile.Currency, f.Profile.Country, now),
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

func logDays(from, to time.Time, loc *time.Location) map[string]bool {
	y, m, d := from.In(loc).AddDate(0, 0, -1).Date()
	last := to.In(loc).Format("2006-01-02")
	days := map[string]bool{}
	for day := time.Date(y, m, d, 0, 0, 0, 0, loc); day.Format("2006-01-02") <= last; day = day.AddDate(0, 0, 1) {
		days[day.Format("2006-01-02")] = true
	}
	return days
}

func attachRoutes(rows []report.EventRow, traces []trace.Trace) {
	byTime := make(map[int64]trace.Trace, len(traces))
	for _, t := range traces {
		byTime[t.EventTime.Unix()] = t
	}
	for i := range rows {
		if rows[i].Type != "disconnected" {
			continue
		}
		at, err := time.Parse(time.RFC3339, rows[i].Time)
		if err != nil {
			continue
		}
		if t, ok := byTime[at.Unix()]; ok {
			rows[i].Route = &report.Route{Hops: len(t.Hops), LastHop: t.LastHop, LastAddr: t.LastAddr, Reached: t.Reached}
		}
	}
}

func (s *Server) serveSeals(w http.ResponseWriter, _ *http.Request) {
	st, err := seal.Verify(s.logDir(), time.Now())
	if err != nil {
		s.logf("seals: %v", err)
		writeError(w, http.StatusInternalServerError, "read_failed", "could not check the sealed logs")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) Insights(days int, now time.Time) []insights.Insight {
	days = max(1, min(days, 90))
	from := now.AddDate(0, 0, -days)
	ev := s.evidenceFor(from, now, now)
	attachRoutes(ev.Summary.Events, jsonl.DecodeDays[trace.Trace](s.logDir(), trace.Prefix, from, now))
	var grades []string
	for _, sp := range jsonl.DecodeDays[report.SpeedSample](s.logDir(), "speedtest_", now.AddDate(0, 0, -14), now) {
		if sp.Bufferbloat != "" {
			grades = append(grades, sp.Bufferbloat)
		}
	}
	if len(grades) > 5 {
		grades = grades[len(grades)-5:]
	}
	recent := false
	if f, err := s.caseStore().Load(); err == nil {
		for _, c := range f.Complaints {
			opened, err := time.Parse(time.RFC3339, c.OpenedAt)
			if c.Status == casefile.StatusOpen || c.Status == casefile.StatusEscalated || (err == nil && now.Sub(opened) < 7*24*time.Hour) {
				recent = true
			}
		}
	}
	in := insights.Input{Summary: ev.Summary, Speed: ev.Speed, Grades: grades, Days: days, OpenOrRecentComplaint: recent}
	if u, ok := s.PlanUsage(now); ok && u.QuotaBytes > 0 {
		in.QuotaUsedPct = u.UsedPct
		in.QuotaProjectedPct = float64(u.ProjectedBytes) / float64(u.QuotaBytes) * 100
		in.QuotaDaysLeft = u.DaysLeft
	}
	return insights.Analyze(in)
}

func (s *Server) serveInsights(w http.ResponseWriter, r *http.Request) {
	days := parseLimit(r.URL.Query().Get("days"), 30, 90)
	list := s.Insights(days, time.Now())
	if list == nil {
		list = []insights.Insight{}
	}
	writeJSON(w, http.StatusOK, list)
}
