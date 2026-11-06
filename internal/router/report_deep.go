package router

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	deepWait         = 30 * time.Second
	deepRequestWait  = 8 * time.Second
	huaweiFamily     = "huawei-web"
	huaweiLoginPath  = "/api/system/user_login"
	huaweiCSRFMeta   = "csrf_param"
	defaultLoginUser = "admin"
)

var huaweiReadPaths = []string{
	"/api/ntwk/dslinfo",
	"/api/ntwk/wan_st",
	"/api/system/deviceinfo",
	"/api/ntwk/wandetect",
}

type DeepReport struct {
	Requested bool            `json:"requested"`
	Outcome   string          `json:"outcome"`
	Family    string          `json:"family,omitempty"`
	Endpoints []EndpointShape `json:"endpoints,omitempty"`
}

type EndpointShape struct {
	Path     string `json:"path"`
	Answered bool   `json:"answered"`
	Error    string `json:"error,omitempty"`
	Shape    any    `json:"shape,omitempty"`
}

type DeepTarget struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
}

type DeepProber interface {
	Family() string
	Matches(web []WebFingerprint) bool
	Probe(ctx context.Context, t DeepTarget) ([]EndpointShape, error)
}

var (
	proberMu    sync.RWMutex
	probers     = []DeepProber{huaweiProber{}}
	errNoProber = errors.New("no prober")
)

func RegisterDeepProber(p DeepProber) {
	proberMu.Lock()
	defer proberMu.Unlock()
	probers = append(probers, p)
}

func registeredProbers() []DeepProber {
	proberMu.RLock()
	defer proberMu.RUnlock()
	return append([]DeepProber(nil), probers...)
}

func pickProber(list []DeepProber, web []WebFingerprint) (DeepProber, error) {
	for _, p := range list {
		if p.Matches(web) {
			return p, nil
		}
	}
	return nil, errNoProber
}

func collectDeep(ctx context.Context, o ReportOptions, web []WebFingerprint) DeepReport {
	if !o.Deep {
		return DeepReport{Outcome: outcomeNotRequested}
	}
	rep := DeepReport{Requested: true}
	if o.Password == "" {
		rep.Outcome = outcomeNoCreds
		return rep
	}
	list := o.Probers
	if list == nil {
		list = registeredProbers()
	}
	p, err := pickProber(list, web)
	if err != nil {
		rep.Outcome = outcomeNoDriver
		return rep
	}
	rep.Family = p.Family()
	if o.PasswordRejected {
		rep.Outcome = outcomeEarlier
		return rep
	}
	pctx, cancel := context.WithTimeout(ctx, deepWait)
	defer cancel()
	user := strings.TrimSpace(o.User)
	if user == "" {
		user = defaultLoginUser
	}
	endpoints, err := p.Probe(pctx, DeepTarget{Host: o.address(), User: user, Password: o.Password, Timeout: deepRequestWait})
	rep.Endpoints = endpoints
	switch {
	case err == nil:
		rep.Outcome = outcomeSignedIn
	case errors.Is(err, ErrWrongPassword):
		rep.Outcome = outcomeRejected
	case errors.Is(err, ErrRouterBusy):
		rep.Outcome = outcomeBusy
	default:
		rep.Outcome = outcomeFailed
	}
	return rep
}

type huaweiProber struct{}

func (huaweiProber) Family() string { return huaweiFamily }

func (huaweiProber) Matches(web []WebFingerprint) bool {
	for _, fp := range web {
		if !fp.Answered {
			continue
		}
		for _, m := range fp.Markers {
			if m == "Huawei" {
				return true
			}
		}
		for _, m := range fp.MetaNames {
			if m == huaweiCSRFMeta {
				return true
			}
		}
		for _, p := range fp.Paths {
			if strings.HasPrefix(p, huaweiLoginPath) {
				return true
			}
		}
	}
	return false
}

func (huaweiProber) Probe(ctx context.Context, t DeepTarget) ([]EndpointShape, error) {
	client, err := webClient(t.Host, t.Timeout)
	if err != nil {
		return nil, err
	}
	s := &session{client: client}
	if err := s.pick(ctx, t.Host); err != nil {
		return nil, err
	}
	if err := s.login(ctx, t.User, t.Password); err != nil {
		return nil, err
	}
	defer s.logout(ctx)
	out := make([]EndpointShape, 0, len(huaweiReadPaths))
	for _, path := range huaweiReadPaths {
		ep := EndpointShape{Path: path}
		body, _, err := s.get(ctx, path)
		if err != nil {
			ep.Error = "not_answered"
			out = append(out, ep)
			continue
		}
		shape, ok := ShapeJSON(unwrap(body))
		if !ok {
			ep.Error = "not_json"
			out = append(out, ep)
			continue
		}
		ep.Answered, ep.Shape = true, shape
		out = append(out, ep)
	}
	return out, nil
}
