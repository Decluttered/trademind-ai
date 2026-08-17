package collectdomain

import (
	"net/url"
	"strings"
)

// PlatformID identifies a dedicated collect provider platform from URL hostname.
type PlatformID string

const (
	PlatformAmazonDE PlatformID = "amazon.de"
)

// HostnameFromURL returns lowercase hostname or empty when invalid.
func HostnameFromURL(urlStr string) string {
	s := strings.TrimSpace(urlStr)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(u.Hostname()))
}

func hostMatchesAmazonDE(host string) bool {
	return host == "amazon.de" || strings.HasSuffix(host, ".amazon.de")
}

// DetectPlatform maps a hostname to a dedicated platform id when recognized.
func DetectPlatform(hostname string) (PlatformID, bool) {
	host := strings.ToLower(strings.TrimSpace(hostname))
	if host == "" {
		return "", false
	}
	if hostMatchesAmazonDE(host) {
		return PlatformAmazonDE, true
	}
	return "", false
}

// ProviderSourceForPlatform maps platform id to collect task source key.
func ProviderSourceForPlatform(p PlatformID) string {
	return string(p)
}
