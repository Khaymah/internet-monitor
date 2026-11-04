package power

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var kernBoot = regexp.MustCompile(`sec\s*=\s*(\d+)`)

func ParseProcStat(out string) (time.Time, bool) {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err == nil && n > 0 {
				return time.Unix(n, 0), true
			}
		}
	}
	return time.Time{}, false
}

func ParseKernBoottime(out string) (time.Time, bool) {
	m := kernBoot.FindStringSubmatch(out)
	if m == nil {
		return time.Time{}, false
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}, false
	}
	return time.Unix(n, 0), true
}

func ParsePmset(out string) (onBattery, known bool) {
	if !strings.Contains(out, "InternalBattery") {
		return false, false
	}
	switch {
	case strings.Contains(out, "'Battery Power'"):
		return true, true
	case strings.Contains(out, "'AC Power'"):
		return false, true
	}
	return false, false
}
