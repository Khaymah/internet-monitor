package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const descXML = `<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
 <device>
  <deviceType>urn:schemas-upnp-org:device:InternetGatewayDevice:1</deviceType>
  <friendlyName>Home Gateway</friendlyName>
  <modelName>ZTE H188A</modelName>
  <deviceList><device>
   <deviceType>urn:schemas-upnp-org:device:WANDevice:1</deviceType>
   <serviceList><service>
    <serviceType>urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1</serviceType>
    <controlURL>/ctl/common</controlURL>
   </service></serviceList>
   <deviceList><device>
    <deviceType>urn:schemas-upnp-org:device:WANConnectionDevice:1</deviceType>
    <serviceList>
     <service>
      <serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType>
      <controlURL>http://203.0.113.9/evil</controlURL>
     </service>
     <service>
      <serviceType>urn:schemas-upnp-org:service:WANPPPConnection:1</serviceType>
      <controlURL>/ctl/ppp</controlURL>
     </service>
    </serviceList>
   </device></deviceList>
  </device></deviceList>
 </device>
</root>`

func soap(action string, fields map[string]string) string {
	var b strings.Builder
	for k, v := range fields {
		fmt.Fprintf(&b, "<%s>%s</%s>", k, v, k)
	}
	return `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:` +
		action + `Response xmlns:u="x">` + b.String() + `</u:` + action + `Response></s:Body></s:Envelope>`
}

type fakeIGD struct {
	link, conn string
	rx, tx     uint64
	uptime     int64
}

func (f *fakeIGD) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/desc.xml", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, descXML) })
	mux.HandleFunc("/ctl/", func(w http.ResponseWriter, r *http.Request) {
		action := r.Header.Get("SOAPAction")
		action = action[strings.Index(action, "#")+1 : len(action)-1]
		switch action {
		case "GetCommonLinkProperties":
			io.WriteString(w, soap(action, map[string]string{
				"NewWANAccessType": "DSL", "NewPhysicalLinkStatus": f.link,
				"NewLayer1DownstreamMaxBitRate": "71234000", "NewLayer1UpstreamMaxBitRate": "10200000",
			}))
		case "GetTotalBytesReceived":
			io.WriteString(w, soap(action, map[string]string{"NewTotalBytesReceived": fmt.Sprint(f.rx)}))
		case "GetTotalBytesSent":
			io.WriteString(w, soap(action, map[string]string{"NewTotalBytesSent": fmt.Sprint(f.tx)}))
		case "GetStatusInfo":
			if r.URL.Path != "/ctl/ppp" {
				http.Error(w, "wrong service", http.StatusBadRequest)
				return
			}
			io.WriteString(w, soap(action, map[string]string{"NewConnectionStatus": f.conn, "NewUptime": fmt.Sprint(f.uptime)}))
		default:
			http.Error(w, "unknown", http.StatusInternalServerError)
		}
	})
	return mux
}

func TestOpenAndStatus(t *testing.T) {
	f := &fakeIGD{link: "Up", conn: "Connected", rx: 5000, tx: 700, uptime: 3600}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	gw, err := Open(context.Background(), srv.URL+"/desc.xml", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gw.Model != "ZTE H188A" {
		t.Fatalf("model %q", gw.Model)
	}
	if !strings.HasSuffix(gw.connURL, "/ctl/ppp") {
		t.Fatalf("a control URL on another host must be ignored, got %q", gw.connURL)
	}
	st, err := gw.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.LinkUp == nil || !*st.LinkUp || st.Connected == nil || !*st.Connected {
		t.Fatalf("status %+v", st)
	}
	if st.SyncDownKbps != 71234 || st.SyncUpKbps != 10200 || st.UptimeSecs != 3600 || st.AccessType != "DSL" {
		t.Fatalf("status %+v", st)
	}
	if !st.HasCounters || st.RxBytes != 5000 || st.TxBytes != 700 {
		t.Fatalf("counters %+v", st)
	}
}

func TestOpenRejectsRouterWithoutWAN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `<root><device><modelName>X</modelName></device></root>`)
	}))
	defer srv.Close()
	if _, err := Open(context.Background(), srv.URL, nil); err == nil {
		t.Fatal("expected an error")
	}
}

func TestParseSSDP(t *testing.T) {
	ok := "HTTP/1.1 200 OK\r\nST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\nLOCATION: http://192.168.1.1:52869/gatedesc.xml\r\n\r\n"
	if got := ParseSSDP([]byte(ok)); got != "http://192.168.1.1:52869/gatedesc.xml" {
		t.Fatalf("got %q", got)
	}
	public := strings.Replace(ok, "192.168.1.1", "8.8.8.8", 1)
	if got := ParseSSDP([]byte(public)); got != "" {
		t.Fatalf("a public location must be ignored, got %q", got)
	}
	other := strings.Replace(ok, "InternetGatewayDevice", "MediaRenderer", 1)
	if got := ParseSSDP([]byte(other)); got != "" {
		t.Fatalf("a non-gateway device must be ignored, got %q", got)
	}
}

func at(min int) time.Time {
	return time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC).Add(time.Duration(min) * time.Minute)
}

func sample(min int, rx, tx uint64, uptime int64) Status {
	return Status{Time: at(min), RxBytes: rx, TxBytes: tx, UptimeSecs: uptime, HasCounters: true}
}

func TestUsageHandlesWrapAndReboot(t *testing.T) {
	samples := []Status{
		sample(0, 1000, 100, 600),
		sample(1, 3000, 200, 660),
		sample(2, counterWrap-100, 300, 720),
		sample(3, 400, 350, 780),
		sample(4, 50, 10, 30),
		sample(5, 150, 20, 90),
	}
	got := Usage(samples, at(0), at(10))
	want := uint64(2000+100) + uint64(counterWrap-100-3000+100) + uint64(500+50) + uint64(50+10) + uint64(100+10)
	if got != want {
		t.Fatalf("usage %d, want %d", got, want)
	}
	if got := Usage(samples, at(4), at(10)); got != 110 {
		t.Fatalf("usage in window %d", got)
	}
}

func TestPeriodAt(t *testing.T) {
	loc := time.UTC
	p := PeriodAt(time.Date(2026, 11, 5, 9, 0, 0, 0, loc), 10)
	if !p.From.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, loc)) || !p.To.Equal(time.Date(2026, 11, 10, 0, 0, 0, 0, loc)) {
		t.Fatalf("period %+v", p)
	}
	p = PeriodAt(time.Date(2026, 11, 15, 9, 0, 0, 0, loc), 31)
	if p.From.Day() != 28 || p.From.Month() != time.October {
		t.Fatalf("reset day must clamp to 28: %+v", p)
	}
	if e := PeriodAt(at(0), 1).Elapsed(at(0)); e <= 0.1 || e >= 0.2 {
		t.Fatalf("elapsed %v", e)
	}
}

func up(v bool) *bool { return &v }

func TestClassify(t *testing.T) {
	start, end := at(10), at(14)
	ok := func(min int, uptime int64) Status {
		return Status{Time: at(min), LinkUp: up(true), Connected: up(true), UptimeSecs: uptime}
	}
	cases := []struct {
		name    string
		samples []Status
		want    string
	}{
		{"line", []Status{ok(9, 1000), {Time: at(11), LinkUp: up(false), UptimeSecs: 1120}}, CauseLineDown},
		{"restart", []Status{ok(9, 1000), ok(12, 40)}, CauseRestarted},
		{"session", []Status{ok(9, 1000), {Time: at(11), LinkUp: up(true), Connected: up(false), UptimeSecs: 1120}}, CauseSession},
		{"online", []Status{ok(9, 1000), ok(11, 1120), ok(12, 1180)}, CauseOnline},
		{"no data", []Status{ok(1, 500)}, ""},
	}
	for _, c := range cases {
		if got := Classify(c.samples, start, end); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPollerFindsSavesAndForgets(t *testing.T) {
	f := &fakeIGD{link: "Up", conn: "Connected", rx: 10, tx: 20, uptime: 99}
	srv := httptest.NewServer(f.handler())
	clock := at(0)
	var saved []Status
	discovered := 0
	p := &Poller{
		Discover: func(context.Context) (string, error) { discovered++; return srv.URL + "/desc.xml", nil },
		Save:     func(_ time.Time, s Status) { saved = append(saved, s) },
		Now:      func() time.Time { return clock },
	}
	p.Tick(context.Background())
	if len(saved) != 1 || saved[0].Model != "ZTE H188A" || !saved[0].Time.Equal(clock) {
		t.Fatalf("saved %+v", saved)
	}
	if snap := p.Snapshot(); !snap.Found || snap.Status == nil {
		t.Fatalf("snapshot %+v", snap)
	}
	srv.Close()
	for range maxFailures {
		clock = clock.Add(time.Minute)
		p.Tick(context.Background())
	}
	if snap := p.Snapshot(); snap.Found || snap.Error == "" {
		t.Fatalf("after %d failures the router must be dropped: %+v", maxFailures, snap)
	}
	clock = clock.Add(time.Minute)
	p.Tick(context.Background())
	if discovered != 1 {
		t.Fatalf("rediscovery must wait %v, discovered %d times", rediscover, discovered)
	}
	p.Enabled = func() bool { return false }
	p.Tick(context.Background())
	if snap := p.Snapshot(); snap.Status != nil {
		t.Fatalf("disabled poller must forget the last status: %+v", snap)
	}
}
