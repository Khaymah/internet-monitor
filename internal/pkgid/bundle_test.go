package pkgid

import "testing"

func TestInBundle(t *testing.T) {
	for exe, want := range map[string]bool{
		"/Applications/Internet Monitor.app/Contents/MacOS/internet-monitor":             true,
		"/Users/a/Downloads/Internet Monitor Window.app/Contents/MacOS/internet-monitor": true,
		"/Users/a/Downloads/internet-monitor-macos-arm64":                                false,
		"/Users/a/foo.app/Contents/Resources/internet-monitor":                           false,
		"/opt/internet-monitor/internet-monitor":                                         false,
		"":                                                                               false,
	} {
		if got := InBundle(exe); got != want {
			t.Errorf("InBundle(%q) = %v, want %v", exe, got, want)
		}
	}
}
