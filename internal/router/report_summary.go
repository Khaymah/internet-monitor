package router

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	IssueNewURL     = "https://github.com/Khaymah/internet-monitor/issues/new"
	IssueLabel      = "router-support"
	maxIssueURL     = 6000
	maxTitleModel   = 60
	unknownRouter   = "unknown router"
	filenamePrefix  = "router-report-"
	maxFilenameSlug = 40
)

var reSlugBad = regexp.MustCompile(`[^a-z0-9]+`)

type ReportResult struct {
	Report   Report `json:"report"`
	Summary  string `json:"summary"`
	Filename string `json:"filename"`
	IssueURL string `json:"issue_url"`
}

func (r Report) Identity() (manufacturer, model string) {
	for _, d := range []*DeviceInfo{r.UPnP.Device, r.TR064.Device} {
		if d == nil {
			continue
		}
		if manufacturer == "" {
			manufacturer = d.Manufacturer
		}
		if model == "" {
			model = d.ModelName
		}
		if model == "" {
			model = d.ModelNumber
		}
	}
	if manufacturer == "" && model == "" {
		for _, fp := range r.Web {
			if len(fp.Markers) > 0 {
				return fp.Markers[0], ""
			}
		}
	}
	return manufacturer, model
}

func (r Report) displayName() string {
	m, model := r.Identity()
	name := strings.TrimSpace(m + " " + model)
	if name == "" {
		return unknownRouter
	}
	return clip(name, maxTitleModel)
}

func reportFilename(r Report) string {
	_, model := r.Identity()
	if model == "" {
		model, _ = r.Identity()
	}
	slug := strings.Trim(reSlugBad.ReplaceAllString(strings.ToLower(model), "-"), "-")
	if slug == "" {
		slug = "unknown"
	}
	if len(slug) > maxFilenameSlug {
		slug = strings.Trim(slug[:maxFilenameSlug], "-")
	}
	return filenamePrefix + slug + ".json"
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func answeredCount(actions []ActionProbe) int {
	n := 0
	for _, a := range actions {
		if a.Answered {
			n++
		}
	}
	return n
}

func Summary(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Router: %s\n", r.displayName())
	if d := r.UPnP.Device; d != nil && d.SoftwareVersion != "" {
		fmt.Fprintf(&b, "Firmware: %s\n", d.SoftwareVersion)
	}
	fmt.Fprintf(&b, "App: %s on %s/%s\n", r.App.Version, r.App.OS, r.App.Arch)
	fmt.Fprintf(&b, "UPnP: answered %s, %d services, %d of %d status actions answered\n",
		yesNo(r.UPnP.Answered), len(r.UPnP.Services), answeredCount(r.UPnP.Actions), len(r.UPnP.Actions))
	fmt.Fprintf(&b, "TR-064: found %s\n", yesNo(r.TR064.Found))
	for _, fp := range r.Web {
		if !fp.Answered {
			fmt.Fprintf(&b, "Web %s:%s: no answer (%s)\n", fp.Scheme, fp.Port, fp.Error)
			continue
		}
		fmt.Fprintf(&b, "Web %s:%s: HTTP %d, title %q, server %q, markers %s\n",
			fp.Scheme, fp.Port, fp.Status, fp.Title, fp.Server, strings.Join(fp.Markers, "/"))
	}
	fmt.Fprintf(&b, "Signed-in probe: %s", r.Deep.Outcome)
	if r.Deep.Family != "" {
		fmt.Fprintf(&b, " (%s, %d endpoints answered)", r.Deep.Family, answeredEndpoints(r.Deep.Endpoints))
	}
	b.WriteString("\n")
	return b.String()
}

func answeredEndpoints(eps []EndpointShape) int {
	n := 0
	for _, e := range eps {
		if e.Answered {
			n++
		}
	}
	return n
}

func issueURL(title, summary, filename string) string {
	tail := fmt.Sprintf("\n\nAttach the downloaded file `%s` below (drag it into this box).\n", filename)
	lines := strings.Split(strings.TrimRight(summary, "\n"), "\n")
	for {
		body := strings.Join(lines, "\n") + tail
		q := url.Values{}
		q.Set("title", title)
		q.Set("labels", IssueLabel)
		q.Set("body", body)
		full := IssueNewURL + "?" + q.Encode()
		if len(full) <= maxIssueURL || len(lines) <= 1 {
			return full
		}
		lines = lines[:len(lines)-1]
	}
}

func BuildResult(r Report) ReportResult {
	summary := Summary(r)
	filename := reportFilename(r)
	title := "Router support: " + r.displayName()
	return ReportResult{
		Report:   r,
		Summary:  summary,
		Filename: filename,
		IssueURL: issueURL(title, summary, filename),
	}
}
