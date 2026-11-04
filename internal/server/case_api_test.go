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
