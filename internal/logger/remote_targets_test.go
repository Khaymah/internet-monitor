package logger

import (
	"encoding/json"
	"internet-monitor/internal/config"
	"internet-monitor/internal/types"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNtfyRequestHeaders(t *testing.T) {
	req, err := NtfyRequest("https://ntfy.sh/topic", "tk_secret", "down", "Internet disconnected", "TCP: failed")
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Title") != "Internet disconnected" || req.Header.Get("Priority") != "high" || req.Header.Get("Tags") != "x" {
		t.Fatalf("headers = %v", req.Header)
	}
	if req.Header.Get("Authorization") != "Bearer tk_secret" {
		t.Fatal("token must be sent as a bearer token")
	}
	body, _ := io.ReadAll(req.Body)
	if string(body) != "TCP: failed" {
		t.Fatalf("body = %q", body)
	}
	if r, _ := NtfyRequest("https://ntfy.sh/topic", "", "weekly", "t", "b"); r.Header.Get("Authorization") != "" || r.Header.Get("Priority") != "low" {
		t.Fatal("no token means no Authorization header; weekly is low priority")
	}
}

func TestPlainBodyDropsTitleAndMarkdown(t *testing.T) {
	if got := PlainBody("❌ *Internet disconnected*\nTCP: *failed*\nDNS: OK"); got != "TCP: failed\nDNS: OK" {
		t.Fatalf("got %q", got)
	}
}

func TestDeliverGenericWebhookAndNtfy(t *testing.T) {
	var mu sync.Mutex
	got := map[string]string{}
	var wg sync.WaitGroup
	wg.Add(2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got[r.URL.Path] = string(b)
		if r.URL.Path == "/alerts" {
			got["title"] = r.Header.Get("Title")
		}
		mu.Unlock()
		wg.Done()
	}))
	defer srv.Close()
	base := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)

	cfg := testConfig(t)
	cfg.WebhookURL = base + "/hook"
	cfg.NtfyURL = base + "/alerts"
	l, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ev := types.Event{Timestamp: time.Now(), EventType: "disconnected", Reason: types.EventReason{TCPPingFailed: true}}
	l.SendEvent("en", ev, "down", types.Targets{Webhook: true, Ntfy: true})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for both deliveries")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(got["/hook"]), &payload); err != nil {
		t.Fatalf("generic payload is not JSON: %v", err)
	}
	if payload["source"] != "internet-monitor" || payload["type"] != "down" || payload["data"] == nil {
		t.Fatalf("generic payload = %v", payload)
	}
	if got["title"] == "" || strings.Contains(got["/alerts"], "*") {
		t.Fatalf("ntfy title %q body %q", got["title"], got["/alerts"])
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Default()
	cfg.LogDir = t.TempDir()
	return cfg
}
