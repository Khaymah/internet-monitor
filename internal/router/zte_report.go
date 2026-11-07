package router

import (
	"context"
	"encoding/json"
	"strings"
)

const zteNewFamily = "zte-web-new"

type zteRead struct {
	path string
	view string
	data string
}

var zteReads = []zteRead{
	{"/?_type=menuData&_tag=dsl_interface_status_lua.lua", "dslWanStatus", "dsl_interface_status_lua.lua"},
	{"/?_type=menuData&_tag=wan_internetstatus_lua.lua", "ethWanStatus&Menu3Location=0", "wan_internetstatus_lua.lua&TypeUplink=2&pageType=1"},
	{"/?_type=menuData&_tag=devmgr_statusmgr_lua.lua", "statusMgr&Menu3Location=0", "devmgr_statusmgr_lua.lua"},
}

func init() { RegisterDeepProber(zteNewProber{}) }

type zteNewProber struct{}

func (zteNewProber) Family() string { return zteNewFamily }

func (zteNewProber) Matches(web []WebFingerprint) bool {
	for _, fp := range web {
		if !fp.Answered {
			continue
		}
		for _, m := range fp.Markers {
			if m == "ZTE" || m == "ZXHN" {
				return true
			}
		}
		for _, p := range fp.Paths {
			if strings.Contains(p, "_type=loginData") || strings.Contains(p, "login_entry") {
				return true
			}
		}
	}
	return false
}

func zteTree(r zteReply) map[string]any {
	tree := map[string]any{"IF_ERRORSTR": r.Status}
	for _, in := range r.Instances {
		fields := make(map[string]any, len(in.Fields))
		for k, v := range in.Fields {
			fields[k] = v
		}
		list, _ := tree[in.Object].([]any)
		tree[in.Object] = append(list, fields)
	}
	return tree
}

func (zteNewProber) Probe(ctx context.Context, t DeepTarget) ([]EndpointShape, error) {
	client, err := webClient(t.Host, t.Timeout)
	if err != nil {
		return nil, err
	}
	s := &session{client: client}
	if _, err := s.pick(ctx, t.Host); err != nil {
		return nil, err
	}
	if err := s.zteLogin(ctx, t.User, t.Password); err != nil {
		return nil, err
	}
	defer s.zteLogoutNow(ctx)
	out := make([]EndpointShape, 0, len(zteReads))
	for _, rd := range zteReads {
		ep := EndpointShape{Path: rd.path}
		reply, err := s.zteData(ctx, rd.view, rd.data)
		if err != nil {
			ep.Error = "not_answered"
			out = append(out, ep)
			continue
		}
		raw, _ := json.Marshal(zteTree(reply))
		shape, ok := ShapeJSON(raw)
		if !ok {
			ep.Error = "not_xml"
			out = append(out, ep)
			continue
		}
		ep.Answered, ep.Shape = true, shape
		out = append(out, ep)
	}
	return out, nil
}
