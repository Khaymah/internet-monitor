package speedtest

import (
	"context"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	probeEvery   = 250 * time.Millisecond
	probeTimeout = 2 * time.Second
	idleSamples  = 5
)

func probeClient() *http.Client {
	return &http.Client{
		Timeout: probeTimeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: probeTimeout,
		},
	}
}

func rtt(ctx context.Context, client *http.Client, url string) (time.Duration, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, false
	}
	t0 := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return time.Since(t0), true
}

func median(ms []int64) int64 {
	if len(ms) == 0 {
		return 0
	}
	s := append([]int64(nil), ms...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

func IdleLatency(ctx context.Context, endpoint string) int64 {
	client := probeClient()
	defer client.CloseIdleConnections()
	url := endpoint + "?bytes=0"
	rtt(ctx, client, url)
	var samples []int64
	for i := 0; i < idleSamples && ctx.Err() == nil; i++ {
		if d, ok := rtt(ctx, client, url); ok {
			samples = append(samples, max(d.Milliseconds(), 1))
		}
	}
	return median(samples)
}

type Probe struct {
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	samples []int64
}

func StartProbe(ctx context.Context, endpoint string) *Probe {
	pctx, cancel := context.WithCancel(ctx)
	p := &Probe{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		client := probeClient()
		defer client.CloseIdleConnections()
		url := endpoint + "?bytes=0"
		rtt(pctx, client, url)
		t := time.NewTicker(probeEvery)
		defer t.Stop()
		for {
			select {
			case <-pctx.Done():
				return
			case <-t.C:
				if d, ok := rtt(pctx, client, url); ok && pctx.Err() == nil {
					p.mu.Lock()
					p.samples = append(p.samples, max(d.Milliseconds(), 1))
					p.mu.Unlock()
				}
			}
		}
	}()
	return p
}

func (p *Probe) Stop() int64 {
	p.cancel()
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return median(p.samples)
}

func BufferbloatGrade(idleMs, loadedMs int64) string {
	if idleMs <= 0 || loadedMs <= 0 {
		return ""
	}
	switch delta := loadedMs - idleMs; {
	case delta < 5:
		return "A+"
	case delta < 30:
		return "A"
	case delta < 60:
		return "B"
	case delta < 200:
		return "C"
	case delta < 400:
		return "D"
	default:
		return "F"
	}
}
