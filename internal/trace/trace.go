package trace

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	MaxHops = 15
	Timeout = 45 * time.Second
)

var ErrUnavailable = errors.New("no traceroute tool on this system")

type Hop struct {
	N     int     `json:"n"`
	Addr  string  `json:"addr,omitempty"`
	RTTms float64 `json:"rtt_ms,omitempty"`
}

type Trace struct {
	Time      time.Time `json:"timestamp"`
	EventTime time.Time `json:"event_time"`
	Target    string    `json:"target"`
	Tool      string    `json:"tool"`
	Hops      []Hop     `json:"hops"`
	Reached   bool      `json:"reached"`
	LastHop   int       `json:"last_hop"`
	LastAddr  string    `json:"last_addr,omitempty"`
	Raw       string    `json:"raw"`
}

type command struct {
	tool string
	args []string
}

func commands(target string) []command {
	if runtime.GOOS == "windows" {
		return []command{{"tracert", []string{"-d", "-h", strconv.Itoa(MaxHops), "-w", "800", target}}}
	}
	return []command{
		{"traceroute", []string{"-n", "-m", strconv.Itoa(MaxHops), "-w", "1", "-q", "1", target}},
		{"tracepath", []string{"-n", "-m", strconv.Itoa(MaxHops), target}},
	}
}

func Run(ctx context.Context, target string) (Trace, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	for _, c := range commands(target) {
		path, err := exec.LookPath(c.tool)
		if err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, path, c.args...)
		hideWindow(cmd)
		out, err := cmd.Output()
		if len(out) == 0 && err != nil {
			return Trace{}, err
		}
		t := Parse(string(out), target)
		t.Tool = c.tool
		return t, nil
	}
	return Trace{}, ErrUnavailable
}

var (
	hopLine = regexp.MustCompile(`^\s*(\d+):?\s+(.*)$`)
	ipv4    = regexp.MustCompile(`\b(\d{1,3}\.){3}\d{1,3}\b`)
	rttRe   = regexp.MustCompile(`([<\d.]+)\s*ms`)
)

func Parse(raw, target string) Trace {
	t := Trace{Target: target, Raw: strings.TrimSpace(raw), Hops: []Hop{}}
	targetIP := target
	if h, _, err := net.SplitHostPort(target); err == nil {
		targetIP = h
	}
	seen := map[int]bool{}
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		m := hopLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 || n > 64 || seen[n] {
			continue
		}
		rest := m[2]
		hop := Hop{N: n}
		if addr := ipv4.FindString(rest); addr != "" && !strings.Contains(strings.ToLower(rest), "pmtu") {
			hop.Addr = addr
			if r := rttRe.FindStringSubmatch(rest); r != nil {
				v, _ := strconv.ParseFloat(strings.TrimPrefix(r[1], "<"), 64)
				hop.RTTms = v
			}
		} else if strings.Contains(strings.ToLower(rest), "pmtu") {
			continue
		}
		seen[n] = true
		t.Hops = append(t.Hops, hop)
		if hop.Addr != "" {
			t.LastHop, t.LastAddr = n, hop.Addr
			if hop.Addr == targetIP {
				t.Reached = true
			}
		}
	}
	return t
}
