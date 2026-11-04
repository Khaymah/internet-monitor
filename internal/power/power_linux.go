package power

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func OnBattery() (onBattery, known bool) {
	dirs, _ := filepath.Glob("/sys/class/power_supply/*")
	hasBattery := false
	for _, d := range dirs {
		kind, _ := os.ReadFile(filepath.Join(d, "type"))
		if strings.TrimSpace(string(kind)) == "Battery" {
			hasBattery = true
		}
	}
	if !hasBattery {
		return false, false
	}
	for _, d := range dirs {
		kind, _ := os.ReadFile(filepath.Join(d, "type"))
		if strings.TrimSpace(string(kind)) != "Mains" {
			continue
		}
		online, err := os.ReadFile(filepath.Join(d, "online"))
		if err == nil {
			return strings.TrimSpace(string(online)) == "0", true
		}
	}
	return false, false
}

func BootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	return ParseProcStat(string(data))
}
