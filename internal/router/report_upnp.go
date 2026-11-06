package router

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
)

const (
	maxReportServices = 24
	maxReportActions  = 80
	maxEmbedded       = 12
)

type UPnPReport struct {
	Answered bool          `json:"answered"`
	Error    string        `json:"error,omitempty"`
	Device   *DeviceInfo   `json:"device,omitempty"`
	Services []ServiceInfo `json:"services,omitempty"`
	Actions  []ActionProbe `json:"status_actions,omitempty"`
}

type ServiceInfo struct {
	Type         string   `json:"type"`
	Actions      []string `json:"actions,omitempty"`
	ActionsError string   `json:"actions_error,omitempty"`
}

type ActionProbe struct {
	Service  string         `json:"service"`
	Action   string         `json:"action"`
	Answered bool           `json:"answered"`
	Error    string         `json:"error,omitempty"`
	Fields   map[string]any `json:"fields,omitempty"`
}

type descDevice struct {
	DeviceType      string        `xml:"deviceType"`
	Manufacturer    string        `xml:"manufacturer"`
	ModelName       string        `xml:"modelName"`
	ModelNumber     string        `xml:"modelNumber"`
	ModelDesc       string        `xml:"modelDescription"`
	FriendlyName    string        `xml:"friendlyName"`
	SoftwareVersion string        `xml:"softwareVersion"`
	FirmwareVersion string        `xml:"firmwareVersion"`
	Services        []descService `xml:"serviceList>service"`
	Devices         []descDevice  `xml:"deviceList>device"`
}

type descService struct {
	Type string `xml:"serviceType"`
	SCPD string `xml:"SCPDURL"`
}

type descRoot struct {
	URLBase string     `xml:"URLBase"`
	Device  descDevice `xml:"device"`
}

type scpdRoot struct {
	Actions []struct {
		Name string `xml:"name"`
	} `xml:"actionList>action"`
}

func deviceInfo(d descDevice) *DeviceInfo {
	info := &DeviceInfo{
		DeviceType:       strings.TrimSpace(d.DeviceType),
		Manufacturer:     strings.TrimSpace(d.Manufacturer),
		ModelName:        strings.TrimSpace(d.ModelName),
		ModelNumber:      strings.TrimSpace(d.ModelNumber),
		ModelDescription: strings.TrimSpace(d.ModelDesc),
		SoftwareVersion:  strings.TrimSpace(d.SoftwareVersion),
	}
	if info.SoftwareVersion == "" {
		info.SoftwareVersion = strings.TrimSpace(d.FirmwareVersion)
	}
	friendly := strings.TrimSpace(d.FriendlyName)
	if info.ModelName != "" && strings.Contains(strings.ToLower(friendly), strings.ToLower(info.ModelName)) {
		info.FriendlyName = friendly
	}
	var walkEmbedded func(children []descDevice)
	walkEmbedded = func(children []descDevice) {
		for _, c := range children {
			if len(info.EmbeddedDevices) < maxEmbedded {
				info.EmbeddedDevices = append(info.EmbeddedDevices, strings.TrimSpace(c.DeviceType))
			}
			walkEmbedded(c.Devices)
		}
	}
	walkEmbedded(d.Devices)
	return info
}

func descServices(d descDevice) []descService {
	out := append([]descService(nil), d.Services...)
	for _, c := range d.Devices {
		out = append(out, descServices(c)...)
	}
	return out
}

func upnpLocation(ctx context.Context, o ReportOptions) string {
	discover := o.Discover
	if discover == nil {
		discover = func(ctx context.Context) (string, error) { return Discover(ctx, discoverWait) }
	}
	if loc, err := discover(ctx); err == nil && loc != "" {
		return loc
	}
	return o.LocationHint
}

func privateHTTP(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "http" && isPrivate(u.Hostname())
}

func collectUPnP(ctx context.Context, o ReportOptions) UPnPReport {
	loc := upnpLocation(ctx, o)
	if loc == "" {
		return UPnPReport{Error: "no_ssdp_answer"}
	}
	if !privateHTTP(loc) {
		return UPnPReport{Error: "location_not_private"}
	}
	client := reportHTTPClient()
	resp, body, err := reportGet(ctx, client, loc)
	if err != nil {
		return UPnPReport{Error: errClass(err)}
	}
	if resp.StatusCode != http.StatusOK {
		return UPnPReport{Error: httpError{resp.StatusCode}.Error()}
	}
	var root descRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return UPnPReport{Error: "description_not_xml"}
	}
	rep := UPnPReport{Answered: true, Device: deviceInfo(root.Device)}
	base := loc
	if root.URLBase != "" && hostOf(root.URLBase) == hostOf(loc) {
		base = root.URLBase
	}
	for i, s := range descServices(root.Device) {
		if i >= maxReportServices {
			break
		}
		info := ServiceInfo{Type: strings.TrimSpace(s.Type)}
		scpd := resolve(base, strings.TrimSpace(s.SCPD))
		if s.SCPD == "" || hostOf(scpd) != hostOf(loc) {
			info.ActionsError = "no_description"
		} else {
			info.Actions, info.ActionsError = serviceActions(ctx, client, scpd)
		}
		rep.Services = append(rep.Services, info)
	}
	rep.Actions = probeStatusActions(ctx, loc)
	return rep
}

func serviceActions(ctx context.Context, client *http.Client, scpd string) ([]string, string) {
	resp, body, err := reportGet(ctx, client, scpd)
	if err != nil {
		return nil, errClass(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpError{resp.StatusCode}.Error()
	}
	var doc scpdRoot
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, "description_not_xml"
	}
	var names []string
	for _, a := range doc.Actions {
		if len(names) >= maxReportActions {
			break
		}
		names = append(names, strings.TrimSpace(a.Name))
	}
	return names, ""
}

func probeStatusActions(ctx context.Context, loc string) []ActionProbe {
	g, err := Open(ctx, loc, nil)
	if err != nil {
		return []ActionProbe{{Service: "WAN", Action: "Open", Error: errClass(err)}}
	}
	type call struct{ url, service, action string }
	var calls []call
	if g.commonURL != "" {
		for _, a := range []string{"GetCommonLinkProperties", "GetTotalBytesReceived", "GetTotalBytesSent"} {
			calls = append(calls, call{g.commonURL, svcCommon, a})
		}
	}
	if g.connURL != "" {
		for _, a := range []string{"GetStatusInfo", "GetLinkLayerMaxBitRates"} {
			calls = append(calls, call{g.connURL, g.connType, a})
		}
	}
	var out []ActionProbe
	for _, c := range calls {
		probe := ActionProbe{Service: c.service, Action: c.action}
		fields, err := g.call(ctx, c.url, c.service, c.action)
		if err != nil {
			probe.Error = errClass(err)
		} else {
			probe.Answered = true
			probe.Fields = upnpShapes(fields)
		}
		out = append(out, probe)
	}
	return out
}

func upnpShapes(fields map[string]string) map[string]any {
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		out[scrubString(k, false)] = stringLeaf(k, v, true)
	}
	return out
}
