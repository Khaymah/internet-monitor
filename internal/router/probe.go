package router

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const defaultWebTimeout = 8 * time.Second

type Probe struct {
	Host    string
	Client  *http.Client
	s       *session
	page    []byte
	pageErr error
	paged   bool
	trHTTP  string
	trHTTPS string
}

func NewProbe(host string, timeout time.Duration) (*Probe, error) {
	if timeout <= 0 {
		timeout = defaultWebTimeout
	}
	client, err := webClient(host, timeout)
	if err != nil {
		return nil, err
	}
	return &Probe{Host: host, Client: client, s: &session{client: client}}, nil
}

func (p *Probe) Hostname() string {
	if h, _, err := net.SplitHostPort(p.Host); err == nil {
		return h
	}
	return p.Host
}

func (p *Probe) Page(ctx context.Context) ([]byte, error) {
	if p.paged {
		return p.page, p.pageErr
	}
	p.paged = true
	p.page, p.pageErr = p.s.pick(ctx, p.Host)
	return p.page, p.pageErr
}

func (p *Probe) Unreachable() bool {
	return p.paged && p.pageErr != nil
}

func (p *Probe) Session(ctx context.Context) (*session, error) {
	if _, err := p.Page(ctx); err != nil {
		return nil, err
	}
	return &session{client: p.Client, base: p.s.base}, nil
}

func (p *Probe) SameHost(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Hostname() == p.Hostname()
}

func (p *Probe) Get(ctx context.Context, rawURL string) ([]byte, int, error) {
	if !p.SameHost(rawURL) {
		return nil, 0, fmt.Errorf("%q is not on the router", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	return body, resp.StatusCode, err
}
