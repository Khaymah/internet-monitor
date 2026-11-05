package router

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Prefix         = "router_"
	ssdpAddr       = "239.255.255.250:1900"
	svcCommon      = "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1"
	svcIP          = "urn:schemas-upnp-org:service:WANIPConnection:1"
	svcIP2         = "urn:schemas-upnp-org:service:WANIPConnection:2"
	svcPPP         = "urn:schemas-upnp-org:service:WANPPPConnection:1"
	requestTimeout = 4 * time.Second
	maxBody        = 256 << 10
)

var ErrNotFound = errors.New("no UPnP router found on this network")

type Status struct {
	Time         time.Time `json:"timestamp"`
	Model        string    `json:"model,omitempty"`
	AccessType   string    `json:"access_type,omitempty"`
	LinkUp       *bool     `json:"link_up,omitempty"`
	SyncDownKbps int64     `json:"sync_down_kbps,omitempty"`
	SyncUpKbps   int64     `json:"sync_up_kbps,omitempty"`
	Connected    *bool     `json:"connected,omitempty"`
	UptimeSecs   int64     `json:"uptime_seconds,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
	MaxDownKbps  int64     `json:"max_down_kbps,omitempty"`
	MaxUpKbps    int64     `json:"max_up_kbps,omitempty"`
	SNRDown      *float64  `json:"snr_down_db,omitempty"`
	SNRUp        *float64  `json:"snr_up_db,omitempty"`
	AttnDown     *float64  `json:"attn_down_db,omitempty"`
	AttnUp       *float64  `json:"attn_up_db,omitempty"`
	LineUpSecs   int64     `json:"line_up_seconds,omitempty"`
	RxBytes      uint64    `json:"rx_bytes,omitempty"`
	TxBytes      uint64    `json:"tx_bytes,omitempty"`
	HasCounters  bool      `json:"has_counters,omitempty"`
}

type Gateway struct {
	Location  string
	Model     string
	commonURL string
	connURL   string
	connType  string
	client    *http.Client
	missing   map[string]bool
}

type device struct {
	ModelName    string    `xml:"modelName"`
	FriendlyName string    `xml:"friendlyName"`
	Services     []service `xml:"serviceList>service"`
	Devices      []device  `xml:"deviceList>device"`
}

type service struct {
	Type    string `xml:"serviceType"`
	Control string `xml:"controlURL"`
}

type root struct {
	URLBase string `xml:"URLBase"`
	Device  device `xml:"device"`
}

func Discover(ctx context.Context, wait time.Duration) (string, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	dst, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return "", err
	}
	for _, st := range []string{"urn:schemas-upnp-org:device:InternetGatewayDevice:1", "urn:schemas-upnp-org:device:InternetGatewayDevice:2"} {
		msg := "M-SEARCH * HTTP/1.1\r\nHOST: " + ssdpAddr + "\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: " + st + "\r\n\r\n"
		if _, err := conn.WriteTo([]byte(msg), dst); err != nil {
			return "", err
		}
	}
	deadline := time.Now().Add(wait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetReadDeadline(deadline)
	buf := make([]byte, 4096)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return "", ErrNotFound
		}
		if loc := ParseSSDP(buf[:n]); loc != "" {
			return loc, nil
		}
	}
}

func ParseSSDP(resp []byte) string {
	r, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(resp)), nil)
	if err != nil {
		return ""
	}
	defer r.Body.Close()
	if !strings.Contains(strings.ToLower(r.Header.Get("ST")+r.Header.Get("USN")), "internetgatewaydevice") {
		return ""
	}
	loc := r.Header.Get("Location")
	u, err := url.Parse(loc)
	if err != nil || u.Scheme != "http" || !isPrivate(u.Hostname()) {
		return ""
	}
	return loc
}

func isPrivate(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}

func Open(ctx context.Context, location string, client *http.Client) (*Gateway, error) {
	if client == nil {
		client = &http.Client{Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("router description answered HTTP %d", resp.StatusCode)
	}
	var r root
	if err := xml.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&r); err != nil {
		return nil, fmt.Errorf("router description: %w", err)
	}
	base := location
	if r.URLBase != "" && hostOf(r.URLBase) == hostOf(location) {
		base = r.URLBase
	}
	g := &Gateway{Location: location, client: client, Model: strings.TrimSpace(r.Device.ModelName), missing: map[string]bool{}}
	if g.Model == "" {
		g.Model = strings.TrimSpace(r.Device.FriendlyName)
	}
	host := hostOf(location)
	walk(r.Device, func(s service) {
		abs := resolve(base, s.Control)
		if hostOf(abs) != host {
			return
		}
		switch s.Type {
		case svcCommon:
			if g.commonURL == "" {
				g.commonURL = abs
			}
		case svcIP, svcIP2, svcPPP:
			if g.connURL == "" {
				g.connURL, g.connType = abs, s.Type
			}
		}
	})
	if g.commonURL == "" && g.connURL == "" {
		return nil, errors.New("the router does not offer UPnP WAN status")
	}
	return g, nil
}

func walk(d device, fn func(service)) {
	for _, s := range d.Services {
		fn(s)
	}
	for _, c := range d.Devices {
		walk(c, fn)
	}
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" {
		return ""
	}
	return u.Host
}

func resolve(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

func (g *Gateway) call(ctx context.Context, controlURL, serviceType, action string) (map[string]string, error) {
	body := `<?xml version="1.0"?>` +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body><u:` + action + ` xmlns:u="` + serviceType + `"/></s:Body></s:Envelope>`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", `"`+serviceType+"#"+action+`"`)
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %w", action, httpError{resp.StatusCode})
	}
	return parseSOAP(io.LimitReader(resp.Body, maxBody))
}

type httpError struct{ status int }

func (e httpError) Error() string { return fmt.Sprintf("HTTP %d", e.status) }

func (g *Gateway) optional(ctx context.Context, controlURL, serviceType, action string) (map[string]string, bool) {
	key := serviceType + "#" + action
	if g.missing[key] {
		return nil, false
	}
	m, err := g.call(ctx, controlURL, serviceType, action)
	var he httpError
	if errors.As(err, &he) {
		g.missing[key] = true
	}
	return m, err == nil
}

func parseSOAP(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	dec := xml.NewDecoder(r)
	var name string
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name = t.Name.Local
		case xml.CharData:
			if v := strings.TrimSpace(string(t)); v != "" && strings.HasPrefix(name, "New") {
				out[name] = v
			}
		case xml.EndElement:
			name = ""
		}
	}
}

func boolPtr(v bool) *bool { return &v }

func (g *Gateway) Status(ctx context.Context) (Status, error) {
	st := Status{Time: time.Now(), Model: g.Model}
	var firstErr error
	if g.commonURL != "" {
		if m, err := g.call(ctx, g.commonURL, svcCommon, "GetCommonLinkProperties"); err == nil {
			st.AccessType = m["NewWANAccessType"]
			if v := m["NewPhysicalLinkStatus"]; v != "" {
				st.LinkUp = boolPtr(strings.EqualFold(v, "Up"))
			}
			st.SyncDownKbps = parseInt(m["NewLayer1DownstreamMaxBitRate"]) / 1000
			st.SyncUpKbps = parseInt(m["NewLayer1UpstreamMaxBitRate"]) / 1000
		} else {
			firstErr = err
		}
		rx, ok1 := g.optional(ctx, g.commonURL, svcCommon, "GetTotalBytesReceived")
		tx, ok2 := g.optional(ctx, g.commonURL, svcCommon, "GetTotalBytesSent")
		if ok1 && ok2 {
			st.RxBytes = uint64(parseInt(rx["NewTotalBytesReceived"]))
			st.TxBytes = uint64(parseInt(tx["NewTotalBytesSent"]))
			st.HasCounters = true
		}
	}
	if g.connURL != "" {
		if m, err := g.call(ctx, g.connURL, g.connType, "GetStatusInfo"); err == nil {
			if v := m["NewConnectionStatus"]; v != "" {
				st.Connected = boolPtr(strings.EqualFold(v, "Connected"))
			}
			st.UptimeSecs = parseInt(m["NewUptime"])
			if v := m["NewLastConnectionError"]; v != "" && v != "ERROR_NONE" {
				st.LastError = v
			}
		} else if firstErr == nil {
			firstErr = err
		}
		if st.SyncDownKbps == 0 {
			if m, ok := g.optional(ctx, g.connURL, g.connType, "GetLinkLayerMaxBitRates"); ok {
				st.SyncDownKbps = parseInt(m["NewDownstreamMaxBitRate"]) / 1000
				st.SyncUpKbps = parseInt(m["NewUpstreamMaxBitRate"]) / 1000
			}
		}
	}
	if st.LinkUp == nil && st.Connected == nil && !st.HasCounters {
		if firstErr == nil {
			firstErr = errors.New("the router returned no status")
		}
		return st, firstErr
	}
	return st, nil
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}
