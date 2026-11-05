package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"internet-monitor/internal/casefile"
	"internet-monitor/internal/report"
	"internet-monitor/internal/seal"
	"internet-monitor/internal/trace"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCaseProfileAndComplaints(t *testing.T) {
	e := newTestEnv(t)

	resp := e.do(t, http.MethodPost, "/api/case/profile", `{"provider":"WE","plan_download_mbps":70,"line_number":"0233334444"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("profile status %d", resp.StatusCode)
	}

	resp = e.do(t, http.MethodPost, "/api/case/complaints", `{}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty complaint status %d, want 400", resp.StatusCode)
	}

	resp = e.do(t, http.MethodPost, "/api/case/complaints", `{"provider":"WE","ticket":"98765","opened_at":"2026-11-02T10:00"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status %d", resp.StatusCode)
	}
	created := decode[casefile.Complaint](t, resp.Body)
	if created.ID == "" || created.Status != casefile.StatusOpen {
		t.Fatalf("created %+v", created)
	}

	resp = e.do(t, http.MethodGet, "/api/case", "", nil)
	f := decode[casefile.File](t, resp.Body)
	if f.Profile.Provider != "WE" || f.Profile.PlanDownloadMbps != 70 || len(f.Complaints) != 1 {
		t.Fatalf("case = %+v", f)
	}

	resp = e.do(t, http.MethodDelete, "/api/case/complaints/"+created.ID, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status %d", resp.StatusCode)
	}
	resp = e.do(t, http.MethodDelete, "/api/case/complaints/"+created.ID, "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status %d, want 404", resp.StatusCode)
	}
}

func TestEvidencePackChecksums(t *testing.T) {
	e := newTestEnv(t)
	month := time.Now().Format("2006-01")
	logs := filepath.Join(e.dir, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"timestamp":"` + time.Now().Format(time.RFC3339) + `","event":"disconnected","reason":{"tcp_ping_failed":true}}` + "\n"
	if err := os.WriteFile(filepath.Join(logs, "connectivity_"+month+"-01.jsonl"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	resp := e.do(t, http.MethodGet, "/api/evidence?month=bad", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad month status %d", resp.StatusCode)
	}

	resp = e.do(t, http.MethodGet, "/api/evidence?month="+month, "", nil)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("status %d type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	body, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		files[f.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	for _, want := range []string{"report.json", "case.json", "README.txt", "SHA256SUMS.txt", "logs/connectivity_" + month + "-01.jsonl"} {
		if _, ok := files[want]; !ok {
			t.Errorf("missing %s", want)
		}
	}
	for _, row := range strings.Split(strings.TrimSpace(string(files["SHA256SUMS.txt"])), "\n") {
		sum, name, ok := strings.Cut(row, "  ")
		if !ok {
			t.Fatalf("bad row %q", row)
		}
		h := sha256.Sum256(files[name])
		if hex.EncodeToString(h[:]) != sum {
			t.Errorf("checksum mismatch for %s", name)
		}
	}
}

func TestCaseEvidenceRange(t *testing.T) {
	e := newTestEnv(t)
	logs := filepath.Join(e.dir, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	start := now.Add(-3 * time.Hour).Truncate(time.Minute)
	var metrics, events strings.Builder
	for m := start; m.Before(now.Add(-time.Minute)); m = m.Add(time.Minute) {
		metrics.WriteString(`{"timestamp":"` + m.Format(time.RFC3339) + `","samples":12,"up_samples":12}` + "\n")
	}
	down := now.Add(-2 * time.Hour)
	events.WriteString(`{"timestamp":"` + down.Format(time.RFC3339) + `","event":"disconnected","reason":{"tcp_ping_failed":true}}` + "\n")
	events.WriteString(`{"timestamp":"` + down.Add(10*time.Minute).Format(time.RFC3339) + `","event":"connected","duration_seconds":600}` + "\n")
	write := func(prefix, body string, at time.Time) {
		f, err := os.OpenFile(filepath.Join(logs, prefix+at.Format("2006-01-02")+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.WriteString(body); err != nil {
			t.Fatal(err)
		}
	}
	write("metrics_", metrics.String(), start)
	write("connectivity_", events.String(), down)

	resp := e.do(t, http.MethodGet, "/api/case/evidence?from=nope", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad from status %d", resp.StatusCode)
	}
	from := now.Add(-4 * time.Hour).UTC().Format(time.RFC3339)
	resp = e.do(t, http.MethodGet, "/api/case/evidence?from="+from, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	ev := decode[rangeEvidence](t, resp.Body)
	if ev.Summary.Disconnections != 1 || ev.Summary.TotalDowntimeSecs != 600 {
		t.Fatalf("summary = %+v", ev.Summary)
	}

	resp = e.do(t, http.MethodGet, "/api/evidence?from="+from, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("range pack status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		found = found || f.Name == "logs/connectivity_"+down.Format("2006-01-02")+".jsonl"
	}
	if !found {
		t.Fatal("range pack is missing the connectivity log for the outage day")
	}
}

func TestTimelineEndpoint(t *testing.T) {
	e := newTestEnv(t)
	resp := e.do(t, http.MethodGet, "/api/timeline?hours=999", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body := decode[struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Segments []struct {
			Status string `json:"status"`
		} `json:"segments"`
	}](t, resp.Body)
	from, _ := time.Parse(time.RFC3339, body.From)
	to, _ := time.Parse(time.RFC3339, body.To)
	if h := to.Sub(from).Hours(); h < 167 || h > 169 {
		t.Fatalf("window = %vh, want 168 (capped)", h)
	}
	if len(body.Segments) != 1 || body.Segments[0].Status != "off" {
		t.Fatalf("segments = %+v, want one off segment with no data", body.Segments)
	}
}

func TestAttachRoutes(t *testing.T) {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	rows := []report.EventRow{
		{Time: at.Format(time.RFC3339), Type: "disconnected"},
		{Time: at.Add(time.Hour).Format(time.RFC3339), Type: "disconnected"},
		{Time: at.Format(time.RFC3339), Type: "degraded"},
	}
	attachRoutes(rows, []trace.Trace{{EventTime: at, LastHop: 1, LastAddr: "192.168.1.1", Hops: []trace.Hop{{N: 1}, {N: 2}}}})
	if rows[0].Route == nil || rows[0].Route.LastAddr != "192.168.1.1" || rows[0].Route.Hops != 2 {
		t.Fatalf("route not attached: %+v", rows[0].Route)
	}
	if rows[1].Route != nil || rows[2].Route != nil {
		t.Fatal("route attached to the wrong rows")
	}
}

func TestEvidencePackIncludesSeals(t *testing.T) {
	e := newTestEnv(t)
	logs := filepath.Join(e.dir, "logs")
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	day := time.Now().AddDate(0, 0, -1)
	month := day.Format("2006-01")
	if err := os.WriteFile(filepath.Join(logs, "metrics_"+day.Format("2006-01-02")+".jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&seal.Sealer{Dir: func() string { return logs }}).SealPending(); err != nil {
		t.Fatal(err)
	}
	resp := e.do(t, http.MethodGet, "/api/evidence?month="+month, "", nil)
	body, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		found = found || f.Name == seal.ChainFile
	}
	if !found {
		t.Fatal("seals.jsonl missing from the evidence pack")
	}
	st := decode[seal.Status](t, e.do(t, http.MethodGet, "/api/seals", "", nil).Body)
	if !st.Intact || st.Days != 1 {
		t.Fatalf("seal status = %+v", st)
	}
}

func TestLogDaysCoverEveryDayOfTheRange(t *testing.T) {
	loc := time.FixedZone("EET", 2*3600)
	from := time.Date(2026, 11, 3, 22, 30, 0, 0, time.UTC)
	to := time.Date(2026, 11, 5, 0, 10, 0, 0, loc)
	got := logDays(from, to, loc)
	for _, want := range []string{"2026-11-03", "2026-11-04", "2026-11-05"} {
		if !got[want] {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if got["2026-11-06"] || len(got) != 3 {
		t.Errorf("unexpected days %v", got)
	}
}
