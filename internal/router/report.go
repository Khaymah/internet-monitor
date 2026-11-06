package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

const (
	ReportSchema        = 1
	reportRequestWait   = 4 * time.Second
	reportWebWait       = 6 * time.Second
	reportPhaseWait     = 30 * time.Second
	reportBodyLimit     = 256 << 10
	tr064Port           = "49000"
	tr064DescPath       = "/tr64desc.xml"
	outcomeNotRequested = "not_requested"
	outcomeNoCreds   = "no_saved_password"
	outcomeNoDriver     = "no_matching_driver"
	outcomeSignedIn     = "signed_in"
	outcomeRejected     = "rejected"
	outcomeEarlier      = "previously_rejected"
	outcomeBusy         = "busy"
	outcomeFailed       = "failed"
)

var (
	ErrReportBusy       = errors.New("a router report is already being built")
	ErrReportNotPrivate = errors.New("the router address is not on the home network")
)

type Report struct {
	Schema    int              `json:"schema"`
	CreatedAt string           `json:"created_at"`
	App       ReportApp        `json:"app"`
	Router    ReportTarget     `json:"router"`
	UPnP      UPnPReport       `json:"upnp"`
	TR064     TR064Report      `json:"tr064"`
	Web       []WebFingerprint `json:"web"`
	Deep      DeepReport       `json:"deep"`
}

type ReportApp struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

type ReportTarget struct {
	Address string `json:"address"`
	Private bool   `json:"private"`
}

type DeviceInfo struct {
	DeviceType       string   `json:"device_type,omitempty"`
	Manufacturer     string   `json:"manufacturer,omitempty"`
	ModelName        string   `json:"model_name,omitempty"`
	ModelNumber      string   `json:"model_number,omitempty"`
	ModelDescription string   `json:"model_description,omitempty"`
	FriendlyName     string   `json:"friendly_name,omitempty"`
	SoftwareVersion  string   `json:"software_version,omitempty"`
	EmbeddedDevices  []string `json:"embedded_devices,omitempty"`
}

type ReportOptions struct {
	Version          string
	Host             string
	LocationHint     string
	Deep             bool
	User             string
	Password         string
	PasswordRejected bool
	Probers          []DeepProber
	Discover         func(ctx context.Context) (string, error)
	WebBases         []string
	TR064Base        string
	Now              func() time.Time
}

func (o ReportOptions) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o ReportOptions) address() string {
	if o.Host == "" {
		return DefaultHost
	}
	return o.Host
}

func (o ReportOptions) hostname() string {
	host := o.address()
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func (o ReportOptions) webBases() []string {
	if len(o.WebBases) > 0 {
		return o.WebBases
	}
	return []string{"https://" + o.address(), "http://" + o.address()}
}

func (o ReportOptions) tr064Base() string {
	if o.TR064Base != "" {
		return o.TR064Base
	}
	return "http://" + net.JoinHostPort(o.hostname(), tr064Port)
}

var reportGate sync.Mutex

func CollectReport(ctx context.Context, o ReportOptions) (Report, error) {
	if !reportGate.TryLock() {
		return Report{}, ErrReportBusy
	}
	defer reportGate.Unlock()
	if !isPrivate(o.hostname()) {
		return Report{}, fmt.Errorf("%w: %q", ErrReportNotPrivate, o.address())
	}
	rep := Report{
		Schema:    ReportSchema,
		CreatedAt: o.now().UTC().Format(time.RFC3339),
		App:       ReportApp{Version: o.Version, OS: runtime.GOOS, Arch: runtime.GOARCH},
		Router:    ReportTarget{Address: o.hostname(), Private: true},
	}
	var wg sync.WaitGroup
	phase := func(run func(context.Context)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(ctx, reportPhaseWait)
			defer cancel()
			run(pctx)
		}()
	}
	phase(func(c context.Context) { rep.UPnP = collectUPnP(c, o) })
	phase(func(c context.Context) { rep.TR064 = collectTR064(c, o) })
	phase(func(c context.Context) { rep.Web = collectWeb(c, o) })
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	rep.Deep = collectDeep(ctx, o, rep.Web)
	return Seal(rep)
}

func reportHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       reportRequestWait,
		Transport:     &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func reportGet(ctx context.Context, client *http.Client, target string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, reportBodyLimit))
	return resp, body, err
}

func errClass(err error) string {
	var ne net.Error
	var he httpError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &he):
		return he.Error()
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "unreachable"
	}
}
