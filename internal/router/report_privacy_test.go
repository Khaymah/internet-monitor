package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestScrubStringMasksEveryKindOfSecret(t *testing.T) {
	cases := map[string]string{
		"mac colon":      "client " + leakMAC + " joined",
		"mac dash":       "client " + leakMACDash + " joined",
		"mac dots":       "client aabb.ccdd.eeff joined",
		"mac bare":       "id 001122334455 here",
		"imei":           "imei " + leakIMEI,
		"imsi bare":      "sim " + leakIMSI,
		"serial label":   "SN: " + leakSerial,
		"serial json":    `{"SerialNumber":"` + leakSerial + `"}`,
		"ssid label":     "ssid=" + leakSSID,
		"wifi key":       "wpa_psk: " + leakWifi,
		"public ipv4":    "wan " + leakPublicIP,
		"public ipv6":    "wan 2001:db8:85a3::8a2e:370:7334",
		"pppoe user":     "pppoe_user=" + leakPPPoE,
		"email":          "mail " + leakEmail,
		"phone intl":     "call " + leakPhone,
		"phone local":    "call " + leakPhone2,
		"host name":      "hostname=" + leakHost,
		"long hex":       "key " + leakHex,
		"base64":         "blob " + leakB64,
		"jwt":            "bearer " + leakJWT,
		"cookie":         "cookie: " + leakCookie,
		"csrf":           "csrf_token=" + leakCSRF,
		"password":       `"password":"` + leakPassword + `"`,
		"session in url": "/login.cgi?sid=" + leakCookie,
	}
	leakOf := map[string]string{
		"mac dots": "aabb.ccdd.eeff", "mac bare": "001122334455", "public ipv6": "2001:db8:85a3::8a2e:370:7334",
		"imsi bare": leakIMSI, "serial json": leakSerial, "serial label": leakSerial, "ssid label": leakSSID,
		"wifi key": leakWifi, "public ipv4": leakPublicIP, "pppoe user": leakPPPoE, "email": leakEmail,
		"phone intl": leakPhone, "phone local": leakPhone2, "host name": leakHost, "long hex": leakHex,
		"base64": leakB64, "jwt": leakJWT, "cookie": leakCookie, "csrf": leakCSRF, "password": leakPassword,
		"session in url": leakCookie, "mac colon": leakMAC, "mac dash": leakMACDash, "imei": leakIMEI,
	}
	for name, in := range cases {
		out := scrubString(in, false)
		if secret := leakOf[name]; strings.Contains(out, secret) {
			t.Errorf("%s: %q survived in %q", name, secret, out)
		}
	}
}

func TestScrubStringKeepsUsefulText(t *testing.T) {
	keep := []string{
		"/api/system/user_login", "/api/ntwk/wan_st", "/?_type=loginData&_tag=login_entry", "/cgi?2",
		"/asp/GetRandCount.asp", "ZXHN H298A", "Boa/0.94.14rc21", "lighttpd/1.4.35", "192.168.1.1",
		"10.0.0.138", "127.0.0.1", "fe80::1", "2026-11-06", "2026-11-06T12:00:00Z", "HG630 V2",
		"urn:schemas-upnp-org:service:WANIPConnection:1", "ERROR_ISP_DISCONNECT", "/cgi-bin/luci/admin/network/wireless",
	}
	for _, s := range keep {
		if got := scrubString(s, false); got != s {
			t.Errorf("%q was changed to %q", s, got)
		}
	}
	if got := scrubString("V2.0.3.1", true); got != "V2.0.3.1" {
		t.Errorf("dotted version changed: %q", got)
	}
	if got := scrubString("85.12.34.56", true); got != "85.12.34.56" {
		t.Errorf("version keys are exempt from IP masking: %q", got)
	}
}

func TestSensitiveKey(t *testing.T) {
	for _, k := range []string{
		"MACAddress", "mac", "SerialNumber", "SN", "IMEI", "imsi", "SSID", "NewSSID", "WifiSSID", "WPAKey", "PreSharedKey",
		"Password", "NewPPPoEUser", "UserName", "Token", "csrf_token", "SessionID", "Cookie", "HostName", "DeviceName",
		"Email", "PhoneNumber", "ICCID", "Passphrase", "UDN", "ownerName",
	} {
		if !sensitiveKey(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	for _, k := range []string{"DownCurrRate", "Status", "DownMargin", "receivebytes", "NewUptime", "NewConnectionStatus", "ShowtimeStart", "Modulation"} {
		if sensitiveKey(k) {
			t.Errorf("%q should not be sensitive", k)
		}
	}
}

func TestShapeKeepsNumbersAndDropsValues(t *testing.T) {
	body := `{"DownCurrRate":40957,"DownMargin":6.5,"Status":"Up","Online":true,"SerialNumber":"` + leakSerial + `",
	"Number":4071234567,"WifiSSID":"` + leakDeepSSID + `","Alias":"` + leakDeepHost + `","Token":"` + leakDeepBearer + `",
	"Public":"` + leakPublicIP + `","Lan":"192.168.1.20","Caller":"` + leakPhone2 + `","Imei":"` + leakIMEI + `",
	"` + leakMAC + `":{"x":1},
	"Clients":[{"HostName":"` + leakDeepHost + `","MAC":"` + leakMAC + `","Rx":12}],"Nothing":null}`
	shape, ok := ShapeJSON([]byte(body))
	if !ok {
		t.Fatal("not json")
	}
	text := marshal(t, shape)
	for _, s := range []string{leakSerial, leakDeepSSID, leakDeepHost, leakDeepBearer, leakPublicIP, leakPhone2, leakIMEI, leakMAC} {
		if strings.Contains(text, s) {
			t.Errorf("shape leaks %q:\n%s", s, text)
		}
	}
	for _, want := range []string{`"value": 40957`, `"value": 6.5`, `"value": true`, `"scope": "private"`, `"scope": "public"`, `"length": 2`, `u003cmac`} {
		if !strings.Contains(text, want) {
			t.Errorf("shape misses %s:\n%s", want, text)
		}
	}
}

func TestShapeIsBounded(t *testing.T) {
	obj := map[string]any{}
	for i := range maxShapeKeys * 3 {
		obj[fmt.Sprintf("k%03d", i)] = i
	}
	shape := shapeOf("", obj, 0, false).(map[string]any)
	if len(shape) != maxShapeKeys+1 || shape["_truncated"] != true {
		t.Fatalf("%d keys", len(shape))
	}
	deep := any(1)
	for range maxShapeDepth * 2 {
		deep = map[string]any{"a": deep}
	}
	if !strings.Contains(marshal(t, shapeOf("", deep, 0, false)), `"truncated": true`) {
		t.Fatal("depth not capped")
	}
}

func TestSealCleansEveryStringFieldOfTheReport(t *testing.T) {
	dirty := "x " + leakMAC + " " + leakEmail + " " + leakPublicIP + " ssid=" + leakSSID + " " + leakHex
	rep := Report{
		CreatedAt: "2026-11-06T12:00:00Z",
		App:       ReportApp{Version: dirty},
		Router:    ReportTarget{Address: "192.168.1.1", Private: true},
		UPnP: UPnPReport{
			Error: dirty,
			Device: &DeviceInfo{
				Manufacturer: dirty, ModelName: dirty, ModelNumber: dirty, ModelDescription: dirty,
				FriendlyName: dirty, SoftwareVersion: "5.0.1.2", EmbeddedDevices: []string{dirty},
			},
			Services: []ServiceInfo{{Type: dirty, Actions: []string{dirty}, ActionsError: dirty}},
			Actions: []ActionProbe{{
				Service: dirty, Action: dirty, Error: dirty,
				Fields: map[string]any{dirty: map[string]any{"type": "string", dirty: dirty}},
			}},
		},
		TR064: TR064Report{Error: dirty, Device: &DeviceInfo{ModelName: dirty}, Services: []string{dirty}},
		Web: []WebFingerprint{{
			Scheme: dirty, Port: dirty, Error: dirty, RedirectedTo: dirty, Server: dirty, AuthRealm: dirty, Title: dirty,
			CookieNames: []string{dirty}, MetaNames: []string{dirty}, Scripts: []string{dirty}, Frames: []string{dirty},
			Paths: []string{dirty}, Markers: []string{dirty},
			Forms: []FormInfo{{Action: dirty, Method: dirty, Inputs: []string{dirty}}},
		}},
		Deep: DeepReport{Outcome: dirty, Family: dirty, Endpoints: []EndpointShape{{Path: dirty, Error: dirty, Shape: map[string]any{dirty: dirty}}}},
	}
	sealed, err := Seal(rep)
	if err != nil {
		t.Fatal(err)
	}
	text := marshal(t, BuildResult(sealed))
	for _, s := range []string{leakMAC, leakEmail, leakPublicIP, leakSSID, leakHex} {
		if strings.Contains(text, s) {
			t.Errorf("sealed report leaks %q", s)
		}
	}
	if sealed.UPnP.Device.SoftwareVersion != "5.0.1.2" || sealed.Router.Address != "192.168.1.1" || sealed.CreatedAt != rep.CreatedAt {
		t.Fatalf("harmless values were changed: %+v", sealed)
	}
}

type huaweiLike struct {
	logins  atomic.Int32
	logouts atomic.Int32
	pass    string
}

func (h *huaweiLike) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, `<html><head><title>HG630 V2</title><meta name="csrf_param" content="P1"/><meta name="csrf_token" content="T1"/></head></html>`)
	})
	mux.HandleFunc("/api/system/user_login", func(w http.ResponseWriter, r *http.Request) {
		h.logins.Add(1)
		var body struct {
			Csrf map[string]string `json:"csrf"`
			Data map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Data["Password"] == loginHash(body.Data["UserName"], h.pass, body.Csrf["csrf_param"], body.Csrf["csrf_token"]) {
			http.SetCookie(w, &http.Cookie{Name: "SessionID_R3", Value: "ok", Path: "/"})
			io.WriteString(w, `while(1); /*{"errorCategory":"ok"}*/`)
			return
		}
		io.WriteString(w, `while(1); /*{"errorCategory":"user_pass_err"}*/`)
	})
	mux.HandleFunc("/api/system/user_logout", func(w http.ResponseWriter, _ *http.Request) {
		h.logouts.Add(1)
		io.WriteString(w, `while(1); /*{"errorCategory":"ok"}*/`)
	})
	authed := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if c, err := r.Cookie("SessionID_R3"); err != nil || c.Value != "ok" {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			io.WriteString(w, body)
		}
	}
	mux.HandleFunc("/api/ntwk/dslinfo", authed(`while(1); /*{"Status":"Up","DownCurrRate":40957,"UpCurrRate":5120,"DownMargin":6.5}*/`))
	mux.HandleFunc("/api/ntwk/wan_st", authed(`[{"receivebytes":3103873845,"sendbytes":1,"IPv4Address":"`+leakPublicIP+`","MACAddress":"`+leakMAC+`","PPPoEUser":"`+leakPPPoE+`"}]`))
	mux.HandleFunc("/api/system/deviceinfo", authed(`{"SerialNumber":"`+leakSerial+`","DeviceName":"HG630 V2","UpTime":12345,"WifiSSID":"`+leakDeepSSID+`","Clients":[{"HostName":"`+leakDeepHost+`","IP":"192.168.1.20","MAC":"`+leakMAC+`","Token":"`+leakDeepBearer+`"}]}`))
	return mux
}

func deepOptions(srv *httptest.Server, password string) ReportOptions {
	return ReportOptions{
		Version:   "v9.9.9",
		Host:      strings.TrimPrefix(srv.URL, "http://"),
		Deep:      true,
		Password:  password,
		Discover:  func(context.Context) (string, error) { return "", errors.New("none") },
		WebBases:  []string{srv.URL},
		TR064Base: srv.URL,
	}
}

func TestDeepProbeRecordsShapesOnly(t *testing.T) {
	h := &huaweiLike{pass: "s3cret"}
	srv := httptest.NewServer(h.handler(t))
	defer srv.Close()
	rep, err := CollectReport(context.Background(), deepOptions(srv, "s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeSignedIn || rep.Deep.Family != huaweiFamily || len(rep.Deep.Endpoints) != 4 {
		t.Fatalf("%+v", rep.Deep)
	}
	if answeredEndpoints(rep.Deep.Endpoints) != 3 {
		t.Fatalf("endpoints %+v", rep.Deep.Endpoints)
	}
	if h.logins.Load() != 1 || h.logouts.Load() != 1 {
		t.Fatalf("logins %d logouts %d", h.logins.Load(), h.logouts.Load())
	}
	text := marshal(t, BuildResult(rep))
	for _, s := range []string{"s3cret", leakSerial, leakDeepSSID, leakDeepHost, leakDeepBearer, leakPublicIP, leakPPPoE, leakMAC} {
		if strings.Contains(text, s) {
			t.Errorf("deep report leaks %q", s)
		}
	}
	for _, want := range []string{`"DownCurrRate"`, `"value": 40957`, `"SerialNumber"`, `"Clients"`, `"receivebytes"`, `"value": 3103873845`} {
		if !strings.Contains(text, want) {
			t.Errorf("deep report misses %s", want)
		}
	}
}

func TestDeepProbeNeverRetriesARejectedPassword(t *testing.T) {
	h := &huaweiLike{pass: "right"}
	srv := httptest.NewServer(h.handler(t))
	defer srv.Close()
	rep, err := CollectReport(context.Background(), deepOptions(srv, "wrong"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeRejected || len(rep.Deep.Endpoints) != 0 || h.logins.Load() != 1 {
		t.Fatalf("%+v logins %d", rep.Deep, h.logins.Load())
	}
	opts := deepOptions(srv, "wrong")
	opts.PasswordRejected = true
	rep, err = CollectReport(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeEarlier || h.logins.Load() != 1 {
		t.Fatalf("%+v logins %d", rep.Deep, h.logins.Load())
	}
}

func TestDeepProbeNeedsOptInAndPassword(t *testing.T) {
	h := &huaweiLike{pass: "s3cret"}
	srv := httptest.NewServer(h.handler(t))
	defer srv.Close()
	opts := deepOptions(srv, "s3cret")
	opts.Deep = false
	rep, _ := CollectReport(context.Background(), opts)
	if rep.Deep.Outcome != outcomeNotRequested || rep.Deep.Requested {
		t.Fatalf("%+v", rep.Deep)
	}
	rep, _ = CollectReport(context.Background(), deepOptions(srv, ""))
	if rep.Deep.Outcome != outcomeNoCreds {
		t.Fatalf("%+v", rep.Deep)
	}
	if h.logins.Load() != 0 {
		t.Fatalf("signed in %d times without consent or password", h.logins.Load())
	}
}

func TestDeepProbeSkipsUnknownFamilies(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		io.WriteString(w, `<html><title>Other</title></html>`)
	}))
	defer srv.Close()
	rep, err := CollectReport(context.Background(), deepOptions(srv, "pw"))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeNoDriver {
		t.Fatalf("%+v", rep.Deep)
	}
}

type fakeProber struct{ called atomic.Int32 }

func (*fakeProber) Family() string { return "fake-family" }

func (*fakeProber) Matches(web []WebFingerprint) bool {
	return len(web) > 0 && web[0].Title == "Other"
}

func (f *fakeProber) Probe(_ context.Context, t DeepTarget) ([]EndpointShape, error) {
	f.called.Add(1)
	if t.User != defaultLoginUser || t.Password != "pw" {
		return nil, errors.New("wrong target")
	}
	return []EndpointShape{{Path: "/x", Answered: true, Shape: map[string]any{"a": 1}}}, nil
}

func TestDeepProbersArePluggable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `<html><title>Other</title></html>`)
	}))
	defer srv.Close()
	p := &fakeProber{}
	opts := deepOptions(srv, "pw")
	opts.Probers = []DeepProber{p}
	rep, err := CollectReport(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deep.Outcome != outcomeSignedIn || rep.Deep.Family != "fake-family" || p.called.Load() != 1 {
		t.Fatalf("%+v", rep.Deep)
	}
}

func TestPollerBlocksTheProbeAfterARejectedPassword(t *testing.T) {
	reader := WebReader{Host: "192.168.1.1", User: "admin", Password: "pw"}
	p := &Poller{Web: func() (WebReader, bool) { return reader, true }}
	opts, got := p.reportOptions("v1", true)
	if opts.PasswordRejected || got.Host != "192.168.1.1" {
		t.Fatalf("%+v", opts)
	}
	p.noteReport(got, Report{Deep: DeepReport{Outcome: outcomeRejected}})
	if opts, _ = p.reportOptions("v1", true); !opts.PasswordRejected {
		t.Fatal("a rejected password must block further attempts")
	}
	p.mu.Lock()
	blocked := p.webBlocked == webKey(reader)
	p.mu.Unlock()
	if !blocked {
		t.Fatal("the poller must stop using the rejected password too")
	}
	reader.Password = "changed"
	if opts, _ = p.reportOptions("v1", true); opts.PasswordRejected {
		t.Fatal("a new password deserves a new attempt")
	}
}

func TestPollerUsesTheGatewayAddressWhenNoneIsConfigured(t *testing.T) {
	p := &Poller{Web: func() (WebReader, bool) { return WebReader{}, false }}
	opts, _ := p.reportOptions("v1", false)
	if opts.Host != DefaultHost || opts.Deep || opts.Password != "" {
		t.Fatalf("%+v", opts)
	}
}
