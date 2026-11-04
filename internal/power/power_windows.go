package power

import (
	"syscall"
	"time"
	"unsafe"
)

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
	getTickCount64       = kernel32.NewProc("GetTickCount64")
)

func OnBattery() (onBattery, known bool) {
	var st systemPowerStatus
	if r, _, _ := getSystemPowerStatus.Call(uintptr(unsafe.Pointer(&st))); r == 0 {
		return false, false
	}
	if st.BatteryFlag == 128 || st.BatteryFlag == 255 {
		return false, false
	}
	switch st.ACLineStatus {
	case 0:
		return true, true
	case 1:
		return false, true
	}
	return false, false
}

func BootTime() (time.Time, bool) {
	ms, _, _ := getTickCount64.Call()
	if ms == 0 {
		return time.Time{}, false
	}
	return time.Now().Add(-time.Duration(ms) * time.Millisecond).Truncate(time.Second), true
}
