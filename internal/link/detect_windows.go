package link

import (
	"os/exec"
	"strings"
	"syscall"
)

func Detect() Info {
	name, ok := DefaultInterface()
	if !ok {
		return Info{Type: Other}
	}
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if out, err := cmd.Output(); err == nil {
		for _, w := range ParseNetsh(string(out)) {
			if strings.EqualFold(w.Interface, name) {
				return w
			}
		}
	}
	t := GuessByName(name)
	if t == Wifi {
		return Info{Type: Wifi, Interface: name}
	}
	return Info{Type: t, Interface: name}
}
