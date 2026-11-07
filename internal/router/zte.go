package router

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DriverZTENew = "zte-new"

	zteEntry    = "/?_type=loginData&_tag=login_entry"
	zteSeedPath = "/?_type=loginData&_tag=login_token"
	zteLogout   = "/?_type=loginData&_tag=logout_entry"
)

var zteBusyWords = []string{"another", "already", "logged in", "occupied", "busy"}

type zteNew struct{}

func (zteNew) Name() string { return DriverZTENew }

func (zteNew) Experimental() bool { return true }

type zteEntryReply struct {
	SessToken   string `json:"sess_token"`
	LockingTime int64  `json:"lockingTime"`
	LoginErrMsg string `json:"loginErrMsg"`
	NeedRefresh int    `json:"login_need_refresh"`
}

func parseZTEEntry(body []byte) (zteEntryReply, bool) {
	var raw map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(body), &raw) != nil {
		return zteEntryReply{}, false
	}
	if _, ok := raw["sess_token"]; !ok {
		return zteEntryReply{}, false
	}
	var r zteEntryReply
	if json.Unmarshal(bytes.TrimSpace(body), &r) != nil {
		return zteEntryReply{}, false
	}
	return r, true
}

func (zteNew) Detect(ctx context.Context, p *Probe) (bool, error) {
	s, err := p.Session(ctx)
	if err != nil {
		return false, err
	}
	body, _, err := s.get(ctx, zteEntry)
	if err != nil {
		return false, nil
	}
	_, ok := parseZTEEntry(body)
	return ok, nil
}

func zteHash(password, token string) string {
	sum := sha256.Sum256([]byte(password + token))
	return hex.EncodeToString(sum[:])
}

func (s *session) postForm(ctx context.Context, path string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func stamp() string { return strconv.FormatInt(time.Now().UnixMilli(), 10) }

func (s *session) zteToken(ctx context.Context) (string, error) {
	body, _, err := s.get(ctx, zteSeedPath+"&_="+stamp())
	if err != nil {
		return "", err
	}
	var root zteNode
	if xml.Unmarshal(bytes.TrimSpace(body), &root) != nil || root.XMLName.Local != "ajax_response_xml_root" {
		return "", ErrWebUnknown
	}
	token := strings.TrimSpace(root.Text)
	if token == "" {
		return "", ErrWebUnknown
	}
	return token, nil
}

func zteBusy(msg string) bool {
	m := strings.ToLower(msg)
	for _, w := range zteBusyWords {
		if strings.Contains(m, w) {
			return true
		}
	}
	return false
}

func (s *session) zteLogin(ctx context.Context, user, password string) error {
	body, _, err := s.get(ctx, zteEntry)
	if err != nil {
		return err
	}
	entry, ok := parseZTEEntry(body)
	if !ok {
		return ErrWebUnknown
	}
	switch {
	case entry.LockingTime > 0:
		return fmt.Errorf("%w (the router locked its login for %d seconds)", ErrRouterBusy, entry.LockingTime)
	case entry.LockingTime < 0:
		return fmt.Errorf("%w (the router locked its login)", ErrWrongPassword)
	case entry.SessToken == "":
		return ErrWebUnknown
	}
	token, err := s.zteToken(ctx)
	if err != nil {
		return err
	}
	body, err = s.postForm(ctx, zteEntry, url.Values{
		"action":        {"login"},
		"Password":      {zteHash(password, token)},
		"Username":      {user},
		"_sessionTOKEN": {entry.SessToken},
	})
	if err != nil {
		return err
	}
	res, ok := parseZTEEntry(body)
	if !ok {
		return ErrWebUnknown
	}
	switch {
	case res.LockingTime != 0:
		return fmt.Errorf("%w (the router locked its login)", ErrWrongPassword)
	case res.LoginErrMsg != "" && zteBusy(res.LoginErrMsg):
		return fmt.Errorf("%w (%s)", ErrRouterBusy, res.LoginErrMsg)
	case res.LoginErrMsg != "":
		return fmt.Errorf("%w (%s)", ErrWrongPassword, res.LoginErrMsg)
	case res.SessToken == "":
		return ErrWebUnknown
	}
	if res.NeedRefresh != 0 {
		_, _, _ = s.get(ctx, "/")
	}
	return nil
}

func (s *session) zteLogoutNow(ctx context.Context) {
	_, _ = s.postForm(ctx, zteLogout, url.Values{"IF_LogOff": {"1"}})
}

func (s *session) zteData(ctx context.Context, view, data string) (zteReply, error) {
	if _, _, err := s.get(ctx, "/?_type=menuView&_tag="+view); err != nil {
		return zteReply{}, err
	}
	body, _, err := s.get(ctx, "/?_type=menuData&_tag="+data+"&_="+stamp())
	if err != nil {
		return zteReply{}, err
	}
	r, ok := parseZTEReply(body)
	if !ok {
		return zteReply{}, ErrWebUnknown
	}
	if r.Status != "" && !strings.EqualFold(r.Status, "SUCC") {
		return zteReply{}, fmt.Errorf("%w (%s)", ErrWebUnknown, r.Status)
	}
	return r, nil
}

func (zteNew) Read(ctx context.Context, p *Probe, c Creds) (WebStats, error) {
	s, err := p.Session(ctx)
	if err != nil {
		return WebStats{}, err
	}
	user := c.User
	if user == "" {
		user = "admin"
	}
	if err := s.zteLogin(ctx, user, c.Password); err != nil {
		return WebStats{}, err
	}
	defer s.zteLogoutNow(ctx)
	var out WebStats
	if r, err := s.zteData(ctx, "dslWanStatus", "dsl_interface_status_lua.lua"); err == nil {
		if dsl, ok := zteDSL(r); ok {
			out = dsl
		}
	} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return out, err
	}
	if r, err := s.zteData(ctx, "ethWanStatus&Menu3Location=0", "wan_internetstatus_lua.lua&TypeUplink=2&pageType=1"); err == nil {
		if connected, uptime, ok := zteWAN(r); ok {
			out.Connected, out.UptimeSecs = &connected, uptime
		}
	}
	if !out.HasLineStats && out.Connected == nil {
		return out, ErrWebUnknown
	}
	if r, err := s.zteData(ctx, "statusMgr&Menu3Location=0", "devmgr_statusmgr_lua.lua"); err == nil {
		out.Model = zteModel(r)
	}
	return out, nil
}
