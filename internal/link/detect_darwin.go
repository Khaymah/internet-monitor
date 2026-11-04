package link

import "os/exec"

func Detect() Info {
	name, ok := DefaultInterface()
	if !ok {
		return Info{Type: Other}
	}
	if out, err := exec.Command("networksetup", "-listallhardwareports").Output(); err == nil {
		if port, ok := ParseHardwarePorts(string(out))[name]; ok {
			return Info{Type: TypeOfPort(port), Interface: name}
		}
	}
	return Info{Type: GuessByName(name), Interface: name}
}
