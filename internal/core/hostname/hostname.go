package hostname

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

const (
	maxDNSHostnameLength = 253
	maxDNSLabelLength    = 63
	maxPageURLLength     = 2048
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

// NormalizePageURLV1 returns the canonical address of a page with whatever
// follows '#' removed. A fragment is the reader's position inside one page, not
// a different page, and it is the one part of an address that is routinely a
// private note to the browser. Credentials and a default port are dropped; the
// path and the query are kept, because a domain alone cannot tell configuring a
// site in its dashboard from reading its public pages.
//
// The result matches what the extension produces byte for byte, so a recorded
// address is the one that was sent.
func NormalizePageURLV1(input string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return ""
	}
	// Cut the fragment off the text, not off the parse: url.ParseRequestURI
	// leaves '#' and everything after it inside the query. A '#' that is part
	// of a query is percent-encoded by the time an address is written down.
	if index := strings.IndexByte(value, '#'); index >= 0 {
		value = value[:index]
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	host := NormalizeURLHostnameV1(value)
	if host == "" {
		return ""
	}
	authority := host
	if strings.Contains(host, ":") {
		authority = "[" + host + "]"
	}
	if port := parsed.Port(); port != "" && port != defaultPort(parsed.Scheme) {
		authority += ":" + port
	}
	origin := parsed.Scheme + "://" + authority
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	full := origin + path
	if query := escapeQueryV1(parsed.RawQuery); query != "" {
		full += "?" + query
	}
	if len(full) <= maxPageURLLength {
		return full
	}
	// Too long to keep whole. A truncated address would name a page that does
	// not exist, so what is dropped is dropped entirely.
	withoutQuery := origin + path
	if len(withoutQuery) <= maxPageURLLength {
		return withoutQuery
	}
	return origin + "/"
}

func defaultPort(scheme string) string {
	if scheme == "https" {
		return "443"
	}
	return "80"
}

// escapeQueryV1 applies the percent-encoding a browser applies to the query of
// an http(s) URL, so both ends of the contract spell the same address.
func escapeQueryV1(raw string) string {
	var out strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c <= 0x20 || c > 0x7e || c == '"' || c == '#' || c == '<' || c == '>' || c == '\'' {
			out.WriteString(fmt.Sprintf("%%%02X", c))
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
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
