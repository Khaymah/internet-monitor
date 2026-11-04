package power

import (
	"os/exec"
	"time"
)

func OnBattery() (onBattery, known bool) {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return false, false
	}
	return ParsePmset(string(out))
}

func BootTime() (time.Time, bool) {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return time.Time{}, false
	}
	return ParseKernBoottime(string(out))
}
