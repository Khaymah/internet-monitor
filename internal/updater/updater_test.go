package updater

import (
	"strings"
	"testing"
)

func TestParseChecksums(t *testing.T) {
	data := []byte(
		"aaa111  internet-monitor-windows.exe\n" +
			"bbb222 *internet-monitor-linux\n" +
			"ccc333  internet-monitor-macos-arm64\n")

	if got := parseChecksums(data, "internet-monitor-windows.exe"); got != "aaa111" {
		t.Errorf("windows hash = %q, want aaa111", got)
	}
	if got := parseChecksums(data, "internet-monitor-linux"); got != "bbb222" {
		t.Errorf("linux hash (binary marker) = %q, want bbb222", got)
	}
	if got := parseChecksums(data, "not-listed"); got != "" {
		t.Errorf("missing asset = %q, want empty", got)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.0", "v1.1.9", 1},
		{"1.2.0", "v1.2.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.2.3", "dev", 0},
		{"v1.2.3", "", 0},
		{"v1.10.0", "v1.9.0", 1},
		{"v2.0.0", "v2.0.0-rc.1", 1},
		{"v2.0.0-rc.1", "v2.0.0", -1},
		{"v2.0.0-rc.2", "v2.0.0-rc.1", 1},
		{"v2.0.0-rc.10", "v2.0.0-rc.9", 1},
		{"v2.0.0-rc.1", "v2.0.0-beta.3", 1},
		{"v2.0.0-rc", "v2.0.0-rc.1", -1},
		{"v2.0.0-rc.1", "v0.10.1", 1},
		{"v2.0.0+build.5", "v2.0.0", 0},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestTrustedURL(t *testing.T) {
	for _, u := range []string{
		"https://github.com/Khaymah/internet-monitor/releases/download/v1/x.exe",
		"https://objects.githubusercontent.com/abc",
	} {
		if err := trustedURL(u); err != nil {
			t.Errorf("%s rejected: %v", u, err)
		}
	}
	for _, u := range []string{
		"http://github.com/x",
		"https://evil.example/x.exe",
		"https://github.com.evil.example/x",
	} {
		if trustedURL(u) == nil {
			t.Errorf("%s accepted", u)
		}
	}
}

func TestPickAssetSkipsGUIForTray(t *testing.T) {
	names := []string{
		"internet-monitor-gui-windows.exe",
		"internet-monitor-gui-macos-arm64",
		"internet-monitor-gui-macos-intel",
		"internet-monitor-gui-linux",
		"internet-monitor-windows.exe",
		"internet-monitor-macos-arm64",
		"internet-monitor-macos-intel",
		"internet-monitor-linux",
	}
	assets := make([]ghAsset, 0, len(names))
	for _, n := range names {
		assets = append(assets, ghAsset{Name: n, BrowserDownloadURL: "https://github.com/x/y/releases/download/v1/" + n})
	}
	got := pickAsset(assets)
	if got == "" {
		t.Fatal("no asset picked")
	}
	if strings.Contains(got, "gui") {
		t.Errorf("tray build picked GUI asset %q", got)
	}
}

func TestPrereleaseChannel(t *testing.T) {
	for v, want := range map[string]bool{
		"v2.0.4-rc.1":       true,
		"2.0.4-rc.2":        true,
		"v2.0.4-dev.abc123": true,
		"v2.0.4":            false,
		"v2.0.2+build.5":    false,
		"dev":               false,
	} {
		if got := IsPrerelease(v); got != want {
			t.Errorf("IsPrerelease(%q) = %v, want %v", v, got, want)
		}
	}

	rels := []ghRelease{
		{TagName: "v2.0.2"},
		{TagName: "v2.0.4-rc.1", Prerelease: true},
		{TagName: "v2.0.4-rc.3", Prerelease: true, Draft: true},
		{TagName: "v2.0.4-rc.2", Prerelease: true},
	}
	got, ok := newestRelease(rels)
	if !ok || got.TagName != "v2.0.4-rc.2" {
		t.Fatalf("newest = %q, want v2.0.4-rc.2 (drafts are skipped)", got.TagName)
	}
	if compareVersions(got.TagName, "v2.0.4-rc.1") <= 0 {
		t.Fatal("rc.2 must be offered to an rc.1 install")
	}

	rels = append(rels, ghRelease{TagName: "v2.0.4"})
	if got, _ := newestRelease(rels); got.TagName != "v2.0.4" {
		t.Fatalf("newest = %q, want the final v2.0.4 over its rcs", got.TagName)
	}
	if _, ok := newestRelease(nil); ok {
		t.Fatal("no releases must report none")
	}
}
