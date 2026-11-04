//go:build !linux && !windows && !darwin

package link

func Detect() Info {
	name, ok := DefaultInterface()
	if !ok {
		return Info{Type: Other}
	}
	return Info{Type: GuessByName(name), Interface: name}
}
