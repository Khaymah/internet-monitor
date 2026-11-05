package app

import (
	"context"
	"errors"
	"internet-monitor/internal/alerts"
	"internet-monitor/internal/casefile"
	"internet-monitor/internal/config"
	"internet-monitor/internal/core"
	"internet-monitor/internal/link"
	"internet-monitor/internal/logger"
	"internet-monitor/internal/monitor"
	"internet-monitor/internal/notifytext"
	"internet-monitor/internal/pkgid"
	"internet-monitor/internal/power"
	"internet-monitor/internal/report"
	"internet-monitor/internal/router"
	"internet-monitor/internal/seal"
	"internet-monitor/internal/server"
	"internet-monitor/internal/sound"
	"internet-monitor/internal/trace"
	"internet-monitor/internal/tray"
	"internet-monitor/internal/types"
	"internet-monitor/internal/updater"
	"internet-monitor/internal/weekly"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Options struct {
	Version    string
	Dir        string
	ListenHost string
	Desktop    func(launchURL func() string) alerts.Desktop
}

type App struct {
	Dir     string
	Version string
	Store   *config.Store
	Log     *logger.Logger
	Server  *server.Server
	Engine  *core.Engine
	Alerts  *alerts.Alerts
	Weekly  *weekly.Scheduler
	Router  *router.Poller

	stop chan struct{}
	once sync.Once
}

func Dir() string {
	if pkgid.Packaged() {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "InternetMonitor")
		}
	}
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

func New(opts Options) (*App, error) {
	dir := opts.Dir
	if dir == "" {
		dir = Dir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.Chdir(dir); err != nil {
		log.Printf("warning: could not chdir to %s: %v", dir, err)
	}

	store, err := config.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		log.Printf("config: %v", err)
	}
	cfg := store.Get()

	lgr, err := logger.New(cfg)
	if err != nil {
		return nil, err
	}
	if lerr := store.LoadError(); lerr != nil {
		lgr.AppLog("CONFIG %v", lerr)
	}
	lgr.AppLog("START version=%s dir=%s", opts.Version, dir)

	tray.Logf = lgr.AppLog
	sound.Logf = lgr.AppLog
	sound.SetDir(dir)
	notifytext.SetLang(cfg.Language)

	history := alerts.OpenHistory(lgr.LogDir, lgr.AppLog)
	var al *alerts.Alerts
	wk := &weekly.Scheduler{
		Dir:   lgr.LogDir,
		State: weekly.StatePath(dir),
		Plan: func() float64 {
			f, _ := casefile.Open(dir).Load()
			return f.Profile.PlanDownloadMbps
		},
		Send: func(w types.WeeklySummary) { al.OnWeekly(w) },
		Logf: lgr.AppLog,
		Now:  time.Now,
	}
	poller := &router.Poller{
		Enabled: func() bool { return store.Get().RouterStatus },
		Web: func() (router.WebReader, bool) {
			c := store.Get()
			return router.WebReader{Host: c.RouterAddress, User: c.RouterUser, Password: c.RouterPassword}, c.RouterPassword != ""
		},
		Save: func(at time.Time, st router.Status) { lgr.LogJSON(router.Prefix, at, st) },
		Logf: lgr.AppLog,
	}
	srv := server.New(server.Options{
		Store:               store,
		Logger:              lgr,
		Version:             opts.Version,
		AppDir:              dir,
		NativeNotifications: opts.Desktop != nil,
		ListenHost:          opts.ListenHost,
		History:             history,
		Hooks: server.Hooks{
			TestNotification: func(lang string, volume *int) { al.Test(lang, volume) },
			PlaySound:        sound.PlayAt,
			TestWebhook:      lgr.SendTestWebhook,
			TestNtfy:         lgr.SendTestNtfy,
			CheckUpdate:      func() (*updater.Info, error) { return checkUpdate(opts.Version) },
			ApplyUpdate:      applyUpdate,
			Restart:          updater.Restart,
			SpeedResult:      func(ev logger.SpeedTestEvent, threshold float64, below bool) { al.OnSpeedResult(ev, threshold, below) },
			SendWeekly:       wk.SendNow,
			Router:           poller.Snapshot,
			TestRouter: func(user, password string) (router.WebStats, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
				defer cancel()
				stats, err := poller.TestWeb(ctx, router.WebReader{Host: store.Get().RouterAddress, User: user, Password: password})
				if err != nil {
					lgr.AppLog("ROUTER page test failed: %v", err)
				}
				return stats, err
			},
		},
	})
	var desktop alerts.Desktop
	if opts.Desktop != nil {
		desktop = opts.Desktop(srv.LaunchURL)
	}
	al = alerts.New(cfg, alerts.Options{
		Desktop: desktop,
		Remote:  lgr,
		History: history,
		Publish: srv.PublishNotification,
		Logf:    lgr.AppLog,
	})
	engine := core.New(cfg, monitor.NewChecker(cfg), lgr, opts.Version)
	links := link.NewCache()
	engine.Link = func() (string, int) {
		i := links.Get()
		return i.Type, i.Signal
	}
	engine.Power = power.OnBattery
	if boot, ok := power.BootTime(); ok {
		now := time.Now()
		lgr.LogJSON(report.SystemPrefix, now, report.Startup{Timestamp: now, Event: "start", BootTime: boot})
	}
	store.Subscribe(engine.ApplyConfig)
	store.Subscribe(al.ApplyConfig)

	if err := srv.Start(cfg.DashboardPort); err != nil {
		lgr.AppLog("port %d unavailable (%v); using a free port instead", cfg.DashboardPort, err)
		if err := srv.Start(0); err != nil {
			return nil, err
		}
	}
	lgr.AppLog("dashboard listening on %s", srv.URL())

	return &App{
		Dir:     dir,
		Version: opts.Version,
		Store:   store,
		Log:     lgr,
		Server:  srv,
		Engine:  engine,
		Alerts:  al,
		Weekly:  wk,
		Router:  poller,
		stop:    make(chan struct{}),
	}, nil
}

func (a *App) LogDir() string {
	dir := a.Store.Get().LogDir
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(a.Dir, dir)
}

func (a *App) Run(notifiers ...core.Notifier) {
	tracer := &trace.Recorder{
		Target: func() string {
			if t := a.Store.Get().PingTargets; len(t) > 0 {
				return t[0]
			}
			return ""
		},
		Save: func(at time.Time, t trace.Trace) { a.Log.LogJSON(trace.Prefix, at, t) },
		Logf: a.Log.AppLog,
	}
	sealer := &seal.Sealer{
		Dir: a.LogDir,
		Stamp: func() seal.Stamper {
			cfg := a.Store.Get()
			if !cfg.SealTimestamp {
				return nil
			}
			u := cfg.TSAURL
			if u == "" {
				u = seal.DefaultTSA
			}
			return seal.HTTPStamper(u, nil)
		},
		Logf: a.Log.AppLog,
	}
	go sealer.Run(a.stop)
	nudger := &weekly.Nudger{
		State: weekly.NudgePath(a.Dir),
		Check: func(days int) (int, bool) {
			for _, in := range a.Server.Insights(days, time.Now()) {
				if in.ID == "file_complaint" {
					n, _ := in.Vars["n"].(int)
					return n, true
				}
			}
			return 0, false
		},
		Send: a.Alerts.OnNudge,
		Now:  time.Now,
	}
	go nudger.Run(a.stop)
	go a.Router.Run(a.stop)
	quota := &weekly.QuotaWatcher{
		State: weekly.QuotaPath(a.Dir),
		Read: func(now time.Time) (weekly.QuotaReading, bool) {
			u, ok := a.Server.PlanUsage(now)
			if !ok || u.QuotaBytes == 0 {
				return weekly.QuotaReading{}, false
			}
			return weekly.QuotaReading{
				Period:   u.From,
				UsedPct:  u.UsedPct,
				UsedGB:   float64(u.UsedBytes) / 1e9,
				QuotaGB:  float64(u.QuotaBytes) / 1e9,
				DaysLeft: u.DaysLeft,
			}, true
		},
		Send: func(step int, r weekly.QuotaReading) {
			a.Alerts.OnQuota(alerts.Quota{Pct: int(r.UsedPct), UsedGB: r.UsedGB, QuotaGB: r.QuotaGB, DaysLeft: r.DaysLeft})
		},
		Now: time.Now,
	}
	go quota.Run(a.stop)
	multi := core.MultiNotifier{server.NewNotifier(a.Server), a.Alerts, tracer}
	multi = append(multi, notifiers...)
	a.Engine.Notifier = multi
	a.Engine.Start()
	go a.updateLoop()
	go a.Weekly.Run(a.stop)
}

func checkUpdate(version string) (*updater.Info, error) {
	if pkgid.Packaged() {
		return &updater.Info{CurrentVersion: version, LatestVersion: version, Managed: "store"}, nil
	}
	return updater.Check(version)
}

func applyUpdate(downloadURL string) error {
	if pkgid.Packaged() {
		return errors.New("updates for this copy come from the Microsoft Store")
	}
	return updater.Apply(downloadURL)
}

func (a *App) updateLoop() {
	if pkgid.Packaged() {
		return
	}
	select {
	case <-time.After(30 * time.Second):
	case <-a.stop:
		return
	}
	for {
		if a.Store.Get().AutoUpdateCheck {
			if info, err := updater.Check(a.Version); err == nil && info.HasUpdate {
				a.Log.AppLog("UPDATE available: %s (current: %s)", info.LatestVersion, info.CurrentVersion)
				a.Server.SetUpdateInfo(info)
			}
		}
		select {
		case <-time.After(6 * time.Hour):
		case <-a.stop:
			return
		}
	}
}

func (a *App) Stop() {
	a.once.Do(func() {
		close(a.stop)
		a.Engine.Stop()
		a.Alerts.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := a.Server.Shutdown(ctx); err != nil {
			a.Log.AppLog("dashboard shutdown: %v", err)
		}
		a.Log.AppLog("STOP")
		_ = a.Log.Close()
		ReleaseSingleInstance()
	})
}
