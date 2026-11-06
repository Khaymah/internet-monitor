package router

import (
	"context"
	"encoding/xml"
	"net/http"
	"strings"
)

type TR064Report struct {
	Found    bool        `json:"found"`
	Status   int         `json:"status,omitempty"`
	Error    string      `json:"error,omitempty"`
	Device   *DeviceInfo `json:"device,omitempty"`
	Services []string    `json:"services,omitempty"`
}

func collectTR064(ctx context.Context, o ReportOptions) TR064Report {
	resp, body, err := reportGet(ctx, reportHTTPClient(), o.tr064Base()+tr064DescPath)
	if err != nil {
		return TR064Report{Error: errClass(err)}
	}
	rep := TR064Report{Status: resp.StatusCode}
	if resp.StatusCode != http.StatusOK {
		return rep
	}
	var root descRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		rep.Error = "description_not_xml"
		return rep
	}
	rep.Found = true
	rep.Device = deviceInfo(root.Device)
	for i, s := range descServices(root.Device) {
		if i >= maxReportServices {
			break
		}
		rep.Services = append(rep.Services, strings.TrimSpace(s.Type))
	}
	return rep
}
