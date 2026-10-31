package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// The destination policy a host has to pass before a request leaves the
// process. IsBlockedIP answers it for a concrete address; the helpers below
// answer it for the two forms a request actually carries — a URL host that may
// be a name, and the address the dialer is about to connect to.
//
// They live here rather than beside the HTTP client because the redirect
// policy, the dial guard and the Oracle source validator are three call sites
// for one policy, and keeping the policy in one package is what lets a reader
// see the whole block list next to the checks that apply it.

// ErrBlockedDestination marks a destination the policy refuses. Callers wrap it
// so errors.Is can distinguish a refused destination from a failed connection.
var ErrBlockedDestination = errors.New("blocked destination")

// resolveTimeout bounds the DNS lookup a redirect check performs. A poisoned or
// stuck resolver must not hold a fetch open.
const resolveTimeout = 2 * time.Second

// HostWithoutPort strips an optional port from a URL host. It tolerates input
// that SplitHostPort rejects, so a malformed host reaches the caller's real
// validation instead of being silently dropped here.
func HostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// ParseIPLiteral returns the address a URL host names, or nil when the host is
// a name rather than a literal. Bracketed IPv6 hosts are accepted with or
// without their brackets.
func ParseIPLiteral(host string) net.IP {
	host = HostWithoutPort(host)
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return net.ParseIP(host[1 : len(host)-1])
	}
	return nil
}

// IsBlockedHost reports whether host (optionally host:port) is an IP literal
// inside a blocked range. A hostname is not resolved here: use HostBlocked for
// that, because this fast path must stay allocation-cheap on the dial path.
func IsBlockedHost(host string) bool {
	ip := ParseIPLiteral(host)
	if ip == nil {
		return false
	}
	return IsBlockedIP(ip)
}

// ValidateDialAddress is the last line of defense. The URL-level checks resolve
// a hostname before the request, but net/http resolves it again when dialing,
// which leaves a DNS-rebinding window between the two lookups. The dialer's
// control hook sees the concrete address immediately before connect, so a
// public answer that later resolves to a private address is still refused.
//
// An address that cannot be parsed is refused rather than guessed at.
func ValidateDialAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: invalid dial address %q", ErrBlockedDestination, address)
	}
	// A zone suffix ("fe80::1%eth0") is local routing information, not part of
	// the address the policy judges.
	if i := strings.LastIndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("%w: unresolved dial address %q", ErrBlockedDestination, address)
	}
	if IsBlockedIP(ip) {
		return fmt.Errorf("%w: dial target %q", ErrBlockedDestination, address)
	}
	return nil
}

// HostBlocked decides whether a redirect target host is out of bounds. It
// handles IP literals directly and resolves hostnames, so a redirect aimed at
// localhost, a cloud-metadata name or another internal DNS entry is refused
// just like a literal private address. An unresolved or invalid host is
// reported as an error by the caller's policy, never followed: a destination
// that cannot be verified must not be reached.
func HostBlocked(host string) (bool, error) {
	host = HostWithoutPort(host)
	if ip := ParseIPLiteral(host); ip != nil {
		return IsBlockedIP(ip), nil
	}
	if strings.ContainsAny(host, "[]") {
		return false, fmt.Errorf("invalid bracketed host %q", host)
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, fmt.Errorf("resolve %q: %w", host, err)
	}
	for _, a := range addrs {
		if IsBlockedIP(a.IP) {
			return true, nil
		}
	}
	return false, nil
}
