package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"internet-monitor/internal/config"
	"internet-monitor/internal/i18n"
	"internet-monitor/internal/jsonl"
	"internet-monitor/internal/monitor"
	"internet-monitor/internal/types"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	cfg     config.Config
	mu      sync.Mutex
	appMu   sync.Mutex
	appLog  *log.Logger
	appFile *os.File
	appPath string
}

const maxAppLogSize = 5 << 20

func New(cfg config.Config) (*Logger, error) {
	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return nil, err
	}

	appLogFile := filepath.Join(cfg.LogDir, "app.log")
	if info, err := os.Stat(appLogFile); err == nil && info.Size() > maxAppLogSize {
		_ = os.Rename(appLogFile, appLogFile+".1")
	}
	f, err := os.OpenFile(appLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	var appLog *log.Logger
	if err == nil {
		appLog = log.New(f, "", log.LstdFlags)
	} else {
		f = nil
	}

	return &Logger{cfg: cfg.Clone(), appLog: appLog, appFile: f, appPath: appLogFile}, nil
}

func (l *Logger) Close() error {
	l.appMu.Lock()
	defer l.appMu.Unlock()
	if l.appFile == nil {
		return nil
	}
	err := l.appFile.Close()
	l.appFile = nil
	l.appLog = nil
	return err
}

func (l *Logger) LogDir() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cfg.LogDir
}

func (l *Logger) SetConfig(cfg config.Config) {
	if cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
			l.AppLog("ERROR create log dir on config reload: %v", err)
			return
		}
	}
	l.mu.Lock()
	l.cfg = cfg.Clone()
	l.mu.Unlock()
}

func (l *Logger) AppLog(format string, args ...interface{}) {
	l.appMu.Lock()
	defer l.appMu.Unlock()
	if l.appLog == nil {
		return
	}
	l.appLog.Printf(format, args...)
	if info, err := l.appFile.Stat(); err == nil && info.Size() > maxAppLogSize {
		l.rotateAppLogLocked()
	}
}

func (l *Logger) rotateAppLogLocked() {
	_ = l.appFile.Close()
	_ = os.Rename(l.appPath, l.appPath+".1")
	f, err := os.OpenFile(l.appPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		l.appFile, l.appLog = nil, nil
		return
	}
	l.appFile, l.appLog = f, log.New(f, "", log.LstdFlags)
}

func (l *Logger) Log(event monitor.Event) {
	l.mu.Lock()
	defer l.mu.Unlock()

	filename := filepath.Join(
		l.cfg.LogDir,
		fmt.Sprintf("connectivity_%s.jsonl", time.Now().Format("2006-01-02")),
	)

	data, err := json.Marshal(event)
	if err != nil {
		l.AppLog("ERROR marshal event: %v", err)
		return
	}
	data = append(data, '\n')

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.Write(data)
		f.Close()
	} else {
		l.AppLog("ERROR open log file: %v", err)
	}

}

func (l *Logger) LastEvent() (types.Event, bool) {
	l.mu.Lock()
	dir := l.cfg.LogDir
	l.mu.Unlock()
	files, _ := filepath.Glob(filepath.Join(dir, "connectivity_*.jsonl"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	for _, f := range files {
		lines, err := jsonl.Tail(f, 1)
		if err != nil || len(lines) == 0 {
			continue
		}
		var ev types.Event
		if json.Unmarshal(lines[0], &ev) == nil {
			return ev, true
		}
	}
	return types.Event{}, false
}

type SpeedTestEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	Event           string    `json:"event"`
	DownloadMbps    float64   `json:"download_mbps"`
	UploadMbps      *float64  `json:"upload_mbps"`
	LatencyMs       int64     `json:"latency_ms"`
	DurationSeconds float64   `json:"duration_seconds"`
	Endpoints       []string  `json:"endpoints"`
	ParallelConns   int       `json:"parallel_connections"`
	TriggeredBy     string    `json:"triggered_by"`
	IdleLatencyMs   int64     `json:"idle_latency_ms,omitempty"`
	LoadedDownMs    int64     `json:"loaded_latency_down_ms,omitempty"`
	LoadedUpMs      int64     `json:"loaded_latency_up_ms,omitempty"`
	Bufferbloat     string    `json:"bufferbloat,omitempty"`
}

func (l *Logger) LogSpeedTest(event SpeedTestEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	filename := filepath.Join(
		l.cfg.LogDir,
		fmt.Sprintf("speedtest_%s.jsonl", time.Now().Format("2006-01-02")),
	)

	data, err := json.Marshal(event)
	if err != nil {
		l.AppLog("ERROR marshal speed test event: %v", err)
		return
	}
	data = append(data, '\n')

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.Write(data)
		f.Close()
	} else {
		l.AppLog("ERROR open speedtest log: %v", err)
	}
}

func (l *Logger) LogSample(s types.MetricSample) {
	l.mu.Lock()
	defer l.mu.Unlock()

	date := s.Timestamp
	if date.IsZero() {
		date = time.Now()
	}
	filename := filepath.Join(
		l.cfg.LogDir,
		fmt.Sprintf("metrics_%s.jsonl", date.Format("2006-01-02")),
	)
	data, err := json.Marshal(s)
	if err != nil {
		l.AppLog("ERROR marshal metric sample: %v", err)
		return
	}
	data = append(data, '\n')

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.Write(data)
		f.Close()
	} else {
		l.AppLog("ERROR open metrics log: %v", err)
	}
}

func (l *Logger) CleanupOldLogs(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := os.ReadDir(l.cfg.LogDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		if !strings.HasPrefix(name, "connectivity_") &&
			!strings.HasPrefix(name, "speedtest_") &&
			!strings.HasPrefix(name, "metrics_") &&
			!strings.HasPrefix(name, "notifications_") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(l.cfg.LogDir, name)); err != nil {
				l.AppLog("ERROR removing old log %s: %v", name, err)
			}
		}
	}
}

type remote struct {
	webhook, token, chat, ntfyURL, ntfyToken string
}

func (l *Logger) remoteTargets() remote {
	l.mu.Lock()
	defer l.mu.Unlock()
	return remote{l.cfg.WebhookURL, l.cfg.TelegramBotToken, l.cfg.TelegramChatID, l.cfg.NtfyURL, l.cfg.NtfyToken}
}

type message struct {
	kind     string
	title    string
	telegram string
	data     interface{}
	at       time.Time
	webhook  func(url string) interface{}
}

func (l *Logger) deliver(to types.Targets, m message) {
	r := l.remoteTargets()
	if to.Webhook && r.webhook != "" && IsSupportedWebhook(r.webhook) {
		payload := m.webhook
		if config.IsGenericWebhook(r.webhook) {
			go l.sendWebhook(r.webhook, GenericPayload(m.kind, m.title, PlainText(m.telegram), m.at, m.data))
		} else {
			go l.sendWebhook(r.webhook, payload(r.webhook))
		}
	}
	if to.Telegram && r.token != "" && r.chat != "" {
		go l.sendTelegram(r.token, r.chat, m.telegram)
	}
	if to.Ntfy && r.ntfyURL != "" && config.IsValidNtfyURL(r.ntfyURL) {
		go l.sendNtfy(r.ntfyURL, r.ntfyToken, m.kind, m.title, PlainBody(m.telegram))
	}
}

func (l *Logger) SendEvent(lang string, event monitor.Event, kind string, to types.Targets) {
	l.deliver(to, message{
		kind:     kind,
		title:    EventTitle(lang, kind),
		telegram: TelegramEventText(lang, event, kind),
		data:     event,
		at:       event.Timestamp,
		webhook:  func(url string) interface{} { return BuildEventPayload(lang, event, kind, url) },
	})
}

func (l *Logger) SendSpeed(lang string, event SpeedTestEvent, thresholdMbps float64, below bool, to types.Targets) {
	_, title, _ := speedTitle(lang, below)
	kind := "speed"
	if below {
		kind = "speed_drop"
	}
	l.deliver(to, message{
		kind:     kind,
		title:    title,
		telegram: TelegramSpeedText(lang, event, thresholdMbps, below),
		data:     event,
		at:       event.Timestamp,
		webhook:  func(url string) interface{} { return BuildSpeedResultPayload(lang, event, thresholdMbps, below, url) },
	})
}

func (l *Logger) SendWeekly(lang string, w types.WeeklySummary, to types.Targets) {
	_, title, _ := weeklyTitle(lang, w)
	l.deliver(to, message{
		kind:     "weekly",
		title:    title,
		telegram: TelegramWeeklyText(lang, w),
		data:     w,
		at:       w.To,
		webhook:  func(url string) interface{} { return BuildWeeklyPayload(lang, w, url) },
	})
}

var ntfyTags = map[string]string{
	"down":       "x",
	"degraded":   "warning",
	"restored":   "white_check_mark",
	"reminder":   "alarm_clock",
	"speed":      "rocket",
	"speed_drop": "warning",
	"weekly":     "bar_chart",
	"test":       "bell",
}

func ntfyPriority(kind string) string {
	switch kind {
	case "down", "reminder":
		return "high"
	case "weekly", "speed", "test":
		return "low"
	}
	return "default"
}

func NtfyRequest(url, token, kind, title, body string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", ntfyPriority(kind))
	if tag := ntfyTags[kind]; tag != "" {
		req.Header.Set("Tags", tag)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func (l *Logger) postNtfy(url, token, kind, title, body string) error {
	req, err := NtfyRequest(url, token, kind, title, body)
	if err != nil {
		return err
	}
	resp, err := NoRedirectClient(10 * time.Second).Do(req)
	if err != nil {
		return errors.New(RedactSecret(RedactSecret(err.Error(), token), url))
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("the ntfy server answered HTTP %d", resp.StatusCode)
	}
	return nil
}

func (l *Logger) sendNtfy(url, token, kind, title, body string) {
	defer func() {
		if r := recover(); r != nil {
			l.AppLog("PANIC recovered in ntfy send: %v", r)
		}
	}()
	if err := l.postNtfy(url, token, kind, title, body); err != nil {
		l.AppLog("NTFY %s", err)
	}
}

func (l *Logger) SendTestNtfy(url, token string) string {
	if !config.IsValidNtfyURL(url) {
		return "use the full topic address, like https://ntfy.sh/my-topic"
	}
	l.mu.Lock()
	lang := l.cfg.Language
	l.mu.Unlock()
	if err := l.postNtfy(url, token, "test", i18n.T(lang, "webhook.test.title", nil), i18n.T(lang, "webhook.test.body", nil)); err != nil {
		l.AppLog("NTFY TEST %s", err)
		return err.Error()
	}
	return ""
}

func (l *Logger) sendTelegram(token, chatID, text string) {
	defer func() {
		if r := recover(); r != nil {
			l.AppLog("PANIC recovered in telegram send: %v", r)
		}
	}()
	if token == "" || chatID == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	url := "https://api.telegram.org/bot" + token + "/sendMessage"
	client := NoRedirectClient(10 * time.Second)
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		l.AppLog("TELEGRAM send error: %s", RedactSecret(err.Error(), token))
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		l.AppLog("TELEGRAM HTTP %d", resp.StatusCode)
	}
}

func (l *Logger) sendWebhook(url string, payload interface{}) {
	defer func() {
		if r := recover(); r != nil {
			l.AppLog("PANIC recovered in webhook send: %v", r)
		}
	}()

	body, err := json.Marshal(payload)
	if err != nil {
		l.AppLog("WEBHOOK marshal error: %v", err)
		return
	}

	client := NoRedirectClient(10 * time.Second)
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		l.AppLog("WEBHOOK send error: %s", RedactSecret(err.Error(), url))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		l.AppLog("WEBHOOK HTTP %d", resp.StatusCode)
	} else {
		l.AppLog("WEBHOOK OK HTTP %d", resp.StatusCode)
	}
}

func (l *Logger) SendTestWebhook(url string) string {
	if !IsSupportedWebhook(url) {
		return "use an https address (http is allowed only for addresses on your own network)"
	}

	l.mu.Lock()
	lang := l.cfg.Language
	l.mu.Unlock()
	title, text := i18n.T(lang, "webhook.test.title", nil), i18n.T(lang, "webhook.test.body", nil)

	var payload interface{}
	if config.IsGenericWebhook(url) {
		payload = GenericPayload("test", title, text, time.Now(), nil)
	} else if IsDiscord(url) {
		payload = map[string]interface{}{
			"username": "Internet Monitor",
			"embeds": []map[string]interface{}{{
				"title":       title,
				"description": text,
				"color":       0x22C55E,
				"footer":      map[string]string{"text": "Internet Monitor"},
			}},
		}
	} else {
		payload = map[string]interface{}{
			"text": "*" + title + "* — " + text,
		}
	}

	body, _ := json.Marshal(payload)
	client := NoRedirectClient(10 * time.Second)
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		msg := RedactSecret(err.Error(), url)
		l.AppLog("WEBHOOK TEST error: %s", msg)
		return msg
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("the webhook service answered HTTP %d", resp.StatusCode)
		l.AppLog("WEBHOOK TEST %s", msg)
		return msg
	}
	l.AppLog("WEBHOOK TEST OK HTTP %d", resp.StatusCode)
	return ""
}

func RedactSecret(msg, secret string) string {
	if secret == "" {
		return msg
	}
	return strings.ReplaceAll(msg, secret, "<redacted>")
}

func NoRedirectClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
