package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	zteSessToken = "2449946313382634"
	zteLoginSeed = "3a6f09d2c1b84e57"
	zteSerial    = "ZTEEQKRM7802949"
	zteWANIP     = "203.0.113.45"
	zteWANMAC    = "aa:bb:cc:dd:ee:01"
)

func zteFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "zte-new", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type fakeZTE struct {
	t        *testing.T
	password string
	dsl      bool
	wan      bool
	busyMsg  string
	lockSecs int
	logins   atomic.Int32
	logouts  atomic.Int32
	mu       sync.Mutex
	views    map[string]bool
	signedIn bool
}

func newFakeZTE(t *testing.T, password string) *fakeZTE {
	return &fakeZTE{t: t, password: password, dsl: true, wan: true, views: map[string]bool{}}
}

func (f *fakeZTE) entry(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if f.lockSecs > 0 {
			io.WriteString(w, `{"lockingTime":`+itoa(f.lockSecs)+`,"loginErrMsg":"","promptMsg":"","sess_token":"`+zteSessToken+`"}`)
			return
		}
		io.WriteString(w, zteFixture(f.t, "entry.json"))
		return
	}
	f.logins.Add(1)
	if err := r.ParseForm(); err != nil {
		f.t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(f.password + zteLoginSeed))
	switch {
	case r.Form.Get("action") != "login" || r.Form.Get("_sessionTOKEN") != zteSessToken:
		io.WriteString(w, `{"loginErrMsg":"Session error","lockingTime":0,"sess_token":"`+zteSessToken+`"}`)
	case f.busyMsg != "":
		io.WriteString(w, `{"loginErrMsg":"`+f.busyMsg+`","lockingTime":0,"promptMsg":"","sess_token":"`+zteSessToken+`"}`)
	case r.Form.Get("Password") != hex.EncodeToString(sum[:]):
		io.WriteString(w, `{"loginErrMsg":"Password error","lockingTime":0,"promptMsg":"","sess_token":"`+zteSessToken+`"}`)
	default:
		f.mu.Lock()
		f.signedIn = true
		f.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "SID_HTTPS_", Value: "ok", Path: "/"})
		io.WriteString(w, `{"login_need_refresh":0,"lockingTime":0,"loginErrMsg":"","promptMsg":"","sess_token":"`+zteSessToken+`"}`)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func (f *fakeZTE) data(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("_tag")
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Query().Get("_type") == "menuView" {
		f.views[tag] = true
		io.WriteString(w, "<html></html>")
		return
	}
	timeout := `<ajax_response_xml_root><IF_ERRORPARAM>x</IF_ERRORPARAM><IF_ERRORTYPE>SUCC</IF_ERRORTYPE><IF_ERRORSTR>SessionTimeout</IF_ERRORSTR><IF_ERRORID>-1</IF_ERRORID></ajax_response_xml_root>`
	views := map[string]string{
		"dsl_interface_status_lua.lua": "dslWanStatus",
		"wan_internetstatus_lua.lua":   "ethWanStatus",
		"devmgr_statusmgr_lua.lua":     "statusMgr",
	}
	if !f.signedIn || !f.views[views[tag]] {
		io.WriteString(w, timeout)
		return
	}
	switch tag {
	case "dsl_interface_status_lua.lua":
		if f.dsl {
			io.WriteString(w, zteFixture(f.t, "dsl.xml"))
		} else {
			io.WriteString(w, zteFixture(f.t, "empty.xml"))
		}
	case "wan_internetstatus_lua.lua":
		if f.wan {
			io.WriteString(w, zteFixture(f.t, "wan.xml"))
		} else {
			io.WriteString(w, timeout)
		}
	case "devmgr_statusmgr_lua.lua":
		io.WriteString(w, zteFixture(f.t, "devinfo.xml"))
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeZTE) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path != "/":
			http.NotFound(w, r)
		case q.Get("_type") == "loginData" && q.Get("_tag") == "login_entry":
			f.entry(w, r)
		case q.Get("_type") == "loginData" && q.Get("_tag") == "login_token":
			io.WriteString(w, strings.Replace(zteFixture(f.t, "token.xml"), "3a6f09d2c1b84e57", zteLoginSeed, 1))
		case q.Get("_type") == "loginData" && q.Get("_tag") == "logout_entry":
			f.logouts.Add(1)
			f.mu.Lock()
			f.signedIn, f.views = false, map[string]bool{}
			f.mu.Unlock()
			io.WriteString(w, "{}")
		case q.Get("_type") == "menuView" || q.Get("_type") == "menuData":
			f.data(w, r)
		default:
			io.WriteString(w, `<html><head><title>ZTE</title></head><body>login</body></html>`)
		}
	})
}

func readZTE(t *testing.T, f *fakeZTE, password string) (WebStats, error) {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return WebReader{Host: hostOfServer(srv), Password: password}.Read(context.Background())
}

func TestZTENewReadsDSLGateway(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	st, err := readZTE(t, f, "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if st.Driver != DriverZTENew || !st.Experimental {
		t.Fatalf("driver %q experimental %v", st.Driver, st.Experimental)
	}
	if !st.HasLineStats || st.Status != "Up" || st.DownKbps != 40315 || st.UpKbps != 16094 || st.MaxDownKbps != 40879 || st.MaxUpKbps != 16094 {
		t.Fatalf("rates %+v", st)
	}
	if st.SNRDown != 5.9 || st.SNRUp != 5.9 || st.AttnDown != 39.3 || st.AttnUp != 19.9 {
		t.Fatalf("line quality %+v", st)
	}
	if st.Connected == nil || !*st.Connected || st.UptimeSecs != 88760 || st.Model != "F6640" {
		t.Fatalf("connection %+v", st)
	}
	if st.HasCounters || st.LineUpSecs != 0 {
		t.Fatalf("nothing is invented: %+v", st)
	}
	if f.logins.Load() != 1 || f.logouts.Load() != 1 {
		t.Fatalf("logins %d logouts %d", f.logins.Load(), f.logouts.Load())
	}
}

func TestZTENewReadsFiberGatewayWithoutDSL(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	f.dsl = false
	st, err := readZTE(t, f, "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if st.HasLineStats || st.HasCounters {
		t.Fatalf("a fiber gateway has no line figures: %+v", st)
	}
	if st.Connected == nil || !*st.Connected || st.UptimeSecs != 88760 || st.Model != "F6640" {
		t.Fatalf("connection %+v", st)
	}
}

func TestZTENewReadsDSLGatewayWhoseWANPageTimesOut(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	f.wan = false
	st, err := readZTE(t, f, "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasLineStats || st.Connected != nil || st.UptimeSecs != 0 {
		t.Fatalf("only the line is known: %+v", st)
	}
}

func TestZTENewNothingReadableIsUnsupported(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	f.dsl, f.wan = false, false
	if _, err := readZTE(t, f, "s3cret"); !errors.Is(err, ErrWebUnknown) {
		t.Fatalf("got %v", err)
	}
	if f.logouts.Load() != 1 {
		t.Fatalf("must still sign out: %d", f.logouts.Load())
	}
}

func TestZTENewWrongPasswordIsSentOnce(t *testing.T) {
	f := newFakeZTE(t, "right")
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	clock, host, pw := at(0), hostOfServer(srv), "wrong"
	p := &Poller{
		Discover: func(context.Context) (string, error) { return "", ErrNotFound },
		Web:      func() (WebReader, bool) { return WebReader{Host: host, Password: pw}, true },
		Save:     func(_ time.Time, _ Status) {},
		Now:      func() time.Time { return clock },
	}
	advance(p, &clock, 6)
	if f.logins.Load() != 1 {
		t.Fatalf("a rejected password must be sent once, not %d times", f.logins.Load())
	}
	if snap := p.Snapshot(); snap.WebError != "wrong_password" || snap.Driver != DriverZTENew || !snap.Experimental {
		t.Fatalf("snapshot %+v", snap)
	}
}

func TestZTENewSessionBusy(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	f.busyMsg = "Another user is already logged in"
	_, err := readZTE(t, f, "s3cret")
	if !errors.Is(err, ErrRouterBusy) || f.logins.Load() != 1 {
		t.Fatalf("err %v logins %d", err, f.logins.Load())
	}
}

func TestZTENewLockedRouterReceivesNoPassword(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	f.lockSecs = 120
	_, err := readZTE(t, f, "s3cret")
	if !errors.Is(err, ErrRouterBusy) || f.logins.Load() != 0 {
		t.Fatalf("err %v logins %d", err, f.logins.Load())
	}
}

func TestZTENewDetectionIsUnambiguous(t *testing.T) {
	zte := newFakeZTE(t, "x")
	hw := &fakeWeb{password: "x"}
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "<html><title>Router</title></html>")
	}))
	defer plain.Close()
	zsrv := httptest.NewServer(zte.handler())
	defer zsrv.Close()
	hsrv := httptest.NewServer(hw.handler(t))
	defer hsrv.Close()
	for _, c := range []struct {
		host string
		want string
	}{{hostOfServer(zsrv), DriverZTENew}, {hostOfServer(hsrv), DriverHuaweiHG}, {hostOfServer(plain), ""}} {
		p, err := NewProbe(c.host, 0)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Detect(context.Background(), p, Drivers(), false)
		got := ""
		if d != nil {
			got = d.Name()
		}
		if got != c.want || (c.want == "" && !errors.Is(err, ErrWebUnknown)) {
			t.Fatalf("%s: got %q err %v", c.host, got, err)
		}
	}
	if zte.logins.Load() != 0 {
		t.Fatal("detection must never send credentials")
	}
}

func TestZTENewIsNotAnonymous(t *testing.T) {
	if canReadAnonymously(zteNew{}) {
		t.Fatal("this driver needs a password")
	}
}

func TestZTEProberRecordsShapesOnly(t *testing.T) {
	f := newFakeZTE(t, "s3cret")
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	rep, err := CollectReport(context.Background(), deepOptions(srv, "s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeSignedIn || rep.Deep.Family != zteNewFamily || len(rep.Deep.Endpoints) != 3 {
		t.Fatalf("%+v", rep.Deep)
	}
	if answeredEndpoints(rep.Deep.Endpoints) != 3 {
		t.Fatalf("endpoints %+v", rep.Deep.Endpoints)
	}
	if f.logins.Load() != 1 || f.logouts.Load() != 1 {
		t.Fatalf("logins %d logouts %d", f.logins.Load(), f.logouts.Load())
	}
	text := marshal(t, BuildResult(rep))
	for _, s := range []string{"s3cret", zteSerial, zteWANIP, zteWANMAC, zteSessToken, zteLoginSeed, "203.0.113.1", "ZTEGF6640P2N10D_V"} {
		if strings.Contains(text, s) {
			t.Errorf("zte report leaks %q", s)
		}
	}
	for _, want := range []string{`"Downstream_current_rate"`, `"value": 40315`, `"OBJ_DSLINTERFACE_ID"`, `"ConnStatus"`, `"SerialNumber"`} {
		if !strings.Contains(text, want) {
			t.Errorf("zte report misses %s", want)
		}
	}
}

func TestZTEProberNeverRetriesARejectedPassword(t *testing.T) {
	f := newFakeZTE(t, "right")
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	rep, err := CollectReport(context.Background(), deepOptions(srv, "wrong"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeRejected || len(rep.Deep.Endpoints) != 0 || f.logins.Load() != 1 {
		t.Fatalf("%+v logins %d", rep.Deep, f.logins.Load())
	}
}
