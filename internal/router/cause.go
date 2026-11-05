package router

import "time"

const (
	CauseLineDown  = "line_down"
	CauseSession   = "session_down"
	CauseDropped   = "provider_dropped"
	CauseRestarted = "router_restarted"
	CauseOnline    = "router_online"
)

func Classify(samples []Status, start, end time.Time) string {
	var during []Status
	var before *Status
	for i := range samples {
		s := samples[i]
		switch {
		case s.Time.Before(start):
			before = &samples[i]
		case !s.Time.After(end.Add(2 * time.Minute)):
			during = append(during, s)
		}
	}
	if len(during) == 0 {
		return ""
	}
	for _, s := range during {
		if s.LinkUp != nil && !*s.LinkUp {
			return CauseLineDown
		}
	}
	for i, s := range during {
		prev := before
		if i > 0 {
			prev = &during[i-1]
		}
		if prev != nil && rebooted(*prev, s) {
			return CauseRestarted
		}
	}
	for _, s := range during {
		if s.Connected != nil && !*s.Connected {
			for _, after := range during {
				if providerError(after.LastError) {
					return CauseDropped
				}
			}
			return CauseSession
		}
	}
	for _, s := range during {
		if !s.Time.After(end) && s.Connected != nil && *s.Connected {
			return CauseOnline
		}
	}
	return ""
}

func providerError(code string) bool {
	return code == "ERROR_ISP_DISCONNECT" || code == "ERROR_ISP_TIME_OUT"
}
