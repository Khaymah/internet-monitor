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
	"testing"
)

const (
	leakMAC        = "AA:BB:CC:11:22:33"
	leakMACDash    = "aa-bb-cc-44-55-66"
	leakSerial     = "ZTEGC8A1B2C3D4E5"
	leakIMEI       = "356938035643809"
	leakIMSI       = "602011234567890"
	leakSSID       = "AliceHomeWiFi"
	leakWifi       = "Sup3rS3cretWifiKey"
	leakPublicIP   = "85.12.34.56"
	leakPPPoE      = "alice_pppoe"
	leakEmail      = "alice@example.com"
	leakPhone      = "+20 100 123 4567"
	leakPhone2     = "01001234567"
	leakHost       = "Alices-iPhone"
	leakHex        = "0123456789abcdef0123456789abcdef"
	leakB64        = "dGhpcy1pcy1hLXNlY3JldC10b2tlbi12YWx1ZQ=="
	leakJWT        = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"
	leakCookie     = "s3ssionVALUEabc123"
	leakCSRF       = "csrfTOKENvalue987"
	leakPassword   = "hunter2hunter2"
	leakFriendly   = "John Smith"
	leakUDN        = "uuid:12345678-aaaa-bbbb-cccc-1234567890ab"
	leakDeepSSID   = "DeepNetworkName"
	leakDeepHost   = "Bobs-Laptop"
	leakDeepBearer = "tok_9f8e7d6c5b4a39281706f5e4d3c2b1a0"
)

var allSecrets = []string{
	leakMAC, leakMACDash, leakSerial, leakIMEI, leakIMSI, leakSSID, leakWifi,
	leakPublicIP, leakPPPoE, leakEmail, leakPhone, leakPhone2, leakHost, leakHex,
	leakB64, leakJWT, leakCookie, leakCSRF, leakPassword, leakFriendly, leakUDN,
	leakDeepSSID, leakDeepHost, leakDeepBearer,
}

const fakeDesc = `<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0"><device>
<deviceType>urn:schemas-upnp-org:device:InternetGatewayDevice:1</deviceType>
<friendlyName>John Smith's home router</friendlyName>
<manufacturer>ZTE Corporation</manufacturer>
<modelName>ZXHN H298A</modelName>
<modelNumber>V9.1</modelNumber>
<softwareVersion>V9.0.10P1N11</softwareVersion>
<serialNumber>ZTEGC8A1B2C3D4E5</serialNumber>
<UDN>uuid:12345678-aaaa-bbbb-cccc-1234567890ab</UDN>
<serviceList>
<service><serviceType>urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1</serviceType><controlURL>/ctl/common</controlURL><SCPDURL>/scpd/common.xml</SCPDURL></service>
</serviceList>
<deviceList><device><deviceType>urn:schemas-upnp-org:device:WANDevice:1</deviceType>
<deviceList><device><deviceType>urn:schemas-upnp-org:device:WANConnectionDevice:1</deviceType>
<serviceList>
<service><serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType><controlURL>/ctl/ip</controlURL><SCPDURL>/scpd/ip.xml</SCPDURL></service>
</serviceList></device></deviceList></device></deviceList>
</device></root>`

const fakeSCPD = `<?xml version="1.0"?><scpd xmlns="urn:schemas-upnp-org:service-1-0"><actionList>
<action><name>GetStatusInfo</name></action><action><name>GetExternalIPAddress</name></action>
<action><name>GetCommonLinkProperties</name></action></actionList></scpd>`

const fakeTR64 = `<?xml version="1.0"?><root xmlns="urn:dslforum-org:device-1-0"><device>
<deviceType>urn:dslforum-org:device:InternetGatewayDevice:1</deviceType>
<friendlyName>ZXHN H298A</friendlyName><manufacturer>ZTE Corporation</manufacturer>
<modelName>ZXHN H298A</modelName><serialNumber>ZTEGC8A1B2C3D4E5</serialNumber>
<serviceList><service><serviceType>urn:dslforum-org:service:DeviceInfo:1</serviceType></service>
<service><serviceType>urn:dslforum-org:service:WANCommonInterfaceConfig:1</serviceType></service></serviceList>
</device></root>`

const fakeLoginPage = `<!DOCTYPE html><html><head><title> ZXHN H298A  </title>
<meta name="x-csrf" content="` + leakCSRF + `"><meta name="viewport" content="width=device-width">
</head><body>
<!-- debug: SN: ` + leakSerial + ` mac=` + leakMAC + ` -->
<form action="/login.cgi?sid=` + leakCookie + `" method="post">
<input type="text" name="Frm_Username" value="` + leakPPPoE + `">
<input type="password" name="Frm_Password" value="` + leakPassword + `">
<input type="hidden" id="token_field" value="` + leakHex + `">
</form>
<script src="/js/login.js"></script>
<script src="https://cdn.example.org/lib.js"></script>
<script>var wifi = "ssid=` + leakSSID + `"; var u = "/?_type=loginData&_tag=login_entry";</script>
<p>Contact ` + leakEmail + ` or ` + leakPhone + `. Admin: ` + leakJWT + `</p>
</body></html>`

const fakeLoginJS = `
var a = "/api/system/user_login"; var b = '/cgi?2'; var c = "/asp/GetRandCount.asp";
var d = "/login.cgi"; var e = "/static/logo.png"; var f = "/?_type=loginData&_tag=login_entry";
var secret = "wifikey=` + leakWifi + `"; var host = "hostname=` + leakHost + `";
var tok = "` + leakB64 + `"; var ip = "` + leakPublicIP + `"; var brand = "ZTE";`

func soapReply(action string, fields map[string]string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:` + action + `Response xmlns:u="x">`)
	for k, v := range fields {
		fmt.Fprintf(&sb, "<%s>%s</%s>", k, v, k)
	}
	sb.WriteString(`</u:` + action + `Response></s:Body></s:Envelope>`)
	return sb.String()
}

func fakeRouter(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/desc.xml", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, fakeDesc) })
	mux.HandleFunc("/scpd/", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, fakeSCPD) })
	mux.HandleFunc("/tr64desc.xml", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, fakeTR64) })
	replies := map[string]map[string]string{
		"GetCommonLinkProperties": {
			"NewWANAccessType": "DSL", "NewPhysicalLinkStatus": "Up",
			"NewLayer1DownstreamMaxBitRate": "40957000", "NewLayer1UpstreamMaxBitRate": "5120000",
			"NewMACAddress": leakMAC, "NewSSID": leakSSID,
		},
		"GetTotalBytesReceived": {"NewTotalBytesReceived": "18446744073709551615"},
		"GetTotalBytesSent":     {"NewTotalBytesSent": "3855775661"},
		"GetStatusInfo": {
			"NewConnectionStatus": "Connected", "NewUptime": "750582", "NewLastConnectionError": "ERROR_NONE",
			"NewExternalIPAddress": leakPublicIP, "NewPPPoEUser": leakPPPoE, "NewIMEI": leakIMEI, "NewIMSI": leakIMSI,
			"NewNote": leakPhone2, "NewClient": leakHost,
		},
	}
	ctl := func(w http.ResponseWriter, r *http.Request) {
		action := strings.Trim(r.Header.Get("SOAPAction"), `"`)
		action = action[strings.LastIndex(action, "#")+1:]
		f, ok := replies[action]
		if !ok {
			http.Error(w, "no", http.StatusInternalServerError)
			return
		}
		io.WriteString(w, soapReply(action, f))
	}
	mux.HandleFunc("/ctl/common", ctl)
	mux.HandleFunc("/ctl/ip", ctl)
	mux.HandleFunc("/js/login.js", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, fakeLoginJS) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Server", "Boa/0.94.14rc21")
		http.SetCookie(w, &http.Cookie{Name: "SessionID", Value: leakCookie, Path: "/"})
		io.WriteString(w, fakeLoginPage)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func baseOptions(srv *httptest.Server) ReportOptions {
	return ReportOptions{
		Version:   "v9.9.9",
		Host:      "127.0.0.1",
		Discover:  func(context.Context) (string, error) { return srv.URL + "/desc.xml", nil },
		WebBases:  []string{srv.URL},
		TR064Base: srv.URL,
	}
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestReportCollectsEverythingAndLeaksNothing(t *testing.T) {
	srv := fakeRouter(t)
	rep, err := CollectReport(context.Background(), baseOptions(srv))
	if err != nil {
		t.Fatal(err)
	}
	res := BuildResult(rep)
	text := marshal(t, res)
	for _, s := range allSecrets {
		if strings.Contains(text, s) {
			t.Errorf("report leaks %q", s)
		}
	}
	if !rep.UPnP.Answered || rep.UPnP.Device == nil || rep.UPnP.Device.Manufacturer != "ZTE Corporation" ||
		rep.UPnP.Device.ModelName != "ZXHN H298A" || rep.UPnP.Device.SoftwareVersion != "V9.0.10P1N11" {
		t.Fatalf("upnp device %+v", rep.UPnP.Device)
	}
	if rep.UPnP.Device.FriendlyName != "" {
		t.Fatalf("friendly name that is not the model must be dropped: %q", rep.UPnP.Device.FriendlyName)
	}
	if len(rep.UPnP.Services) != 2 || len(rep.UPnP.Services[0].Actions) != 3 {
		t.Fatalf("services %+v", rep.UPnP.Services)
	}
	if len(rep.UPnP.Actions) != 5 || answeredCount(rep.UPnP.Actions) != 4 {
		t.Fatalf("status actions %+v", rep.UPnP.Actions)
	}
	if !rep.TR064.Found || rep.TR064.Device.ModelName != "ZXHN H298A" || len(rep.TR064.Services) != 2 {
		t.Fatalf("tr064 %+v", rep.TR064)
	}
	if rep.Deep.Outcome != outcomeNotRequested || len(rep.Web) != 1 {
		t.Fatalf("deep %+v web %d", rep.Deep, len(rep.Web))
	}
	fp := rep.Web[0]
	if !fp.Answered || fp.Status != 200 || fp.Server != "Boa/0.94.14rc21" || fp.Title != "ZXHN H298A" {
		t.Fatalf("fingerprint %+v", fp)
	}
	wantPaths := []string{"/api/system/user_login", "/cgi?2", "/asp/GetRandCount.asp", "/login.cgi", "/?_type=loginData&_tag=login_entry"}
	for _, p := range wantPaths {
		if !contains(fp.Paths, p) {
			t.Errorf("missing path %q in %v", p, fp.Paths)
		}
	}
	if contains(fp.Paths, "/static/logo.png") {
		t.Errorf("static asset listed: %v", fp.Paths)
	}
	if !contains(fp.Markers, "ZTE") || !contains(fp.Markers, "ZXHN") {
		t.Errorf("markers %v", fp.Markers)
	}
	if !contains(fp.Scripts, "/js/login.js") || !contains(fp.Scripts, "external:cdn.example.org") {
		t.Errorf("scripts %v", fp.Scripts)
	}
	if len(fp.Forms) != 1 || !contains(fp.Forms[0].Inputs, "Frm_Username (text)") || !contains(fp.Forms[0].Inputs, "Frm_Password (password)") {
		t.Errorf("forms %+v", fp.Forms)
	}
	if !contains(fp.CookieNames, "SessionID") || !contains(fp.MetaNames, "x-csrf") {
		t.Errorf("cookies %v meta %v", fp.CookieNames, fp.MetaNames)
	}
	if res.Filename != "router-report-zxhn-h298a.json" {
		t.Errorf("filename %q", res.Filename)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func fieldLeaf(t *testing.T, rep Report, action, field string) map[string]any {
	t.Helper()
	for _, a := range rep.UPnP.Actions {
		if a.Action == action {
			leaf, ok := a.Fields[field].(map[string]any)
			if !ok {
				t.Fatalf("%s.%s missing in %v", action, field, a.Fields)
			}
			return leaf
		}
	}
	t.Fatalf("action %s missing", action)
	return nil
}

func TestReportUPnPFieldShapes(t *testing.T) {
	srv := fakeRouter(t)
	rep, err := CollectReport(context.Background(), baseOptions(srv))
	if err != nil {
		t.Fatal(err)
	}
	if l := fieldLeaf(t, rep, "GetStatusInfo", "NewUptime"); l["type"] != "int" || fmt.Sprint(l["value"]) != "750582" {
		t.Errorf("uptime %v", l)
	}
	if l := fieldLeaf(t, rep, "GetStatusInfo", "NewConnectionStatus"); l["type"] != "enum" || l["value"] != "Connected" {
		t.Errorf("status %v", l)
	}
	if l := fieldLeaf(t, rep, "GetStatusInfo", "NewExternalIPAddress"); l["type"] != "ip" || l["scope"] != "public" {
		t.Errorf("public ip %v", l)
	}
	for _, f := range []string{"NewPPPoEUser", "NewIMEI", "NewIMSI"} {
		if l := fieldLeaf(t, rep, "GetStatusInfo", f); l["type"] == "int" || l["type"] == "enum" || l["value"] != nil {
			t.Errorf("%s kept a value: %v", f, l)
		}
	}
	if l := fieldLeaf(t, rep, "GetStatusInfo", "NewNote"); l["value"] != nil {
		t.Errorf("phone-like note kept: %v", l)
	}
	if l := fieldLeaf(t, rep, "GetCommonLinkProperties", "NewLayer1DownstreamMaxBitRate"); fmt.Sprint(l["value"]) != "40957000" {
		t.Errorf("long field names must survive: %v", l)
	}
	if l := fieldLeaf(t, rep, "GetCommonLinkProperties", "NewMACAddress"); l["type"] != "redacted" {
		t.Errorf("mac %v", l)
	}
	if l := fieldLeaf(t, rep, "GetTotalBytesReceived", "NewTotalBytesReceived"); fmt.Sprint(l["value"]) != "18446744073709551615" {
		t.Errorf("big counter lost precision: %v", l)
	}
}

func TestReportDoesNotContactPublicAddresses(t *testing.T) {
	_, err := CollectReport(context.Background(), ReportOptions{Host: "8.8.8.8"})
	if !errors.Is(err, ErrReportNotPrivate) {
		t.Fatalf("got %v", err)
	}
	_, err = CollectReport(context.Background(), ReportOptions{Host: "router.example.com"})
	if !errors.Is(err, ErrReportNotPrivate) {
		t.Fatalf("hostnames must be refused: %v", err)
	}
}

func TestReportRunsOneAtATime(t *testing.T) {
	reportGate.Lock()
	defer reportGate.Unlock()
	if _, err := CollectReport(context.Background(), ReportOptions{Host: "127.0.0.1"}); !errors.Is(err, ErrReportBusy) {
		t.Fatalf("got %v", err)
	}
}

func TestReportWithNothingAnswering(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	base := dead.URL
	dead.Close()
	rep, err := CollectReport(context.Background(), ReportOptions{
		Host:      "127.0.0.1",
		Discover:  func(context.Context) (string, error) { return "", errors.New("no answer") },
		WebBases:  []string{base},
		TR064Base: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.UPnP.Answered || rep.TR064.Found || rep.Web[0].Answered {
		t.Fatalf("%+v", rep)
	}
	res := BuildResult(rep)
	if res.Filename != "router-report-unknown.json" || !strings.Contains(res.IssueURL, "unknown+router") {
		t.Fatalf("%q %q", res.Filename, res.IssueURL)
	}
}

func TestIssueURLStaysShortAndPrefilled(t *testing.T) {
	srv := fakeRouter(t)
	rep, err := CollectReport(context.Background(), baseOptions(srv))
	if err != nil {
		t.Fatal(err)
	}
	rep.Web[0].Title = strings.Repeat("title ", 2000)
	res := BuildResult(rep)
	if len(res.IssueURL) > maxIssueURL {
		t.Fatalf("url is %d bytes", len(res.IssueURL))
	}
	for _, want := range []string{IssueNewURL + "?", "labels=router-support", "title=Router+support%3A+ZTE+Corporation+ZXHN+H298A", "router-report-zxhn-h298a.json"} {
		if !strings.Contains(res.IssueURL, want) {
			t.Errorf("url misses %q: %s", want, res.IssueURL)
		}
	}
	if strings.Contains(res.IssueURL, "%22schema%22") {
		t.Error("the JSON must not be put in the URL")
	}
}
