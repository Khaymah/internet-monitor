package router

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var digestParam = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|([^,\s]+))`)

type digestChallenge struct {
	realm, nonce, opaque, qop, algorithm string
	stale                                bool
}

func parseChallenge(header string) (digestChallenge, bool) {
	h := strings.TrimSpace(header)
	if len(h) < 7 || !strings.EqualFold(h[:6], "digest") {
		return digestChallenge{}, false
	}
	var c digestChallenge
	for _, m := range digestParam.FindAllStringSubmatch(h[6:], -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		switch strings.ToLower(m[1]) {
		case "realm":
			c.realm = v
		case "nonce":
			c.nonce = v
		case "opaque":
			c.opaque = v
		case "qop":
			c.qop = v
		case "algorithm":
			c.algorithm = v
		case "stale":
			c.stale = strings.EqualFold(v, "true")
		}
	}
	return c, c.nonce != ""
}

func (c digestChallenge) hasher() func() hash.Hash {
	if strings.HasPrefix(strings.ToUpper(c.algorithm), "SHA-256") {
		return sha256.New
	}
	return md5.New
}

func (c digestChallenge) authorize(user, password, method, uri string, nc int, cnonce string) string {
	sum := func(parts ...string) string {
		h := c.hasher()()
		io.WriteString(h, strings.Join(parts, ":"))
		return hex.EncodeToString(h.Sum(nil))
	}
	ha1 := sum(user, c.realm, password)
	ha2 := sum(method, uri)
	ncText := fmt.Sprintf("%08x", nc)
	var response string
	fields := []string{
		`username="` + user + `"`,
		`realm="` + c.realm + `"`,
		`nonce="` + c.nonce + `"`,
		`uri="` + uri + `"`,
	}
	if strings.Contains(c.qop, "auth") {
		response = sum(ha1, c.nonce, ncText, cnonce, "auth", ha2)
		fields = append(fields, `qop=auth`, `nc=`+ncText, `cnonce="`+cnonce+`"`)
	} else {
		response = sum(ha1, c.nonce, ha2)
	}
	fields = append(fields, `response="`+response+`"`)
	if c.opaque != "" {
		fields = append(fields, `opaque="`+c.opaque+`"`)
	}
	if c.algorithm != "" {
		fields = append(fields, `algorithm=`+c.algorithm)
	}
	return "Digest " + strings.Join(fields, ", ")
}

func soapBody(serviceType, action string) string {
	return `<?xml version="1.0"?>` +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body><u:` + action + ` xmlns:u="` + serviceType + `"/></s:Body></s:Envelope>`
}

type trSession struct {
	p       *Probe
	creds   Creds
	ch      *digestChallenge
	nc      int
	sentAny bool
	trusted bool
}

func (t *trSession) user() string {
	if t.creds.User == "" {
		return "admin"
	}
	return t.creds.User
}

func (t *trSession) send(ctx context.Context, controlURL, serviceType, action string) (*http.Response, error) {
	if !t.p.SameHost(controlURL) {
		return nil, fmt.Errorf("%s is not on the router", controlURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL, strings.NewReader(soapBody(serviceType, action)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", `"`+serviceType+"#"+action+`"`)
	if t.ch != nil {
		t.nc++
		raw := make([]byte, 8)
		_, _ = rand.Read(raw)
		uri := req.URL.RequestURI()
		if u, err := url.Parse(controlURL); err == nil {
			uri = u.RequestURI()
		}
		req.Header.Set("Authorization", t.ch.authorize(t.user(), t.creds.Password, http.MethodPost, uri, t.nc, hex.EncodeToString(raw)))
		t.sentAny = true
	}
	return t.p.Client.Do(req)
}

func (t *trSession) call(ctx context.Context, controlURL, serviceType, action string) (map[string]string, error) {
	for range 3 {
		resp, err := t.send(ctx, controlURL, serviceType, action)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			header := resp.Header.Get("WWW-Authenticate")
			resp.Body.Close()
			ch, ok := parseChallenge(header)
			switch {
			case !ok:
				return nil, fmt.Errorf("%s: %w", action, httpError{resp.StatusCode})
			case t.creds.Password == "":
				return nil, ErrPasswordRequired
			case t.trusted && !ch.stale:
				return nil, fmt.Errorf("%s: %w", action, httpError{resp.StatusCode})
			case t.sentAny && !ch.stale:
				return nil, fmt.Errorf("%w (HTTP 401)", ErrWrongPassword)
			}
			t.ch, t.nc = &ch, 0
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s: %w", action, httpError{resp.StatusCode})
		}
		t.trusted = t.trusted || t.ch != nil
		return parseSOAP(io.LimitReader(resp.Body, maxBody))
	}
	return nil, fmt.Errorf("%w (HTTP 401)", ErrWrongPassword)
}
