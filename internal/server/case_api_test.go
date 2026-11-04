package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"internet-monitor/internal/casefile"
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
