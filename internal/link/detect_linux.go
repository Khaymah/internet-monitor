package link

import (
	"os"
	"path/filepath"
)

func Detect() Info {
	name, ok := DefaultInterface()
	if !ok {
		return Info{Type: Other}
	}
	info := Info{Type: Ethernet, Interface: name}
	if _, err := os.Stat(filepath.Join("/sys/class/net", name, "wireless")); err == nil {
		info.Type = Wifi
	} else if _, err := os.Stat(filepath.Join("/sys/class/net", name, "phy80211")); err == nil {
		info.Type = Wifi
	} else if _, err := os.Stat(filepath.Join("/sys/class/net", name, "device")); err != nil {
		info.Type = Other
	}
	if info.Type == Wifi {
		if data, err := os.ReadFile("/proc/net/wireless"); err == nil {
			if q, ok := ParseProcWireless(string(data), name); ok {
				info.Signal = q
			}
		}
	}
	return info
}
