package link

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Wifi     = "wifi"
	Ethernet = "ethernet"
	Other    = "other"
)

type Info struct {
	Type      string `json:"type"`
	Interface string `json:"interface,omitempty"`
	SSID      string `json:"ssid,omitempty"`
	Signal    int    `json:"signal,omitempty"`
}

func DefaultInterface() (string, bool) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", false
	}
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()
	if !ok {
		return "", false
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", false
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(local.IP) {
				return iface.Name, true
			}
		}
	}
	return "", false
}

func GuessByName(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "wl"), strings.Contains(n, "wi-fi"), strings.Contains(n, "wifi"),
		strings.Contains(n, "wireless"), strings.Contains(n, "wlan"):
		return Wifi
	case strings.HasPrefix(n, "eth"), strings.HasPrefix(n, "en"), strings.Contains(n, "ethernet"):
		return Ethernet
	}
	return Other
}

var (
	netshField = regexp.MustCompile(`^\s*([^:]+?)\s*:\s*(.*?)\s*$`)
	percent    = regexp.MustCompile(`(\d+)\s*%`)
)

func ParseNetsh(out string) []Info {
	var list []Info
	var cur *Info
	connected := false
	flush := func() {
		if cur != nil && connected {
			list = append(list, *cur)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		m := netshField.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		key, val := strings.ToLower(m[1]), m[2]
		switch {
		case key == "name":
			flush()
			cur, connected = &Info{Type: Wifi, Interface: val}, false
		case cur == nil:
		case key == "state":
			connected = strings.EqualFold(val, "connected")
		case key == "ssid":
			cur.SSID = val
		case key == "signal":
			if p := percent.FindStringSubmatch(val); p != nil {
				cur.Signal, _ = strconv.Atoi(p[1])
			}
		}
	}
	flush()
	return list
}

func ParseProcWireless(out, iface string) (int, bool) {
	for _, line := range strings.Split(out, "\n") {
		name, rest, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || name != iface {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 2 {
			return 0, true
		}
		q, err := strconv.ParseFloat(strings.TrimSuffix(f[1], "."), 64)
		if err != nil {
			return 0, true
		}
		return min(100, max(0, int(q*100/70+0.5))), true
	}
	return 0, false
}

func ParseHardwarePorts(out string) map[string]string {
	ports := map[string]string{}
	port := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "Hardware Port:"); ok {
			port = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "Device:"); ok && port != "" {
			ports[strings.TrimSpace(v)] = port
		}
	}
	return ports
}

func TypeOfPort(port string) string {
	p := strings.ToLower(port)
	switch {
	case strings.Contains(p, "wi-fi"), strings.Contains(p, "airport"):
		return Wifi
	case strings.Contains(p, "ethernet"), strings.Contains(p, "lan"), strings.Contains(p, "thunderbolt"):
		return Ethernet
	}
	return Other
}

type Cache struct {
	Every  time.Duration
	detect func() Info
	mu     sync.Mutex
	info   Info
	at     time.Time
}

func NewCache() *Cache { return &Cache{Every: time.Minute, detect: Detect} }

func (c *Cache) Get() Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.at.IsZero() || time.Since(c.at) >= c.Every {
		c.info, c.at = c.detect(), time.Now()
	}
	return c.info
}
