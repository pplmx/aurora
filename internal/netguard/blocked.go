// Package netguard centralizes the destination-address policy shared by the
// Oracle domain validator, redirect policy and guarded HTTP dialer.
package netguard

import (
	"net"
	"net/netip"
)

// blockedPrefixes are address ranges no Oracle source may reach, whether from
// an initial URL, a redirect, or the concrete address resolved at dial time:
// loopback, RFC1918 private, link-local (including cloud metadata), CGNAT,
// IPv6 loopback/link-local/unique-local, and the unspecified address. The
// remaining categories have no legitimate public feed endpoint: IETF
// assignments, benchmarking, reserved class E, multicast, NAT64/6to4
// translation of embedded IPv4, and documentation/discard-only space.
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	// Multicast is never a unicast HTTP feed destination. The range includes
	// the MCAST-TEST-NET allocation as well as local-scope groups.
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	// The low IPv6 range contains the unspecified/loopback addresses,
	// deprecated IPv4-compatible addresses, and the IPv4-translated block.
	// None is a routable public HTTP destination; the compatible/translated
	// forms could otherwise look like an IPv6 escape around the IPv4 policy.
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("::ffff:0:0:0/96"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	// Deprecated IPv6 site-local space remains routable on legacy networks and
	// has the same internal-service risk as RFC1918.
	netip.MustParsePrefix("fec0::/10"),
	netip.MustParsePrefix("ff00::/8"),
	// NAT64 can translate an embedded IPv4 address at the network edge:
	// [64:ff9b::7f00:1] becomes 127.0.0.1 on an IPv6-only network, which
	// would route around an IPv4-only block list. Refuse both the well-known
	// and local-use prefixes.
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	// 6to4 embeds an arbitrary IPv4 destination in its address, offering
	// another route around an IPv4-only guard.
	netip.MustParsePrefix("2002::/16"),
	// IETF protocol assignments cover Teredo, benchmarking, ORCHID, anycast
	// control services and other non-web allocations. None is a legitimate
	// public data-feed destination, so refuse the whole assignment as a single
	// fail-closed policy boundary rather than maintaining a fragile allowlist.
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("100::/64"),
}

// IsBlockedIP reports whether ip belongs to a destination class Oracle sources
// must not reach. An unparseable address is blocked rather than guessed at.
func IsBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
