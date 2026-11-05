package server

import (
	"bytes"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"rsc.io/qr"
)

func TestPhoneHiddenOutsideLAN(t *testing.T) {
	e := newTestEnv(t)
	resp := e.do(t, http.MethodGet, "/api/phone", "", nil)
	if got := decode[map[string]interface{}](t, resp.Body); got["available"] != false || got["address"] != "" {
		t.Errorf("loopback phone info = %v", got)
	}
	if r := e.do(t, http.MethodGet, "/api/phone/qr.png", "", nil); r.StatusCode != http.StatusNotFound {
		t.Errorf("loopback qr: status %d, want 404", r.StatusCode)
	}
	for _, p := range []string{"/api/phone", "/api/phone/qr.png"} {
		resp, err := http.Get(e.base + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without session: %d, want 401", p, resp.StatusCode)
		}
	}
}

func TestPhoneUsesTheAddressTheBrowserUsed(t *testing.T) {
	s := &Server{port: 8765, listen: "0.0.0.0", token: strings.Repeat("ab", 32)}
	if got := s.phoneBase("192.168.1.20:8765"); got != "http://192.168.1.20:8765" {
		t.Errorf("phoneBase(LAN IP) = %q", got)
	}
	for _, host := range []string{"127.0.0.1:8765", "raspberrypi.local:8765", "[fd00::5]:8765"} {
		got := s.phoneBase(host)
		if strings.Contains(got, "127.0.0.1") || strings.Contains(got, ".local") || strings.Contains(got, "fd00") {
			t.Errorf("phoneBase(%q) = %q, want a LAN IPv4 address or nothing", host, got)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/phone/qr.png", nil)
	req.Host = "192.168.1.20:8765"
	rec := httptest.NewRecorder()
	s.servePhoneQR(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("qr: status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil || img.Bounds().Dx() < 200 || img.Bounds().Dx() != img.Bounds().Dy() {
		t.Fatalf("qr png: %v %v", err, img)
	}

	rec = httptest.NewRecorder()
	s.servePhone(rec, req)
	if body := rec.Body.String(); strings.Contains(body, s.token) || !strings.Contains(body, `"address":"http://192.168.1.20:8765"`) {
		t.Errorf("phone info must name the address without the key: %s", body)
	}
}

func TestTerminalQRMatchesTheCode(t *testing.T) {
	text := "http://192.168.1.20:8765/?session=" + strings.Repeat("ab", 32)
	out, err := TerminalQR(text)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := qr.Encode(text, qr.L)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if want := (code.Size + 5) / 2; len(lines) != want {
		t.Fatalf("%d lines, want %d", len(lines), want)
	}
	for i, line := range lines {
		if n := utf8.RuneCountInString(line); n != code.Size+4 {
			t.Fatalf("line %d has %d cells, want %d", i, n, code.Size+4)
		}
	}
	finder := []rune(lines[1])
	if finder[1] != '█' || finder[2] != ' ' || finder[3] != '▄' {
		t.Errorf("quiet zone or finder pattern drawn wrong: %q", lines[1])
	}
}

func TestManifestAndIconsLoadWithoutSession(t *testing.T) {
	if _, err := fs.Stat(uiFS, "manifest.webmanifest"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("web/dist/manifest.webmanifest missing; build web/ before go test")
		}
		t.Skip("web/dist not built")
	}
	e := newTestEnv(t)
	for path, want := range map[string]string{
		"/manifest.webmanifest": "application/manifest+json",
		"/icon-192.png":         "image/png",
		"/icon-512.png":         "image/png",
	} {
		resp, err := http.Get(e.base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != want || len(body) == 0 {
			t.Errorf("GET %s without session: %d %q", path, resp.StatusCode, resp.Header.Get("Content-Type"))
		}
	}
}
