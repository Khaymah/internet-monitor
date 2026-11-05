package router

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"
)

var (
	ErrWrongPassword = errors.New("the router rejected the user name or password")
	ErrWebUnknown    = errors.New("this router's web page is not one the app can read")
	ErrRouterBusy    = errors.New("someone else is signed in to the router's page")

	csrfParam = regexp.MustCompile(`name="csrf_param"\s+content="([^"]+)"`)
	csrfToken = regexp.MustCompile(`name="csrf_token"\s+content="([^"]+)"`)
)

type WebStats struct {
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
	HasCounters  bool
	HasLineStats bool
}

type WebReader struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
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
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 || !isPrivate(req.URL.Hostname()) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}, nil
}

func loginHash(user, password, param, token string) string {
	inner := sha256.Sum256([]byte(password))
	encoded := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(inner[:])))
	outer := sha256.Sum256([]byte(user + encoded + param + token))
	return hex.EncodeToString(outer[:])
}

func unwrap(body []byte) []byte {
	s := strings.TrimSpace(string(body))
	s = strings.TrimPrefix(s, "while(1);")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "/*")
	s = strings.TrimSuffix(s, "*/")
	return []byte(strings.TrimSpace(s))
}

type session struct {
	client *http.Client
	base   string
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

func (s *session) csrf(ctx context.Context) (string, string, error) {
	page, _, err := s.get(ctx, "/")
	if err != nil {
		return "", "", err
	}
	p, t := csrfParam.FindSubmatch(page), csrfToken.FindSubmatch(page)
	if p == nil || t == nil {
		return "", "", ErrWebUnknown
	}
	return string(p[1]), string(t[1]), nil
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

func (s *session) login(ctx context.Context, user, password string) error {
	param, token, err := s.csrf(ctx)
	if err != nil {
		return err
	}
	body, err := s.post(ctx, "/api/system/user_login", map[string]any{
		"csrf": map[string]string{"csrf_param": param, "csrf_token": token},
		"data": map[string]string{"UserName": user, "Password": loginHash(user, password, param, token)},
	})
	if err != nil {
		return err
	}
	var res struct {
		ErrorCategory string `json:"errorCategory"`
	}
	if json.Unmarshal(unwrap(body), &res) != nil {
		return ErrWebUnknown
	}
	switch res.ErrorCategory {
	case "", "ok":
		return nil
	case "Two_user", "Duplicate_login":
		return fmt.Errorf("%w (%s)", ErrRouterBusy, res.ErrorCategory)
	default:
		return fmt.Errorf("%w (%s)", ErrWrongPassword, res.ErrorCategory)
	}
}

func (s *session) logout(ctx context.Context) {
	param, token, err := s.csrf(ctx)
	if err != nil {
		return
	}
	_, _ = s.post(ctx, "/api/system/user_logout", map[string]any{
		"csrf": map[string]string{"csrf_param": param, "csrf_token": token},
	})
}

type dslInfo struct {
	Status               string  `json:"Status"`
	DownCurrRate         int64   `json:"DownCurrRate"`
	UpCurrRate           int64   `json:"UpCurrRate"`
	DownstreamMaxBitRate int64   `json:"DownstreamMaxBitRate"`
	UpstreamMaxBitRate   int64   `json:"UpstreamMaxBitRate"`
	DownMargin           float64 `json:"DownMargin"`
	UpMargin             float64 `json:"UpMargin"`
	DownAttenuation      float64 `json:"DownAttenuation"`
	UpAttenuation        float64 `json:"UpAttenuation"`
	ShowtimeStart        int64   `json:"ShowtimeStart"`
}

type wanStat struct {
	ReceiveBytes uint64 `json:"receivebytes"`
	SendBytes    uint64 `json:"sendbytes"`
}

func parseDSL(body []byte) (dslInfo, bool) {
	var d dslInfo
	if json.Unmarshal(unwrap(body), &d) != nil || d.Status == "" {
		return d, false
	}
	return d, true
}

func parseWAN(body []byte) (rx, tx uint64, ok bool) {
	var list []wanStat
	if json.Unmarshal(unwrap(body), &list) != nil {
		return 0, 0, false
	}
	for _, w := range list {
		rx += w.ReceiveBytes
		tx += w.SendBytes
	}
	return rx, tx, len(list) > 0
}

func (r WebReader) Read(ctx context.Context) (WebStats, error) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	client, err := webClient(r.Host, timeout)
	if err != nil {
		return WebStats{}, err
	}
	s := &session{client: client, base: "http://" + r.Host}
	user := r.User
	if user == "" {
		user = "admin"
	}
	if err := s.login(ctx, user, r.Password); err != nil {
		return WebStats{}, err
	}
	defer s.logout(ctx)
	var out WebStats
	body, _, err := s.get(ctx, "/api/ntwk/dslinfo")
	if err != nil {
		return out, err
	}
	d, ok := parseDSL(body)
	if !ok {
		return out, ErrWebUnknown
	}
	out = WebStats{
		Status:       d.Status,
		DownKbps:     d.DownCurrRate,
		UpKbps:       d.UpCurrRate,
		MaxDownKbps:  d.DownstreamMaxBitRate,
		MaxUpKbps:    d.UpstreamMaxBitRate,
		SNRDown:      d.DownMargin,
		SNRUp:        d.UpMargin,
		AttnDown:     d.DownAttenuation,
		AttnUp:       d.UpAttenuation,
		LineUpSecs:   d.ShowtimeStart,
		HasLineStats: true,
	}
	if body, _, err := s.get(ctx, "/api/ntwk/wan_st"); err == nil {
		out.RxBytes, out.TxBytes, out.HasCounters = parseWAN(body)
	}
	return out, nil
}
