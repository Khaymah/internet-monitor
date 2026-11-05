package server

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestPolicyAllowsTutorialVideoFromGitHubOnly(t *testing.T) {
	e := newTestEnv(t)
	policy := e.do(t, http.MethodGet, "/api/config", "", nil).Header.Get("Content-Security-Policy")
	var media string
	for _, d := range strings.Split(policy, ";") {
		if d = strings.TrimSpace(d); strings.HasPrefix(d, "media-src ") {
			media = d
		}
	}
	want := "media-src 'self' https://github.com https://objects.githubusercontent.com https://release-assets.githubusercontent.com"
	if media != want {
		t.Errorf("media-src = %q, want %q", media, want)
	}
	for _, d := range []string{"default-src 'self'", "script-src 'self'", "connect-src 'self'"} {
		if !strings.Contains(policy, d) {
			t.Errorf("policy lost %q: %s", d, policy)
		}
	}
}

func TestTutorialSubtitlesServedAsWebVTT(t *testing.T) {
	if _, err := fs.Stat(uiFS, "tutorial/en.vtt"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("web/dist/tutorial/en.vtt missing; build web/ before go test")
		}
		t.Skip("web/dist not built")
	}
	e := newTestEnv(t)
	for _, lang := range []string{"en", "ar", "fr", "tr", "ur"} {
		resp, err := http.Get(e.base + "/tutorial/" + lang + ".vtt")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/vtt; charset=utf-8" || !strings.HasPrefix(string(body), "WEBVTT") {
			t.Errorf("GET /tutorial/%s.vtt: %d %q", lang, resp.StatusCode, resp.Header.Get("Content-Type"))
		}
	}
}
