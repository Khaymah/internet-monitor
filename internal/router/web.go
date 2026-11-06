package router

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

var (
	ErrWrongPassword = errors.New("the router rejected the user name or password")
	ErrWebUnknown    = errors.New("this router's web page is not one the app can read")
	ErrRouterBusy    = errors.New("someone else is signed in to the router's page")
)

type WebStats struct {
	Driver       string
	Model        string
	Status       string
	DownKbps     int64
	UpKbps       int64
	MaxDownKbps  int64
	MaxUpKbps    int64
	SNRDown      float64
	SNRUp        float64
	AttnDown     float64
	AttnUp       float64
	LineUpSecs   int64
	RxBytes      uint64
	TxBytes      uint64
	Connected    *bool
	UptimeSecs   int64
	LastError    string
	HasCounters  bool
	HasLineStats bool
}

type WebReader struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
}

func (r WebReader) Read(ctx context.Context) (WebStats, error) {
	p, err := NewProbe(r.Host, r.Timeout)
	if err != nil {
		return WebStats{}, err
	}
	d, err := Detect(ctx, p, Drivers(), r.Password == "")
	if err != nil {
		return WebStats{}, err
	}
	stats, err := d.Read(ctx, p, Creds{User: r.User, Password: r.Password})
	stats.Driver = d.Name()
	return stats, err
}

func webClient(host string, timeout time.Duration) (*http.Client, error) {
	name := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		name = h
	}
	if !isPrivate(name) {
		return nil, fmt.Errorf("router address %q is not on the home network", host)
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: timeout,
		Jar:     jar,
		Transport: &http.Transport{
			Proxy:           nil,
			TLSClientConfig: routerTLS(),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 || req.URL.Hostname() != name {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}, nil
}

func routerTLS() *tls.Config {
	var suites []uint16
	for _, c := range tls.CipherSuites() {
		suites = append(suites, c.ID)
	}
	for _, c := range tls.InsecureCipherSuites() {
		suites = append(suites, c.ID)
	}
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		CipherSuites:       suites,
	}
}

type session struct {
	client *http.Client
	base   string
}

func (s *session) pick(ctx context.Context, host string) ([]byte, error) {
	var errs []error
	for _, scheme := range []string{"https://", "http://"} {
		s.base = scheme + host
		body, _, err := s.get(ctx, "/")
		if err == nil {
			return body, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", scheme+host, err))
	}
	return nil, errors.Join(errs...)
}

func (s *session) get(ctx context.Context, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+path, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if resp.Request != nil && resp.Request.URL != nil {
		s.base = resp.Request.URL.Scheme + "://" + resp.Request.URL.Host
	}
	if resp.StatusCode != http.StatusOK {
		return body, "", fmt.Errorf("%s answered HTTP %d", path, resp.StatusCode)
	}
	return body, resp.Header.Get("Content-Type"), err
}

func (s *session) post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("%s answered HTTP %d", path, resp.StatusCode)
	}
	return body, nil
}

func unwrap(body []byte) []byte {
	s := strings.TrimSpace(string(body))
	s = strings.TrimPrefix(s, "while(1);")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "/*")
	s = strings.TrimSuffix(s, "*/")
	return []byte(strings.TrimSpace(s))
}
