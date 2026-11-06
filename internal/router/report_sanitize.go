package router

import (
	"bytes"
	"encoding/json"
	"math"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	maskRedacted    = "<redacted>"
	maskMAC         = "<mac>"
	maskEmail       = "<email>"
	maskPublicIP    = "<public-ip>"
	maskPublicIPv6  = "<public-ipv6>"
	maskNumber      = "<number>"
	maskHex         = "<hex>"
	maskSecret      = "<secret>"
	maskPhone       = "<phone>"
	minSecretBits   = 3.5
	minPlausible    = 0.8
	minWordLen      = 3
	minSecretLen    = 24
	minPhoneDigits  = 8
	maxShapeKeys    = 40
	maxShapeDepth   = 5
	maxEnumLen      = 32
	maxCountedDigit = 12
)

var (
	reLabeled = regexp.MustCompile(`(?i)\b([\w-]*(?:serial|imei|imsi|iccid|ssid|passphrase|passw(?:or)?d|pwd|psk|token|session|sid|cookie|csrf|user|login|secret|api_?key|hostname|host_name|devicename|device_name|clientname|computername)[\w-]*)(["']?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s"',;&<>]+)`)
	reSN      = regexp.MustCompile(`(?i)\b(s/n|sn)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s"',;&<>]+)`)
	reEmail   = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	reMAC     = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{2}[:-]){5}[0-9a-f]{2}\b`)
	reMACDots = regexp.MustCompile(`(?i)\b[0-9a-f]{4}\.[0-9a-f]{4}\.[0-9a-f]{4}\b`)
	reIPv4    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	reIPv6    = regexp.MustCompile(`(?i)[0-9a-f]{0,4}(?::[0-9a-f]{0,4}){2,7}(?:%[0-9a-z]+)?`)
	reDigits  = regexp.MustCompile(`\b\d{13,}\b`)
	reHex     = regexp.MustCompile(`[0-9A-Fa-f]{12,}`)
	reSecret  = regexp.MustCompile(`[A-Za-z0-9+/_=-]{24,}`)
	rePhone   = regexp.MustCompile(`(?:\+|\b)\d[\d ()-]{6,}\d`)
	reISODate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

var sensitiveWords = map[string]bool{
	"mac": true, "sn": true, "pin": true, "key": true, "sid": true, "user": true, "name": true,
	"host": true, "pwd": true, "psk": true, "wpa": true, "wep": true, "mail": true, "phone": true,
	"uuid": true, "udn": true, "guid": true, "wps": true, "account": true, "login": true,
	"pass": true, "msisdn": true, "iccid": true, "pppoe": true, "owner": true,
}

var sensitiveParts = []string{
	"password", "passwd", "passphrase", "ssid", "serial", "imei", "imsi", "macaddr", "hostname",
	"username", "secret", "cookie", "csrf", "token", "session", "credential", "email",
}

var versionWords = map[string]bool{
	"version": true, "firmware": true, "software": true, "release": true, "build": true, "fw": true, "sw": true,
}

func keyWords(name string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	rs := []rune(name)
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(cur) > 0 {
			prev := rs[i-1]
			lowerToUpper := unicode.IsLower(prev) && unicode.IsUpper(r)
			acronymEnd := unicode.IsUpper(prev) && unicode.IsUpper(r) && i+1 < len(rs) && unicode.IsLower(rs[i+1])
			digitEdge := unicode.IsDigit(prev) != unicode.IsDigit(r)
			if lowerToUpper || acronymEnd || digitEdge {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

func sensitiveKey(name string) bool {
	lower := strings.ToLower(name)
	for _, p := range sensitiveParts {
		if strings.Contains(lower, p) {
			return true
		}
	}
	for _, w := range keyWords(name) {
		if sensitiveWords[w] {
			return true
		}
	}
	return false
}

func versionKey(name string) bool {
	for _, w := range keyWords(name) {
		if versionWords[w] {
			return true
		}
	}
	return false
}

func entropy(s string) float64 {
	counts := map[rune]int{}
	total := 0
	for _, r := range s {
		counts[r]++
		total++
	}
	var bits float64
	for _, n := range counts {
		p := float64(n) / float64(total)
		bits -= p * math.Log2(p)
	}
	return bits
}

func plausibleWord(w string) bool {
	return len(w) >= minWordLen && strings.ContainsAny(w, "aeiouy")
}

func identifierLike(s string) bool {
	plausible, letters := 0, 0
	for _, w := range keyWords(s) {
		if unicode.IsDigit(rune(w[0])) {
			continue
		}
		letters += len(w)
		if plausibleWord(w) {
			plausible += len(w)
		}
	}
	return letters > 0 && float64(plausible)/float64(letters) >= minPlausible
}

func mixedSecret(s string) bool {
	var digit, letter bool
	for _, r := range s {
		digit = digit || unicode.IsDigit(r)
		letter = letter || unicode.IsLetter(r)
	}
	return digit && letter && !identifierLike(s) && entropy(s) >= minSecretBits
}

func maskIPv4(s string, keepDotted bool) string {
	keepDotted = keepDotted && !strings.ContainsAny(s, " \t")
	return reIPv4.ReplaceAllStringFunc(s, func(m string) string {
		ip := net.ParseIP(m)
		if ip == nil || keepDotted || isPrivate(m) {
			return m
		}
		return maskPublicIP
	})
}

func maskIPv6(s string) string {
	return reIPv6.ReplaceAllStringFunc(s, func(m string) string {
		bare, _, _ := strings.Cut(m, "%")
		ip := net.ParseIP(bare)
		if ip == nil || ip.To4() != nil || ip.IsUnspecified() || isPrivate(bare) {
			return m
		}
		return maskPublicIPv6
	})
}

func maskSecretRun(m string) string {
	if strings.HasPrefix(m, "/") {
		parts := strings.Split(m, "/")
		for i, p := range parts {
			if len(p) >= minSecretLen && mixedSecret(p) {
				parts[i] = maskSecret
			}
		}
		return strings.Join(parts, "/")
	}
	if mixedSecret(m) {
		return maskSecret
	}
	return m
}

func maskPhoneRun(m string) string {
	if reISODate.MatchString(m) {
		return m
	}
	digits := 0
	for _, r := range m {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	if digits < minPhoneDigits {
		return m
	}
	return maskPhone
}

func scrubString(s string, keepDotted bool) string {
	s = reLabeled.ReplaceAllString(s, "${1}${2}"+maskRedacted)
	s = reSN.ReplaceAllString(s, "${1}${2}"+maskRedacted)
	s = reEmail.ReplaceAllString(s, maskEmail)
	s = reMAC.ReplaceAllString(s, maskMAC)
	s = reMACDots.ReplaceAllString(s, maskMAC)
	s = maskIPv4(s, keepDotted)
	s = maskIPv6(s)
	s = reDigits.ReplaceAllString(s, maskNumber)
	s = reHex.ReplaceAllStringFunc(s, func(m string) string {
		if strings.ContainsAny(m, "0123456789") {
			return maskHex
		}
		return m
	})
	s = reSecret.ReplaceAllStringFunc(s, maskSecretRun)
	return rePhone.ReplaceAllStringFunc(s, maskPhoneRun)
}

func scrubTree(key string, v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			out[scrubString(k, false)] = scrubTree(k, child)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			out[i] = scrubTree(key, child)
		}
		return out
	case string:
		return scrubString(t, versionKey(key))
	default:
		return v
	}
}

func Seal(r Report) (Report, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return Report{}, err
	}
	var generic any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return Report{}, err
	}
	clean, err := json.Marshal(scrubTree("", generic))
	if err != nil {
		return Report{}, err
	}
	var out Report
	dec = json.NewDecoder(bytes.NewReader(clean))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return Report{}, err
	}
	return out, nil
}

func leaf(kind string, extra map[string]any) map[string]any {
	out := map[string]any{"type": kind}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func redactedLeaf() map[string]any { return leaf("redacted", nil) }

func looksPhoneLike(s string, trusted bool) bool {
	if len(s) >= minPhoneDigits+1 && (s[0] == '0' || s[0] == '+') {
		return true
	}
	return !trusted && len(s) > maxCountedDigit
}

func numericString(s string, trusted bool) (json.Number, bool) {
	if s == "" || looksPhoneLike(s, trusted) {
		return "", false
	}
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return "", false
	}
	return json.Number(s), true
}

func numberKind(n json.Number) string {
	if _, err := strconv.ParseUint(string(n), 10, 64); err == nil {
		return "int"
	}
	if _, err := n.Int64(); err == nil {
		return "int"
	}
	return "number"
}

var enumFields = map[string]bool{
	"NewWANAccessType": true, "NewPhysicalLinkStatus": true, "NewConnectionStatus": true,
	"NewLastConnectionError": true, "NewConnectionType": true, "NewPossibleConnectionTypes": true,
}

func enumLike(s string) bool {
	if s == "" || len(s) > maxEnumLen || !unicode.IsLetter(rune(s[0])) {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func stringLeaf(key, s string, trusted bool) map[string]any {
	if sensitiveKey(key) {
		return redactedLeaf()
	}
	if ip := net.ParseIP(s); ip != nil {
		scope := "public"
		if isPrivate(s) {
			scope = "private"
		}
		return leaf("ip", map[string]any{"scope": scope})
	}
	if n, ok := numericString(s, trusted); ok {
		kind := "numeric_string"
		if trusted {
			kind = numberKind(n)
		}
		return leaf(kind, map[string]any{"value": n})
	}
	if trusted && enumFields[key] && enumLike(s) {
		return leaf("enum", map[string]any{"value": s})
	}
	return leaf("string", map[string]any{"length": len([]rune(s))})
}

func shapeOf(key string, v any, depth int, trusted bool) any {
	switch t := v.(type) {
	case map[string]any:
		if depth >= maxShapeDepth {
			return leaf("object", map[string]any{"truncated": true})
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := map[string]any{}
		for i, k := range keys {
			if i >= maxShapeKeys {
				out["_truncated"] = true
				break
			}
			out[scrubString(k, false)] = shapeOf(k, t[k], depth+1, trusted)
		}
		return out
	case []any:
		out := leaf("array", map[string]any{"length": len(t)})
		if len(t) > 0 {
			out["item"] = shapeOf(key, t[0], depth+1, trusted)
		}
		return out
	case string:
		return stringLeaf(key, t, trusted)
	case json.Number:
		if sensitiveKey(key) {
			return redactedLeaf()
		}
		return leaf(numberKind(t), map[string]any{"value": t})
	case bool:
		if sensitiveKey(key) {
			return redactedLeaf()
		}
		return leaf("bool", map[string]any{"value": t})
	default:
		return leaf("null", nil)
	}
}

func ShapeJSON(body []byte) (any, bool) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	return shapeOf("", v, 0, false), true
}
