package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"rsc.io/qr"
)

func (s *Server) lanBases() []string {
	var out []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok || ipn.IP.IsLoopback() || ipn.IP.To4() == nil || ipn.IP.IsLinkLocalUnicast() {
			continue
		}
		out = append(out, fmt.Sprintf("http://%s:%d", ipn.IP, s.port))
	}
	return out
}

func (s *Server) phoneBase(host string) string {
	if !s.LANMode() {
		return ""
	}
	if name, _, err := net.SplitHostPort(host); err == nil {
		if ip := net.ParseIP(name); ip != nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			return "http://" + strings.ToLower(host)
		}
	}
	if bases := s.lanBases(); len(bases) > 0 {
		return bases[0]
	}
	return ""
}

func (s *Server) phoneLink(base string) string {
	return fmt.Sprintf("%s/?%s=%s", base, sessionQuery, s.token)
}

func (s *Server) servePhone(w http.ResponseWriter, r *http.Request) {
	base := s.phoneBase(r.Host)
	writeJSON(w, http.StatusOK, map[string]interface{}{"available": base != "", "address": base})
}

func (s *Server) servePhoneQR(w http.ResponseWriter, r *http.Request) {
	base := s.phoneBase(r.Host)
	if base == "" {
		writeError(w, http.StatusNotFound, "not_lan", "the dashboard is not open to the network")
		return
	}
	code, err := qr.Encode(s.phoneLink(base), qr.M)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "qr_failed", "could not draw the code")
		return
	}
	code.Scale = 8
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(code.PNG())
}

func TerminalQR(text string) (string, error) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return "", err
	}
	const quiet = 2
	end := code.Size + quiet
	light := func(x, y int) bool { return y < end && !code.Black(x, y) }
	var b strings.Builder
	for y := -quiet; y < end; y += 2 {
		for x := -quiet; x < end; x++ {
			switch top, bottom := light(x, y), light(x, y+1); {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteString(" ")
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
