package router

import (
	"sort"
	"time"
)

const counterWrap = uint64(1) << 32

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func PeriodAt(now time.Time, resetDay int) Period {
	day := min(max(resetDay, 1), 28)
	y, m, d := now.Date()
	from := time.Date(y, m, day, 0, 0, 0, 0, now.Location())
	if d < day {
		from = from.AddDate(0, -1, 0)
	}
	return Period{From: from, To: from.AddDate(0, 1, 0)}
}

func (p Period) Elapsed(now time.Time) float64 {
	total := p.To.Sub(p.From)
	if total <= 0 {
		return 0
	}
	return min(max(float64(now.Sub(p.From))/float64(total), 0), 1)
}

func rebooted(a, b Status) bool {
	if a.UptimeSecs <= 0 || b.UptimeSecs <= 0 {
		return false
	}
	return b.UptimeSecs+60 < a.UptimeSecs+int64(b.Time.Sub(a.Time).Seconds())
}

func delta(prev, cur uint64, reboot bool) uint64 {
	switch {
	case reboot:
		return cur
	case cur >= prev:
		return cur - prev
	case prev < counterWrap:
		return cur + counterWrap - prev
	default:
		return cur
	}
}

func Usage(samples []Status, from, to time.Time) uint64 {
	var in []Status
	for _, s := range samples {
		if s.HasCounters && !s.Time.Before(from) && s.Time.Before(to) {
			in = append(in, s)
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].Time.Before(in[j].Time) })
	var total uint64
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		reboot := rebooted(a, b)
		total += delta(a.RxBytes, b.RxBytes, reboot) + delta(a.TxBytes, b.TxBytes, reboot)
	}
	return total
}
