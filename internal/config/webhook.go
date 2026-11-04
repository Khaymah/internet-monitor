package config

import (
	"net"
	"net/url"
	"strings"
)

var webhookHosts = map[string]string{
	"discord.com":        "/api/webhooks/",
	"discordapp.com":     "/api/webhooks/",
	"canary.discord.com": "/api/webhooks/",
	"ptb.discord.com":    "/api/webhooks/",
	"hooks.slack.com":    "/services/",
}

func parseWebhook(raw string) (*url.URL, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return nil, false
	}
	prefix, ok := webhookHosts[strings.ToLower(u.Hostname())]
	if !ok || !strings.HasPrefix(u.Path, prefix) || len(u.Path) <= len(prefix) {
		return nil, false
	}
	return u, true
}

func parseGeneric(raw string) (*url.URL, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" {
		return nil, false
	}
	switch u.Scheme {
	case "https":
		return u, true
	case "http":
		return u, isPrivateHost(u.Hostname())
	}
	return nil, false
}

func isPrivateHost(host string) bool {
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func IsSupportedWebhook(raw string) bool {
	if _, ok := parseWebhook(raw); ok {
		return true
	}
	_, ok := parseGeneric(raw)
	return ok
}

func IsGenericWebhook(raw string) bool {
	if _, ok := parseWebhook(raw); ok {
		return false
	}
	_, ok := parseGeneric(raw)
	return ok
}

func IsValidNtfyURL(raw string) bool {
	u, ok := parseGeneric(raw)
	return ok && strings.Trim(u.Path, "/") != "" && u.RawQuery == ""
}

func IsDiscordWebhook(raw string) bool {
	u, ok := parseWebhook(raw)
	return ok && strings.Contains(strings.ToLower(u.Hostname()), "discord")
}

func IsSlackWebhook(raw string) bool {
	u, ok := parseWebhook(raw)
	return ok && strings.EqualFold(u.Hostname(), "hooks.slack.com")
}
