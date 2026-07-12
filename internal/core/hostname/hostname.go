package hostname

import (
	"net/netip"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

const (
	maxDNSHostnameLength = 253
	maxDNSLabelLength    = 63
)

// NormalizeHostnameV1 returns the canonical A-label hostname used by domain
// bindings. It accepts a bare DNS name, IPv4 address, bracketed IPv6 literal,
// localhost, or a single-label internal hostname. Invalid input returns "".
func NormalizeHostnameV1(input string) string {
	value := strings.TrimSpace(input)
	if value == "" || strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.ContainsAny(value, "\\/?#@") {
		return ""
	}

	if strings.HasPrefix(value, "[") || strings.HasSuffix(value, "]") {
		if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
			return ""
		}
		return normalizeIPv6(value[1 : len(value)-1])
	}
	if strings.Contains(value, ":") {
		return ""
	}

	value = strings.TrimSuffix(value, ".")
	if value == "" || strings.HasSuffix(value, ".") {
		return ""
	}
	if isNumericHostname(value) {
		return normalizeIPv4(value)
	}
	return normalizeDNS(value)
}

// NormalizeURLHostnameV1 returns the canonical hostname from an HTTP(S) URL.
// Ports, paths, credentials, and fragments are intentionally not preserved.
func NormalizeURLHostnameV1(input string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return ""
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		return ""
	}
	if address, err := netip.ParseAddr(host); err == nil && address.Is6() {
		return address.String()
	}
	return NormalizeHostnameV1(host)
}

func normalizeIPv4(value string) string {
	address, err := netip.ParseAddr(value)
	if err != nil || !address.Is4() || address.String() != value {
		return ""
	}
	return address.String()
}

func normalizeIPv6(value string) string {
	address, err := netip.ParseAddr(value)
	if err != nil || !address.Is6() {
		return ""
	}
	return address.String()
}

func normalizeDNS(value string) string {
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil {
		return ""
	}
	ascii = strings.ToLower(strings.TrimSuffix(ascii, "."))
	if ascii == "" || strings.HasSuffix(ascii, ".") || len(ascii) > maxDNSHostnameLength {
		return ""
	}
	for _, label := range strings.Split(ascii, ".") {
		if len(label) == 0 || len(label) > maxDNSLabelLength || !isDNSLabel(label) {
			return ""
		}
	}
	return ascii
}

func isNumericHostname(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && char != '.' {
			return false
		}
	}
	return true
}

func isDNSLabel(label string) bool {
	if !isASCIIAlphaNum(label[0]) || !isASCIIAlphaNum(label[len(label)-1]) {
		return false
	}
	for _, char := range label {
		if !isASCIIAlphaNum(byte(char)) && char != '-' {
			return false
		}
	}
	return true
}

func isASCIIAlphaNum(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}
