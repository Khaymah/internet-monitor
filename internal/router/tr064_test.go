package router

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

const trDescXML = `<?xml version="1.0"?>
<root xmlns="urn:dslforum-org:device-1-0">
 <device>
  <deviceType>urn:dslforum-org:device:InternetGatewayDevice:1</deviceType>
  <friendlyName>FRITZ!Box 7590</friendlyName>
  <modelName>FRITZ!Box 7590</modelName>
  <serviceList>
   <service><serviceType>urn:dslforum-org:service:DeviceInfo:1</serviceType><controlURL>/upnp/control/deviceinfo</controlURL></service>
  </serviceList>
  <deviceList>
   <device>
    <serviceList>
     <service><serviceType>urn:dslforum-org:service:WANCommonInterfaceConfig:1</serviceType><controlURL>/upnp/control/wancommonifconfig1</controlURL></service>
     <service><serviceType>urn:dslforum-org:service:WANDSLInterfaceConfig:1</serviceType><controlURL>/upnp/control/wandslifconfig1</controlURL></service>
    </serviceList>
    <deviceList><device><serviceList>
     <service><serviceType>urn:dslforum-org:service:WANPPPConnection:1</serviceType><controlURL>/upnp/control/wanpppconn1</controlURL></service>
     <service><serviceType>urn:dslforum-org:service:WANIPConnection:1</serviceType><controlURL>http://203.0.113.9/evil</controlURL></service>
    </serviceList></device></deviceList>
   </device>
  </deviceList>
 </device>
</root>`

var authField = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|([^,\s]+))`)

type fakeTR struct {
	mu          sync.Mutex
	user        string
	password    string
	open        bool
	dsl         bool
	avm         bool
	nonces      int
	anonCalls   int
	authedOK    int
	authedFails int
}

func md5hex(parts ...string) string {
	sum := md5.Sum([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(sum[:])
}

func (f *fakeTR) authorized(r *http.Request) (bool, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return false, false
	}
	fields := map[string]string{}
	for _, m := range authField.FindAllStringSubmatch(strings.TrimPrefix(h, "Digest "), -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		fields[m[1]] = v
	}
	ha1 := md5hex(f.user, "F!Box SOAP-Auth", f.password)
	ha2 := md5hex(r.Method, fields["uri"])
	want := md5hex(ha1, fields["nonce"], fields["nc"], fields["cnonce"], fields["qop"], ha2)
	return true, fields["username"] == f.user && fields["nonce"] == "nonce-1" && fields["response"] == want
}

func (f *fakeTR) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tr64desc.xml", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, trDescXML) })
	mux.HandleFunc("/upnp/control/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		sent, ok := f.authorized(r)
		switch {
		case !sent && f.open:
			f.anonCalls++
		case !sent:
			f.nonces++
			w.Header().Set("WWW-Authenticate", `Digest realm="F!Box SOAP-Auth", nonce="nonce-1", algorithm=MD5, qop="auth"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		case !ok:
			f.authedFails++
			w.Header().Set("WWW-Authenticate", `Digest realm="F!Box SOAP-Auth", nonce="nonce-1", algorithm=MD5, qop="auth"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		default:
			f.authedOK++
		}
		action := r.Header.Get("SOAPAction")
		action = action[strings.Index(action, "#")+1 : len(action)-1]
		f.answer(w, r.URL.Path, action)
	})
	return mux
}

func (f *fakeTR) answer(w http.ResponseWriter, path, action string) {
	switch {
	case strings.HasSuffix(path, "wancommonifconfig1") && action == "GetAddonInfos" && f.avm:
		io.WriteString(w, soap(action, map[string]string{
			"NewTotalBytesReceived": "5", "NewTotalBytesSent": "6",
			"NewX_AVM_DE_TotalBytesReceived64": "9000000000", "NewX_AVM_DE_TotalBytesSent64": "4500000000",
		}))
	case strings.HasSuffix(path, "wancommonifconfig1") && action == "GetTotalBytesReceived":
		io.WriteString(w, soap(action, map[string]string{"NewTotalBytesReceived": "1234567"}))
	case strings.HasSuffix(path, "wancommonifconfig1") && action == "GetTotalBytesSent":
		io.WriteString(w, soap(action, map[string]string{"NewTotalBytesSent": "7654321"}))
	case strings.HasSuffix(path, "wandslifconfig1") && action == "GetInfo" && f.dsl:
		io.WriteString(w, soap(action, map[string]string{
			"NewStatus": "Up", "NewDownstreamCurrRate": "40957", "NewUpstreamCurrRate": "10000",
			"NewDownstreamMaxRate": "65532", "NewUpstreamMaxRate": "17271",
			"NewDownstreamNoiseMargin": "61", "NewUpstreamNoiseMargin": "159",
			"NewDownstreamAttenuation": "133", "NewUpstreamAttenuation": "52",
		}))
	case strings.HasSuffix(path, "wanpppconn1") && action == "GetStatusInfo":
		io.WriteString(w, soap(action, map[string]string{
			"NewConnectionStatus": "Connected", "NewUptime": "4242", "NewLastConnectionError": "ERROR_ISP_DISCONNECT",
		}))
	case strings.HasSuffix(path, "deviceinfo") && action == "GetInfo":
		io.WriteString(w, soap(action, map[string]string{"NewModelName": "FRITZ!Box 7590", "NewUpTime": "99999"}))
	default:
		http.Error(w, "Invalid Action", http.StatusInternalServerError)
	}
}

func trProbe(t *testing.T, srv *httptest.Server) *Probe {
	t.Helper()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	p, err := NewProbe("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	p.trHTTP = port
	return p
}

func TestDigestMatchesRFC2617Example(t *testing.T) {
	c := digestChallenge{realm: "testrealm@host.com", nonce: "dcd98b7102dd2f0e8b11d0f600bfb0c093", qop: "auth,auth-int"}
	got := c.authorize("Mufasa", "Circle Of Life", "GET", "/dir/index.html", 1, "0a4f113b")
	if !strings.Contains(got, `response="6629fae49393a05397450978507c4ef1"`) {
		t.Fatalf("digest response is wrong: %s", got)
	}
}

func TestTR064ReadsFritzStyleRouter(t *testing.T) {
	f := &fakeTR{user: "fritz", password: "s3cret", dsl: true, avm: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	p := trProbe(t, srv)
	ok, err := tr064{}.Detect(context.Background(), p)
	if !ok || err != nil {
		t.Fatalf("detect: %v %v", ok, err)
	}
	st, err := tr064{}.Read(context.Background(), p, Creds{User: "fritz", Password: "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasCounters || st.RxBytes != 9000000000 || st.TxBytes != 4500000000 {
		t.Fatalf("a 64-bit counter must be preferred: %+v", st)
	}
	if !st.HasLineStats || st.Status != "Up" || st.DownKbps != 40957 || st.UpKbps != 10000 || st.MaxDownKbps != 65532 || st.MaxUpKbps != 17271 {
		t.Fatalf("line rates %+v", st)
	}
	if st.SNRDown != 6.1 || st.SNRUp != 15.9 || st.AttnDown != 13.3 || st.AttnUp != 5.2 {
		t.Fatalf("margins and attenuation are in 0.1 dB: %+v", st)
	}
	if st.Connected == nil || !*st.Connected || st.UptimeSecs != 4242 || st.LastError != "ERROR_ISP_DISCONNECT" || st.Model != "FRITZ!Box 7590" {
		t.Fatalf("connection %+v", st)
	}
	if f.authedFails != 0 || f.nonces != 1 {
		t.Fatalf("the challenge must be answered once and reused: nonces %d, failures %d", f.nonces, f.authedFails)
	}
}

func TestTR064FallsBackTo32BitCounters(t *testing.T) {
	f := &fakeTR{user: "u", password: "p", open: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	st, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{})
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasCounters || st.RxBytes != 1234567 || st.TxBytes != 7654321 {
		t.Fatalf("counters %+v", st)
	}
}

func TestTR064TriesWithoutPasswordFirst(t *testing.T) {
	f := &fakeTR{user: "u", password: "p", open: true, dsl: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	st, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{User: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasLineStats || f.authedOK != 0 || f.nonces != 0 || f.anonCalls == 0 {
		t.Fatalf("an open router must be read without sending the password: %+v anon %d authed %d", st, f.anonCalls, f.authedOK)
	}
}

func TestTR064WrongPasswordIsTriedOnce(t *testing.T) {
	f := &fakeTR{user: "fritz", password: "right", dsl: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	_, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{User: "fritz", Password: "wrong"})
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("got %v", err)
	}
	if f.authedFails != 1 {
		t.Fatalf("a rejected password must be sent once, sent %d times", f.authedFails)
	}
}

func TestTR064NeedsPasswordWhenNoneSet(t *testing.T) {
	f := &fakeTR{user: "fritz", password: "right"}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	_, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{})
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("got %v", err)
	}
	if f.authedFails != 0 || f.authedOK != 0 {
		t.Fatal("no credentials may be invented")
	}
}

func TestTR064RouterWithoutDSLService(t *testing.T) {
	f := &fakeTR{user: "u", password: "p", dsl: false, avm: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	st, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{User: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if st.HasLineStats || !st.HasCounters || st.Connected == nil || !*st.Connected {
		t.Fatalf("a fiber or cable router has counters and a session but no DSL figures: %+v", st)
	}
	var merged Status
	merge(&merged, st)
	if merged.SNRDown != nil || !merged.HasCounters || merged.Connected == nil || merged.UptimeSecs != 4242 {
		t.Fatalf("merged %+v", merged)
	}
}

func TestTR064IgnoresServicesOnOtherHosts(t *testing.T) {
	f := &fakeTR{user: "u", password: "p", open: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	p := trProbe(t, srv)
	desc, err := loadTRDesc(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if _, bad := desc.services[svcTRIP]; bad {
		t.Fatal("a control URL on another host must be dropped")
	}
}

func TestTR064NotDetectedOnOtherRouters(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	ok, _ := tr064{}.Detect(context.Background(), trProbe(t, srv))
	if ok {
		t.Fatal("a router without /tr64desc.xml is not TR-064")
	}
}

func TestTR064ForbiddenActionAfterLoginIsNotAWrongPassword(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/tr64desc.xml", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, trDescXML) })
	mux.HandleFunc("/upnp/control/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") == "" || strings.HasSuffix(r.URL.Path, "wandslifconfig1") {
			w.Header().Set("WWW-Authenticate", `Digest realm="x", nonce="n", qop="auth"`)
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		io.WriteString(w, soap("GetStatusInfo", map[string]string{"NewConnectionStatus": "Connected", "NewUptime": fmt.Sprint(calls)}))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	st, err := tr064{}.Read(context.Background(), trProbe(t, srv), Creds{User: "u", Password: "p"})
	if err != nil || st.Connected == nil {
		t.Fatalf("an action the account may not use must be skipped once signed in: %v %+v", err, st)
	}
}
