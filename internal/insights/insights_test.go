package insights

import (
	"internet-monitor/internal/report"
	"testing"
)

func ids(list []Insight) map[string]Insight {
	out := map[string]Insight{}
	for _, i := range list {
		out[i.ID] = i
	}
	return out
}

func TestAnalyzeFindsTheRightAdvice(t *testing.T) {
	var hours [24]report.Hour
	hours[21].DowntimeSecs, hours[22].DowntimeSecs, hours[9].DowntimeSecs = 3000, 2000, 500
	route := &report.Route{LastHop: 1, LastAddr: "192.168.1.1"}
	s := report.MonthlySummary{
		WindowSeconds:  7 * 86400,
		CoveragePct:    65,
		Disconnections: 4,
		Causes:         map[string]report.CauseStat{"tcp": {Outages: 1}, "dns": {Outages: 3}},
		Hours:          hours,
		Links:          report.Links{Minutes: 5000, WifiPct: 80, AvgSignalPct: 35},
		Events: []report.EventRow{
			{Type: "disconnected", Route: route},
			{Type: "disconnected", Route: route},
			{Type: "disconnected"},
			{Type: "disconnected", OnBattery: true},
		},
		Gaps: []report.GapRow{{Kind: report.GapPower}},
	}
	speed := report.SpeedSummary{Tests: 4, PlanDownloadMbps: 70, PctOfPlan: 40, PeakTests: 2, PeakMedian: 10, OffPeakMedian: 40}
	got := ids(Analyze(Input{Summary: s, Speed: speed, Grades: []string{"D", "C", "A"}, Days: 7}))
	for _, want := range []string{"coverage_low", "provider_route", "dns_failures", "peak_hours", "bufferbloat", "weak_wifi", "power_cuts", "slow_vs_plan", "slow_evenings", "file_complaint"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing insight %q (got %v)", want, got)
		}
	}
	if p := got["peak_hours"]; p.Vars["from"] != 19 && p.Vars["from"] != 20 && p.Vars["from"] != 21 {
		t.Errorf("peak window = %v", p.Vars)
	}
	if got["bufferbloat"].Vars["grade"] != "D" {
		t.Errorf("worst grade = %v", got["bufferbloat"].Vars)
	}
	if _, ok := got["all_good"]; ok {
		t.Error("all_good must not show next to problems")
	}

	quiet := ids(Analyze(Input{Summary: s, Speed: speed, Grades: []string{"D", "C"}, Days: 7, OpenOrRecentComplaint: true}))
	if _, ok := quiet["file_complaint"]; ok {
		t.Error("no complaint nudge while a complaint is already open")
	}
}

func TestAnalyzeAllGood(t *testing.T) {
	s := report.MonthlySummary{WindowSeconds: 7 * 86400, CoveragePct: 99, Causes: map[string]report.CauseStat{}}
	got := Analyze(Input{Summary: s, Days: 7})
	if len(got) != 1 || got[0].ID != "all_good" || got[0].Level != Good {
		t.Fatalf("got %+v", got)
	}
}
