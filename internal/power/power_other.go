//go:build !linux && !windows && !darwin

package power

import "time"

func OnBattery() (onBattery, known bool) { return false, false }

func BootTime() (time.Time, bool) { return time.Time{}, false }
