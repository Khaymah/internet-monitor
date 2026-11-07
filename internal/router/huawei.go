package router

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

const DriverHuaweiHG = "huawei-hg"

var (
	csrfParam = regexp.MustCompile(`name="csrf_param"\s+content="([^"]+)"`)
	csrfToken = regexp.MustCompile(`name="csrf_token"\s+content="([^"]+)"`)

	huaweiPages = []string{"/", "/html/index.html"}
)

type huaweiHG struct{}

func (huaweiHG) Name() string { return DriverHuaweiHG }

func csrfFrom(page []byte) (string, string, bool) {
	p, t := csrfParam.FindSubmatch(page), csrfToken.FindSubmatch(page)
	if p == nil || t == nil {
		return "", "", false
	}
	return string(p[1]), string(t[1]), true
}

func (huaweiHG) Detect(ctx context.Context, p *Probe) (bool, error) {
	page, err := p.Page(ctx)
	if err != nil {
		return false, err
	}
	if _, _, ok := csrfFrom(page); ok {
		return true, nil
	}
	s, err := p.Session(ctx)
	if err != nil {
		return false, err
	}
	if body, _, err := s.get(ctx, huaweiPages[1]); err == nil {
		if _, _, ok := csrfFrom(body); ok {
			return true, nil
		}
	}
	if body, _, err := s.get(ctx, "/api/system/deviceinfo"); err == nil {
		var info struct {
			DeviceName   string `json:"DeviceName"`
			SerialNumber string `json:"SerialNumber"`
		}
		if json.Unmarshal(unwrap(body), &info) == nil && (info.DeviceName != "" || info.SerialNumber != "") {
			return true, nil
		}
	}
	return false, nil
}

func loginHash(user, password, param, token string) string {
	inner := sha256.Sum256([]byte(password))
	encoded := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(inner[:])))
	outer := sha256.Sum256([]byte(user + encoded + param + token))
	return hex.EncodeToString(outer[:])
}

func (s *session) csrf(ctx context.Context) (string, string, error) {
	for _, path := range huaweiPages {
		page, _, err := s.get(ctx, path)
		if err != nil {
			continue
		}
		if p, t, ok := csrfFrom(page); ok {
			return p, t, nil
		}
	}
	return "", "", ErrWebUnknown
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
	raw := unwrap(body)
	var list []wanStat
	if json.Unmarshal(raw, &list) != nil {
		var one wanStat
		if json.Unmarshal(raw, &one) != nil {
			return 0, 0, false
		}
		return one.ReceiveBytes, one.SendBytes, true
	}
	for _, w := range list {
		rx += w.ReceiveBytes
		tx += w.SendBytes
	}
	return rx, tx, len(list) > 0
}

func (huaweiHG) Read(ctx context.Context, p *Probe, c Creds) (WebStats, error) {
	s, err := p.Session(ctx)
	if err != nil {
		return WebStats{}, err
	}
	user := c.User
	if user == "" {
		user = "admin"
	}
	if err := s.login(ctx, user, c.Password); err != nil {
		return WebStats{}, err
	}
	defer s.logout(ctx)
	var out WebStats
	dslBody, _, dslErr := s.get(ctx, "/api/ntwk/dslinfo")
	d, dslOK := parseDSL(dslBody)
	if dslErr == nil && dslOK {
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
	}
	if body, _, err := s.get(ctx, "/api/ntwk/wan_st"); err == nil {
		out.RxBytes, out.TxBytes, out.HasCounters = parseWAN(body)
	}
	if !out.HasLineStats && !out.HasCounters {
		if dslErr != nil {
			return out, dslErr
		}
		return out, ErrWebUnknown
	}
	if body, _, err := s.get(ctx, "/api/system/deviceinfo"); err == nil {
		var info struct {
			DeviceName string `json:"DeviceName"`
		}
		if json.Unmarshal(unwrap(body), &info) == nil {
			out.Model = info.DeviceName
		}
	}
	return out, nil
}
