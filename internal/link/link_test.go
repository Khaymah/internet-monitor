package link

import (
	"testing"
	"time"
)

const netshOut = "\r\nThere is 1 interface on the system: \r\n\r\n    Name                   : Wi-Fi\r\n    Description            : Intel(R) Wi-Fi 6 AX201 160MHz\r\n    State                  : connected\r\n    SSID                   : WE_8F2A10\r\n    Radio type             : 802.11ac\r\n    Receive rate (Mbps)    : 400\r\n    Signal                 : 62% \r\n    Profile                : WE_8F2A10\r\n\r\n    Hosted network status  : Not available\r\n"

const netshTwo = `
    Name                   : Wi-Fi 2
    State                  : disconnected

    Name                   : Wi-Fi
    State                  : connected
    SSID                   : Home
    Signal                 : 91%
`

func TestParseNetsh(t *testing.T) {
	got := ParseNetsh(netshOut)
	if len(got) != 1 || got[0].Interface != "Wi-Fi" || got[0].SSID != "WE_8F2A10" || got[0].Signal != 62 || got[0].Type != Wifi {
		t.Fatalf("got %+v", got)
	}
	two := ParseNetsh(netshTwo)
	if len(two) != 1 || two[0].Interface != "Wi-Fi" || two[0].Signal != 91 {
		t.Fatalf("only connected interfaces count: %+v", two)
	}
}

func TestParseProcWireless(t *testing.T) {
	out := "Inter-| sta-|   Quality        |   Discarded packets\n face | tus | link level noise |  nwid  crypt\n wlan0: 0000   54.  -56.  -256        0      0\n"
	if q, ok := ParseProcWireless(out, "wlan0"); !ok || q != 77 {
		t.Fatalf("q=%d ok=%v, want 77", q, ok)
	}
	if _, ok := ParseProcWireless(out, "eth0"); ok {
		t.Fatal("eth0 is not wireless")
	}
}

func TestHardwarePortsAndNames(t *testing.T) {
	ports := ParseHardwarePorts("Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: x\n\nHardware Port: Thunderbolt Ethernet Slot 1\nDevice: en5\n")
	if TypeOfPort(ports["en0"]) != Wifi || TypeOfPort(ports["en5"]) != Ethernet {
		t.Fatalf("ports = %v", ports)
	}
	for name, want := range map[string]string{"Wi-Fi": Wifi, "wlp2s0": Wifi, "Ethernet 2": Ethernet, "eth0": Ethernet, "tun0": Other} {
		if got := GuessByName(name); got != want {
			t.Errorf("GuessByName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCacheRefreshes(t *testing.T) {
	calls := 0
	c := &Cache{Every: time.Hour, detect: func() Info { calls++; return Info{Type: Wifi} }}
	c.Get()
	c.Get()
	if calls != 1 {
		t.Fatalf("detect called %d times within the refresh window", calls)
	}
}
