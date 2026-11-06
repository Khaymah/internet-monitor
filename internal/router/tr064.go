package router

import (
	"context"
	"encoding/xml"
	"errors"
	"net"
	"strings"
)

const (
	DriverTR064 = "tr064"

	trPrefix    = "urn:dslforum-org:service:"
	svcTRCommon = trPrefix + "WANCommonInterfaceConfig:1"
	svcTRDSL    = trPrefix + "WANDSLInterfaceConfig:1"
	svcTRPPP    = trPrefix + "WANPPPConnection:1"
	svcTRIP     = trPrefix + "WANIPConnection:1"
	svcTRInfo   = trPrefix + "DeviceInfo:1"
)

type tr064 struct{}

func (tr064) Name() string    { return DriverTR064 }
func (tr064) Anonymous() bool { return true }

type trDesc struct {
	url      string
	model    string
	services map[string]string
}

func trBases(p *Probe) []string {
	h := p.Hostname()
	plain, secure := p.trHTTP, p.trHTTPS
	if plain == "" {
		plain = "49000"
	}
	if secure == "" {
		secure = "49443"
	}
	return []string{"http://" + net.JoinHostPort(h, plain), "https://" + net.JoinHostPort(h, secure)}
}

func loadTRDesc(ctx context.Context, p *Probe) (trDesc, error) {
	var last error
	for _, base := range trBases(p) {
		descURL := base + "/tr64desc.xml"
		body, status, err := p.Get(ctx, descURL)
		if err != nil {
			last = err
			continue
		}
		if status != 200 {
			last = httpError{status}
			continue
		}
		var r root
		if xml.Unmarshal(body, &r) != nil {
			last = ErrWebUnknown
			continue
		}
		d := trDesc{url: descURL, services: map[string]string{}}
		d.model = strings.TrimSpace(r.Device.ModelName)
		if d.model == "" {
			d.model = strings.TrimSpace(r.Device.FriendlyName)
		}
		walk(r.Device, func(s service) {
			ctl := resolve(descURL, s.Control)
			if !strings.HasPrefix(s.Type, trPrefix) || !p.SameHost(ctl) {
				return
			}
			if _, ok := d.services[s.Type]; !ok {
				d.services[s.Type] = ctl
			}
		})
		if len(d.services) > 0 {
			return d, nil
		}
		last = ErrWebUnknown
	}
	return trDesc{}, last
}

func (tr064) Detect(ctx context.Context, p *Probe) (bool, error) {
	_, err := loadTRDesc(ctx, p)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func fatalRead(err error) bool {
	return errors.Is(err, ErrWrongPassword) || errors.Is(err, ErrPasswordRequired)
}

func (tr064) Read(ctx context.Context, p *Probe, c Creds) (WebStats, error) {
	desc, err := loadTRDesc(ctx, p)
	if err != nil {
		return WebStats{}, ErrWebUnknown
	}
	t := &trSession{p: p, creds: c}
	out := WebStats{Model: desc.model}
	var got bool
	ask := func(svc, action string) (map[string]string, error) {
		ctl, ok := desc.services[svc]
		if !ok {
			return nil, errNoService
		}
		return t.call(ctx, ctl, svc, action)
	}
	if err := readCounters(ask, &out); fatalRead(err) {
		return WebStats{}, err
	} else if out.HasCounters {
		got = true
	}
	if m, err := ask(svcTRDSL, "GetInfo"); fatalRead(err) {
		return WebStats{}, err
	} else if err == nil {
		readDSL(m, &out)
		got = got || out.HasLineStats
	}
	conn, err := readConnection(ask)
	if fatalRead(err) {
		return WebStats{}, err
	}
	if conn != nil {
		out.Connected = boolPtr(strings.EqualFold(conn["NewConnectionStatus"], "Connected"))
		out.UptimeSecs = parseInt(conn["NewUptime"])
		if v := conn["NewLastConnectionError"]; v != "" && v != "ERROR_NONE" {
			out.LastError = v
		}
		got = true
	}
	if m, err := ask(svcTRInfo, "GetInfo"); fatalRead(err) {
		return WebStats{}, err
	} else if err == nil {
		if v := strings.TrimSpace(m["NewModelName"]); v != "" {
			out.Model = v
		}
		if out.UptimeSecs == 0 {
			out.UptimeSecs = parseInt(m["NewUpTime"])
		}
	}
	if !got {
		return out, ErrWebUnknown
	}
	return out, nil
}

var errNoService = errors.New("the router does not offer this service")

type askFunc func(svc, action string) (map[string]string, error)

func readCounters(ask askFunc, out *WebStats) error {
	m, err := ask(svcTRCommon, "GetAddonInfos")
	if fatalRead(err) {
		return err
	}
	if err == nil {
		rx, tx := m["NewX_AVM_DE_TotalBytesReceived64"], m["NewX_AVM_DE_TotalBytesSent64"]
		if rx != "" && tx != "" {
			out.RxBytes, out.TxBytes, out.HasCounters = uint64(parseInt(rx)), uint64(parseInt(tx)), true
			return nil
		}
	}
	rx, err := ask(svcTRCommon, "GetTotalBytesReceived")
	if err != nil {
		return err
	}
	tx, err := ask(svcTRCommon, "GetTotalBytesSent")
	if err != nil {
		return err
	}
	if rx["NewTotalBytesReceived"] != "" && tx["NewTotalBytesSent"] != "" {
		out.RxBytes = uint64(parseInt(rx["NewTotalBytesReceived"]))
		out.TxBytes = uint64(parseInt(tx["NewTotalBytesSent"]))
		out.HasCounters = true
	}
	return nil
}

func readDSL(m map[string]string, out *WebStats) {
	out.Status = m["NewStatus"]
	out.DownKbps = parseInt(m["NewDownstreamCurrRate"])
	out.UpKbps = parseInt(m["NewUpstreamCurrRate"])
	out.MaxDownKbps = parseInt(m["NewDownstreamMaxRate"])
	out.MaxUpKbps = parseInt(m["NewUpstreamMaxRate"])
	out.SNRDown = float64(parseInt(m["NewDownstreamNoiseMargin"])) / 10
	out.SNRUp = float64(parseInt(m["NewUpstreamNoiseMargin"])) / 10
	out.AttnDown = float64(parseInt(m["NewDownstreamAttenuation"])) / 10
	out.AttnUp = float64(parseInt(m["NewUpstreamAttenuation"])) / 10
	out.HasLineStats = out.Status != "" || out.DownKbps > 0 || out.MaxDownKbps > 0
}

func readConnection(ask askFunc) (map[string]string, error) {
	var first map[string]string
	for _, svc := range []string{svcTRPPP, svcTRIP} {
		m, err := ask(svc, "GetStatusInfo")
		if fatalRead(err) {
			return nil, err
		}
		if err != nil {
			continue
		}
		if strings.EqualFold(m["NewConnectionStatus"], "Connected") {
			return m, nil
		}
		if first == nil {
			first = m
		}
	}
	return first, nil
}
