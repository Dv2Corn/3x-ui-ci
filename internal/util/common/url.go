package common

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

var hostProfile = idna.New(idna.MapForLookup(), idna.StrictDomainName(false))

// NormalizeHost reduces a host, with or without a port, to the form browsers send
// in Host: lower case, IDN as punycode, no IPv6 brackets or root's trailing dot.
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.Trim(host, "[]"), ".")
	if ascii, err := hostProfile.ToASCII(host); err == nil {
		host = ascii
	}
	return strings.ToLower(host)
}

// EnsureURLScheme prepends https:// to a URL that carries no scheme, so
// subscription apps and browsers don't resolve it relative to the panel's own
// domain (e.g. "t.me/support" turning into "https://panel.example/t.me/support").
// Values with an explicit scheme (https://, tg://, mailto:, tel:) and empty
// strings pass through untouched.
func EnsureURLScheme(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Schemes are case-insensitive (RFC 3986), and phone keyboards often
	// capitalize the first letter ("Mailto:", "Tel:").
	lower := strings.ToLower(trimmed)
	if strings.Contains(trimmed, "://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") {
		return trimmed
	}
	return "https://" + trimmed
}

// ParseRemoteRoutingURL classifies a routing settings value: one single-line
// absolute HTTPS URL is a remote source (canonicalized); anything else is inline.
func ParseRemoteRoutingURL(raw string) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.ContainsAny(trimmed, "\r\n") {
		return "", false, nil
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		return "", false, nil
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return "", true, errors.New("must be an absolute HTTPS URL")
	}
	if u.User != nil {
		return "", true, errors.New("must not contain URL credentials")
	}
	u.Scheme = "https"
	u.Fragment = ""
	return u.String(), true, nil
}
