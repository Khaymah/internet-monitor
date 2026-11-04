package speedtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBufferbloatGrade(t *testing.T) {
	cases := []struct {
		idle, loaded int64
		want         string
	}{
		{20, 22, "A+"},
		{20, 45, "A"},
		{20, 70, "B"},
		{20, 150, "C"},
		{20, 350, "D"},
		{20, 900, "F"},
		{0, 50, ""},
		{20, 0, ""},
	}
	for _, c := range cases {
		if got := BufferbloatGrade(c.idle, c.loaded); got != c.want {
			t.Errorf("BufferbloatGrade(%d, %d) = %q, want %q", c.idle, c.loaded, got, c.want)
		}
	}
}

func TestProbeSeesAddedDelayUnderLoad(t *testing.T) {
	var loaded atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loaded.Load() {
			time.Sleep(80 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx := context.Background()

	idle := IdleLatency(ctx, srv.URL)
	if idle <= 0 || idle > 40 {
		t.Fatalf("idle latency = %dms, want a small positive number", idle)
	}

	loaded.Store(true)
	p := StartProbe(ctx, srv.URL)
	time.Sleep(1200 * time.Millisecond)
	under := p.Stop()
	if under < 80 {
		t.Fatalf("loaded latency = %dms, want at least the 80ms the server adds", under)
	}
	if g := BufferbloatGrade(idle, under); g != "C" && g != "B" {
		t.Fatalf("grade = %q for idle %d / loaded %d", g, idle, under)
	}
}
