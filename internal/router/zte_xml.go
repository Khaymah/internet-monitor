package router

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
)

type zteNode struct {
	XMLName xml.Name
	Text    string    `xml:",chardata"`
	Nodes   []zteNode `xml:",any"`
}

type zteInstance struct {
	Object string
	Fields map[string]string
}

type zteReply struct {
	Status    string
	Instances []zteInstance
}

func parseZTEReply(body []byte) (zteReply, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '<' {
		return zteReply{}, false
	}
	var root zteNode
	if xml.Unmarshal(body, &root) != nil || root.XMLName.Local != "ajax_response_xml_root" {
		return zteReply{}, false
	}
	var out zteReply
	for _, n := range root.Nodes {
		if n.XMLName.Local == "IF_ERRORSTR" {
			out.Status = strings.TrimSpace(n.Text)
		}
		for _, in := range n.Nodes {
			if in.XMLName.Local == "Instance" {
				out.Instances = append(out.Instances, zteInstance{Object: n.XMLName.Local, Fields: zteFields(in)})
			}
		}
	}
	return out, true
}

func zteFields(in zteNode) map[string]string {
	fields := map[string]string{}
	var name string
	var haveName bool
	for _, c := range in.Nodes {
		switch c.XMLName.Local {
		case "ParaName":
			name, haveName = strings.TrimSpace(c.Text), true
		case "ParaValue":
			if haveName && name != "" {
				fields[name] = strings.TrimSpace(c.Text)
			}
			haveName = false
		}
	}
	return fields
}

func (r zteReply) first(object string) (map[string]string, bool) {
	for _, in := range r.Instances {
		if in.Object == object {
			return in.Fields, true
		}
	}
	return nil, false
}

func zteInt(fields map[string]string, key string) (int64, bool) {
	v, err := strconv.ParseInt(fields[key], 10, 64)
	return v, err == nil
}

func zteTenths(fields map[string]string, key string) (float64, bool) {
	v, ok := zteInt(fields, key)
	return float64(v) / 10, ok
}

func zteDSL(r zteReply) (WebStats, bool) {
	f, ok := r.first("OBJ_DSLINTERFACE_ID")
	if !ok || f["Status"] == "" {
		return WebStats{}, false
	}
	out := WebStats{Status: f["Status"], HasLineStats: true}
	out.DownKbps, _ = zteInt(f, "Downstream_current_rate")
	out.UpKbps, _ = zteInt(f, "Upstream_current_rate")
	out.MaxDownKbps, _ = zteInt(f, "Downstream_max_rate")
	out.MaxUpKbps, _ = zteInt(f, "Upstream_max_rate")
	out.SNRDown, _ = zteTenths(f, "Downstream_noise_margin")
	out.SNRUp, _ = zteTenths(f, "Upstream_noise_margin")
	out.AttnDown, _ = zteTenths(f, "Downstream_attenuation")
	out.AttnUp, _ = zteTenths(f, "Upstream_attenuation")
	return out, true
}

func zteWAN(r zteReply) (connected bool, uptime int64, ok bool) {
	var best map[string]string
	rank := -1
	for _, in := range r.Instances {
		f := in.Fields
		if f["ConnStatus"] == "" {
			continue
		}
		score := 0
		if strings.Contains(strings.ToLower(f["WANCName"]), "internet") {
			score = 1
		}
		if f["IsDefGW"] == "1" {
			score = 2
		}
		if score > rank {
			best, rank = f, score
		}
	}
	if best == nil {
		return false, 0, false
	}
	up, _ := zteInt(best, "UpTime")
	return strings.EqualFold(best["ConnStatus"], "Connected"), up, true
}

func zteModel(r zteReply) string {
	f, _ := r.first("OBJ_DEVINFO_ID")
	return f["ModelName"]
}
