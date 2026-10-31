package netguard

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsBlockedHost covers the literal fast path: a URL host that is already an
// address, with or without a port or IPv6 brackets, is judged without a lookup.
func TestIsBlockedHost(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.0.0.1:8080", "[::1]", "[::1]:8080",
		"169.254.169.254", "10.0.0.5", "172.16.0.1", "192.168.1.1",
		"100.64.0.1", "0.0.0.0", "203.0.113.1", "224.0.0.1",
		"[2002:7f00:1::]",
	}
	for _, host := range blocked {
		require.Truef(t, IsBlockedHost(host), "IsBlockedHost(%q) = false, want blocked", host)
	}

	allowed := []string{"8.8.8.8", "[2606:4700:4700::1111]", "93.184.216.34", ""}
	for _, host := range allowed {
		require.Falsef(t, IsBlockedHost(host), "IsBlockedHost(%q) = true, want allowed", host)
	}

	// A hostname is not an address: resolving it belongs to HostBlocked, and
	// this fast path must not pretend to have decided.
	require.False(t, IsBlockedHost("example.com"), "a name is not judged by the literal path")
}

// TestHostBlockedResolvesNames proves a redirect target given as a name is
// resolved and refused when it lands in blocked space. localhost is the
// canonical SSRF target and would slip through an address-only check.
func TestHostBlockedResolvesNames(t *testing.T) {
	blocked, err := HostBlocked("localhost:8080")
	require.NoError(t, err)
	require.True(t, blocked, "localhost must resolve to a blocked address")

	blocked, err = HostBlocked("127.0.0.1:9")
	require.NoError(t, err)
	require.True(t, blocked)

	blocked, err = HostBlocked("8.8.8.8")
	require.NoError(t, err)
	require.False(t, blocked)

	blocked, err = HostBlocked("[2606:4700:4700::1111]")
	require.NoError(t, err)
	require.False(t, blocked, "a public IPv6 literal is allowed")

	// A name that cannot be resolved is reported as an error, so the caller's
	// policy refuses the redirect instead of following an unverified host.
	blocked, err = HostBlocked("does-not-exist-aurora.invalid")
	require.Error(t, err)
	require.False(t, blocked)

	// A malformed bracketed host is refused rather than parsed loosely.
	_, err = HostBlocked("[not-an-address]")
	require.Error(t, err)
}

// TestValidateDialAddress covers the check the dialer runs immediately before
// connecting, where a name would already have been resolved by the transport.
func TestValidateDialAddress(t *testing.T) {
	blockedAddresses := []string{
		"127.0.0.1:80", "10.0.0.5:443", "169.254.169.254:80",
		"[::1]:443", "[fe80::1%eth0]:443", "[fec0::1]:443", "[::127.0.0.1]:80",
		// Unparseable input fails closed: an address the policy cannot read
		// must not be dialed.
		"not-an-address", "unresolved.example:443",
	}
	for _, address := range blockedAddresses {
		err := ValidateDialAddress(address)
		require.Truef(t, errors.Is(err, ErrBlockedDestination),
			"ValidateDialAddress(%q) = %v, want ErrBlockedDestination", address, err)
	}

	allowedAddresses := []string{
		"8.8.8.8:53", "[::ffff:8.8.8.8]:443", "[2606:4700:4700::1111]:443",
	}
	for _, address := range allowedAddresses {
		require.NoErrorf(t, ValidateDialAddress(address),
			"ValidateDialAddress(%q) must allow a public address", address)
	}
}

// TestHostWithoutPort keeps the trimming helper honest: it accepts the forms a
// URL host and a dial address actually take, and leaves anything else for the
// caller's real validation.
func TestHostWithoutPort(t *testing.T) {
	for in, want := range map[string]string{
		"example.com:8080": "example.com",
		"example.com":      "example.com",
		"[::1]:443":        "::1",
		"[::1]":            "[::1]",
		"  example.com  ":  "example.com",
		"":                 "",
	} {
		require.Equalf(t, want, HostWithoutPort(in), "HostWithoutPort(%q)", in)
	}
}
