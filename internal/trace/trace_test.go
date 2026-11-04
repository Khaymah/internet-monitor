package trace

import "testing"

const windowsOut = `
Tracing route to 8.8.8.8 over a maximum of 15 hops

  1    <1 ms    <1 ms    <1 ms  192.168.1.1
  2     *        *        *     Request timed out.
  3     *        *        *     Request timed out.

Trace complete.
`

const unixOut = `traceroute to 8.8.8.8 (8.8.8.8), 15 hops max, 60 byte packets
 1  192.168.1.1  0.512 ms
 2  10.20.0.1  8.211 ms
 3  *
 4  8.8.8.8  21.4 ms
`

const tracepathOut = ` 1?: [LOCALHOST]                      pmtu 1500
 1:  192.168.1.1                                           0.512ms
 2:  no reply
 3:  no reply
     Too many hops: pmtu 1500
`

func TestParseWindowsStopsAfterRouter(t *testing.T) {
	tr := Parse(windowsOut, "8.8.8.8")
	if len(tr.Hops) != 3 || tr.LastHop != 1 || tr.LastAddr != "192.168.1.1" || tr.Reached {
		t.Fatalf("trace = %+v", tr)
	}
	if tr.Hops[0].RTTms != 1 && tr.Hops[0].RTTms != 0 {
		t.Errorf("hop 1 rtt = %v", tr.Hops[0].RTTms)
	}
	if tr.Hops[1].Addr != "" {
		t.Errorf("a timed-out hop must have no address: %+v", tr.Hops[1])
	}
}

func TestParseUnixReachesTarget(t *testing.T) {
	tr := Parse(unixOut, "8.8.8.8:53")
	if len(tr.Hops) != 4 || !tr.Reached || tr.LastHop != 4 || tr.Hops[1].RTTms != 8.211 {
		t.Fatalf("trace = %+v", tr)
	}
}

func TestParseTracepath(t *testing.T) {
	tr := Parse(tracepathOut, "8.8.8.8")
	if tr.LastHop != 1 || tr.LastAddr != "192.168.1.1" || len(tr.Hops) != 3 {
		t.Fatalf("trace = %+v", tr)
	}
}
