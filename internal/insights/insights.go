package insights

import (
	"internet-monitor/internal/report"
	"math"
	"sort"
)

const (
	Warn = "warn"
	Info = "info"
	Good = "good"
)

type Insight struct {
	ID    string         `json:"id"`
	Level string         `json:"level"`
	Vars  map[string]any `json:"vars,omitempty"`
}

type Input struct {
	Summary               report.MonthlySummary
	Speed                 report.SpeedSummary
	Grades                []string
	Days                  int
	OpenOrRecentComplaint bool
}

func Analyze(in Input) []Insight {
	s := in.Summary
	var out []Insight
	add := func(id, level string, vars map[string]any) {
		out = append(out, Insight{ID: id, Level: level, Vars: vars})
	}

	if s.WindowSeconds >= 3*86400 && s.CoveragePct < 80 {
		add("coverage_low", Warn, map[string]any{"pct": math.Round(s.CoveragePct)})
	}

	providerSide, power, routerOnly := 0, 0, 0
	for _, e := range s.Events {
		if e.Type != "disconnected" {
			continue
		}
		if e.OnBattery || e.AfterGap {
			power++
			continue
		}
		providerSide++
		if e.Route != nil && !e.Route.Reached && e.Route.LastHop == 1 {
			routerOnly++
		}
	}
	for _, g := range s.Gaps {
		if g.Kind == report.GapPower {
			power++
		}
	}

	if routerOnly >= 2 {
		add("provider_route", Warn, map[string]any{"n": routerOnly})
	}

	if s.Disconnections >= 3 && s.Causes["dns"].Outages*2 >= s.Disconnections && s.Causes["tcp"].Outages*2 < s.Disconnections {
		add("dns_failures", Warn, map[string]any{"n": s.Causes["dns"].Outages})
	}

	if s.Disconnections >= 3 {
		if from, share := busiestWindow(s.Hours, 4); share >= 60 {
			add("peak_hours", Info, map[string]any{"from": from, "to": (from + 4) % 24, "pct": math.Round(share)})
		}
	}

	if bad := countBad(in.Grades); len(in.Grades) >= 2 && bad*2 > len(in.Grades) {
		add("bufferbloat", Warn, map[string]any{"grade": worst(in.Grades)})
	}

	if s.Links.Minutes > 60 && s.Links.WifiPct >= 50 && s.Links.AvgSignalPct > 0 && s.Links.AvgSignalPct < 50 {
		add("weak_wifi", Warn, map[string]any{"signal": s.Links.AvgSignalPct})
	} else if s.Links.Minutes > 60 && s.Links.WifiPct >= 50 && providerSide >= 3 {
		add("use_cable", Info, nil)
	}

	if power >= 2 {
		add("power_cuts", Info, map[string]any{"n": power})
	}

	if in.Speed.Tests >= 3 && in.Speed.PlanDownloadMbps > 0 && in.Speed.PctOfPlan < 50 {
		add("slow_vs_plan", Warn, map[string]any{"pct": math.Round(in.Speed.PctOfPlan)})
	}
	if in.Speed.PeakTests >= 2 && in.Speed.OffPeakMedian > 0 && in.Speed.PeakMedian < in.Speed.OffPeakMedian*0.6 {
		add("slow_evenings", Info, map[string]any{
			"peak":    math.Round(in.Speed.PeakMedian*10) / 10,
			"offpeak": math.Round(in.Speed.OffPeakMedian*10) / 10,
		})
	}

	if providerSide >= 3 && !in.OpenOrRecentComplaint {
		add("file_complaint", Warn, map[string]any{"n": providerSide, "days": in.Days})
	}

	if len(out) == 0 && s.WindowSeconds >= 86400 && s.Disconnections == 0 && s.CoveragePct >= 90 {
		add("all_good", Good, map[string]any{"days": in.Days})
	}

	rank := map[string]int{Warn: 0, Info: 1, Good: 2}
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Level] < rank[out[j].Level] })
	return out
}

func busiestWindow(h [24]report.Hour, width int) (int, float64) {
	var total float64
	for _, x := range h {
		total += x.DowntimeSecs
	}
	if total <= 0 {
		return 0, 0
	}
	best, bestSum := 0, -1.0
	for start := 0; start < 24; start++ {
		var sum float64
		for k := 0; k < width; k++ {
			sum += h[(start+k)%24].DowntimeSecs
		}
		if sum > bestSum {
			best, bestSum = start, sum
		}
	}
	return best, bestSum / total * 100
}

var gradeRank = map[string]int{"A+": 0, "A": 1, "B": 2, "C": 3, "D": 4, "F": 5}

func countBad(grades []string) int {
	n := 0
	for _, g := range grades {
		if gradeRank[g] >= 3 {
			n++
		}
	}
	return n
}

func worst(grades []string) string {
	w := ""
	for _, g := range grades {
		if r, ok := gradeRank[g]; ok && (w == "" || r > gradeRank[w]) {
			w = g
		}
	}
	return w
}
