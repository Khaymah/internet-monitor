package app

import (
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	home := filepath.FromSlash("/Users/sam")
	support := filepath.Join(home, "Library", "Application Support", "Internet Monitor")
	for _, c := range []struct {
		exe, home, want string
	}{
		{"/Applications/Internet Monitor.app/Contents/MacOS/internet-monitor", home, support},
		{"/Users/sam/Downloads/Internet Monitor Window.app/Contents/MacOS/internet-monitor-gui", home, support},
		{"/Users/sam/Downloads/internet-monitor-macos-arm64", home, filepath.FromSlash("/Users/sam/Downloads")},
		{"/opt/im/internet-monitor", home, filepath.FromSlash("/opt/im")},
		{"/Applications/Internet Monitor.app/Contents/MacOS/internet-monitor", "", filepath.FromSlash("/Applications/Internet Monitor.app/Contents/MacOS")},
	} {
		if got := dataDir(filepath.FromSlash(c.exe), c.home); got != c.want {
			t.Errorf("dataDir(%q, %q) = %q, want %q", c.exe, c.home, got, c.want)
		}
	}
}
