package report

import (
	"internet-monitor/internal/types"
	"sort"
	"time"
)

const SegmentOff = "off"

type Segment struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Status string    `json:"status"`
}

func Timeline(events []types.Event, samples []types.MetricSample, from, to time.Time) []Segment {
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	status := types.StatusConnected.String()
	var raw []Segment
	cursor := from
	for _, ev := range events {
		if !ev.Timestamp.After(from) {
			status = ev.EventType
			continue
		}
		if !ev.Timestamp.Before(to) {
			break
		}
		raw = append(raw, Segment{Start: cursor, End: ev.Timestamp, Status: status})
		cursor, status = ev.Timestamp, ev.EventType
	}
	raw = append(raw, Segment{Start: cursor, End: to, Status: status})

	inWindow := make([]types.MetricSample, 0, len(samples))
	for _, s := range samples {
		if !s.Timestamp.Before(from) && s.Timestamp.Before(to) {
			inWindow = append(inWindow, s)
		}
	}
	var off []gap
	if len(inWindow) == 0 {
		off = []gap{{start: from, end: to}}
	} else {
		first := inWindow[0].Timestamp
		for _, s := range inWindow {
			if s.Timestamp.Before(first) {
				first = s.Timestamp
			}
		}
		if first.Sub(from) >= minGap {
			off = append(off, gap{start: from, end: first})
		}
		off = append(off, findGaps(inWindow, to)...)
	}

	var out []Segment
	for _, seg := range raw {
		out = append(out, subtract(seg, off)...)
	}
	for _, g := range off {
		out = append(out, Segment{Start: g.start, End: g.end, Status: SegmentOff})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return merge(out)
}

func subtract(seg Segment, off []gap) []Segment {
	parts := []Segment{seg}
	for _, g := range off {
		var next []Segment
		for _, p := range parts {
			if !g.start.Before(p.End) || !g.end.After(p.Start) {
				next = append(next, p)
				continue
			}
			if g.start.After(p.Start) {
				next = append(next, Segment{Start: p.Start, End: g.start, Status: p.Status})
			}
			if g.end.Before(p.End) {
				next = append(next, Segment{Start: g.end, End: p.End, Status: p.Status})
			}
		}
		parts = next
	}
	return parts
}

func merge(in []Segment) []Segment {
	out := make([]Segment, 0, len(in))
	for _, s := range in {
		if !s.End.After(s.Start) {
			continue
		}
		if n := len(out); n > 0 && out[n-1].Status == s.Status && !s.Start.After(out[n-1].End) {
			if s.End.After(out[n-1].End) {
				out[n-1].End = s.End
			}
			continue
		}
		out = append(out, s)
	}
	return out
}
