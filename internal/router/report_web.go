package router

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

const (
	maxTitleLen    = 120
	maxPathLen     = 120
	maxScriptFetch = 8
	maxListedRefs  = 30
	maxForms       = 6
	maxInputs      = 30
	maxPaths       = 60
	maxMetaNames   = 10
	maxCookieNames = 8
	maxCorpus      = 1 << 20
	httpsPort      = "443"
	httpPort       = "80"
)

type WebFingerprint struct {
	Scheme       string     `json:"scheme"`
	Port         string     `json:"port"`
	Answered     bool       `json:"answered"`
	Error        string     `json:"error,omitempty"`
	Status       int        `json:"status,omitempty"`
	RedirectedTo string     `json:"redirected_to,omitempty"`
	Server       string     `json:"server,omitempty"`
	AuthRealm    string     `json:"auth_realm,omitempty"`
	Title        string     `json:"title,omitempty"`
	CookieNames  []string   `json:"cookie_names,omitempty"`
	MetaNames    []string   `json:"meta_names,omitempty"`
	Forms        []FormInfo `json:"forms,omitempty"`
	Scripts      []string   `json:"scripts,omitempty"`
	Frames       []string   `json:"frames,omitempty"`
	Paths        []string   `json:"paths,omitempty"`
	Markers      []string   `json:"markers,omitempty"`
}

type FormInfo struct {
	Action string   `json:"action"`
	Method string   `json:"method,omitempty"`
	Inputs []string `json:"inputs,omitempty"`
}

const looseInputsAction = "(inputs outside a form)"

var markerNames = []string{
	"ZTE", "ZXHN", "Huawei", "TP-Link", "Archer", "Zyxel", "Nokia", "Alcatel", "AVM",
	"FRITZ!Box", "D-Link", "Netgear", "Tenda", "Mercusys", "Technicolor", "Sagemcom", "Arris", "Askey",
	"Ubiquiti", "MikroTik", "ASUS", "Linksys", "Vodafone", "Etisalat", "TE Data", "Genexis",
	"FiberHome", "Realtek", "Broadcom", "OpenWrt", "DD-WRT", "Keenetic", "Netis", "Totolink", "Skyworth",
	"Sercomm", "Xiaomi", "Digisol", "Draytek", "Cisco", "Ericsson", "Calix",
}

var (
	reRealm      = regexp.MustCompile(`(?i)realm="([^"]*)"`)
	reQuoted     = regexp.MustCompile("[\"'`]((?:/|\\./|\\.\\./)?[A-Za-z0-9_\\-.~/%]*(?:\\?[A-Za-z0-9_=&.%:,/-]{0,80})?)[\"'`]")
	pathKeywords = []string{
		"api", "cgi", "asp", "goform", "login", "logout", "auth", "session", "ubus", "rpc", "json", "xml",
		"_type", "status", "wan", "dsl", "ppp", "device", "system", "ntwk", "network", "info", "token", "rand",
	}
	staticSuffixes = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".css", ".woff", ".woff2", ".ttf", ".map", ".eot"}
)

func collectWeb(ctx context.Context, o ReportOptions) []WebFingerprint {
	var out []WebFingerprint
	for _, base := range o.webBases() {
		out = append(out, fingerprint(ctx, o.hostname(), base))
	}
	return out
}

func defaultPort(scheme string) string {
	if scheme == "https" {
		return httpsPort
	}
	return httpPort
}

func refToPath(ref string, base *url.URL) string {
	u, err := base.Parse(strings.TrimSpace(ref))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	if u.Host != base.Host && !isPrivate(u.Hostname()) {
		return "external:" + u.Hostname()
	}
	out := u.EscapedPath()
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if len(out) > maxPathLen {
		out = out[:maxPathLen]
	}
	return out
}

func fingerprint(ctx context.Context, host, base string) WebFingerprint {
	bu, err := url.Parse(base)
	if err != nil {
		return WebFingerprint{Error: "bad_address"}
	}
	fp := WebFingerprint{Scheme: bu.Scheme, Port: bu.Port()}
	if fp.Port == "" {
		fp.Port = defaultPort(bu.Scheme)
	}
	client, err := webClient(host, reportWebWait)
	if err != nil {
		fp.Error = "not_private"
		return fp
	}
	resp, body, err := reportGet(ctx, client, base+"/")
	if err != nil && resp == nil {
		fp.Error = errClass(err)
		return fp
	}
	fp.Answered = true
	fp.Status = resp.StatusCode
	fp.Server = clip(resp.Header.Get("Server"), maxTitleLen)
	if m := reRealm.FindStringSubmatch(resp.Header.Get("Www-Authenticate")); m != nil {
		fp.AuthRealm = clip(m[1], maxTitleLen)
	}
	for _, c := range resp.Cookies() {
		if len(fp.CookieNames) < maxCookieNames {
			fp.CookieNames = append(fp.CookieNames, c.Name)
		}
	}
	final := resp.Request.URL
	if p := refToPath(final.String(), bu); p != "" && p != "/" {
		fp.RedirectedTo = p
	}
	page := parsePage(body, final)
	fp.Title, fp.MetaNames, fp.Forms = page.title, page.meta, page.forms
	fp.Scripts, fp.Frames = page.scripts, page.frames
	corpus := string(body) + "\n" + resp.Header.Get("Server") + "\n" + resp.Header.Get("Www-Authenticate")
	corpus += fetchScripts(ctx, client, final, page.scriptRefs)
	fp.Paths = interestingPaths(corpus)
	fp.Markers = findMarkers(corpus)
	return fp
}

type pageInfo struct {
	title      string
	meta       []string
	forms      []FormInfo
	scripts    []string
	frames     []string
	scriptRefs []string
}

func attrs(t html.Token) map[string]string {
	out := map[string]string{}
	for _, a := range t.Attr {
		out[strings.ToLower(a.Key)] = a.Val
	}
	return out
}

func inputLabel(a map[string]string) string {
	name := a["name"]
	if name == "" {
		name = a["id"]
	}
	if name == "" {
		return ""
	}
	if kind := a["type"]; kind != "" {
		return name + " (" + strings.ToLower(kind) + ")"
	}
	return name
}

func parsePage(body []byte, base *url.URL) pageInfo {
	var p pageInfo
	z := html.NewTokenizer(strings.NewReader(string(body)))
	var inTitle bool
	var cur *FormInfo
	loose := FormInfo{Action: looseInputsAction}
	closeForm := func() {
		if cur != nil && len(p.forms) < maxForms {
			p.forms = append(p.forms, *cur)
		}
		cur = nil
	}
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		switch tt {
		case html.TextToken:
			if inTitle && p.title == "" {
				p.title = clip(strings.Join(strings.Fields(string(z.Text())), " "), maxTitleLen)
			}
		case html.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "title":
				inTitle = false
			case "form":
				closeForm()
			}
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			a := attrs(t)
			switch t.Data {
			case "title":
				inTitle = true
			case "meta":
				if n := a["name"]; n != "" && len(p.meta) < maxMetaNames {
					p.meta = append(p.meta, n)
				}
			case "script":
				if src := a["src"]; src != "" {
					p.scriptRefs = append(p.scriptRefs, src)
					if ref := refToPath(src, base); ref != "" && len(p.scripts) < maxListedRefs {
						p.scripts = append(p.scripts, ref)
					}
				}
			case "frame", "iframe":
				if ref := refToPath(a["src"], base); ref != "" && len(p.frames) < maxListedRefs {
					p.frames = append(p.frames, ref)
				}
			case "form":
				closeForm()
				cur = &FormInfo{Action: refToPath(a["action"], base), Method: strings.ToUpper(a["method"])}
			case "input", "select", "textarea":
				label := inputLabel(a)
				target := &loose
				if cur != nil {
					target = cur
				}
				if label != "" && len(target.Inputs) < maxInputs {
					target.Inputs = append(target.Inputs, label)
				}
			}
		}
	}
	closeForm()
	if len(loose.Inputs) > 0 && len(p.forms) < maxForms {
		p.forms = append(p.forms, loose)
	}
	return p
}

func clip(s string, limit int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > limit {
		return string(r[:limit])
	}
	return string(r)
}

func fetchScripts(ctx context.Context, client *http.Client, base *url.URL, refs []string) string {
	var sb strings.Builder
	fetched := 0
	for _, ref := range refs {
		u, err := base.Parse(ref)
		if err != nil || u.Host != base.Host || fetched >= maxScriptFetch || sb.Len() >= maxCorpus {
			continue
		}
		fetched++
		resp, body, err := reportGet(ctx, client, u.String())
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		sb.WriteString("\n")
		sb.Write(body)
	}
	return sb.String()
}

func interestingPath(p string) bool {
	if len(p) < 3 {
		return false
	}
	lower := strings.ToLower(p)
	bare, _, _ := strings.Cut(lower, "?")
	for _, s := range staticSuffixes {
		if strings.HasSuffix(bare, s) {
			return false
		}
	}
	rooted := strings.HasPrefix(p, "/")
	dynamic := false
	for _, ext := range []string{".cgi", ".asp", ".aspx", ".lua", ".cmd", ".jsp", ".php"} {
		dynamic = dynamic || strings.Contains(bare, ext)
	}
	if !rooted && !dynamic {
		return false
	}
	for _, k := range pathKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func interestingPaths(corpus string) []string {
	seen := map[string]bool{}
	for _, m := range reQuoted.FindAllStringSubmatch(corpus, -1) {
		p := m[1]
		if len(p) > maxPathLen || !interestingPath(p) {
			continue
		}
		seen[p] = true
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	if len(out) > maxPaths {
		out = out[:maxPaths]
	}
	return out
}

var markerPatterns = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(markerNames))
	for _, m := range markerNames {
		out[m] = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(m) + `(?:[^A-Za-z]|$)`)
	}
	return out
}()

func findMarkers(corpus string) []string {
	var out []string
	for _, m := range markerNames {
		if markerPatterns[m].MatchString(corpus) {
			out = append(out, m)
		}
	}
	return out
}
