package netguard

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"0.0.0.0", "10.1.2.3", "100.64.1.1", "127.0.0.1", "169.254.169.254",
		"172.16.0.1", "192.168.1.1", "192.0.0.1", "198.18.0.1", "240.0.0.1",
		"255.255.255.255",
		"::", "::1", "fc00::1", "fe80::1", "ff02::1",
		"64:ff9b::7f00:1", "64:ff9b:1::7f00:1", "2001:db8::1", "100::1",
	}
	for _, text := range blocked {
		ip := net.ParseIP(text)
		require.NotNilf(t, ip, "test address %q does not parse", text)
		require.Truef(t, IsBlockedIP(ip), "IsBlockedIP(%s) = false, want blocked", text)
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111"}
	for _, text := range allowed {
		require.Falsef(t, IsBlockedIP(net.ParseIP(text)), "IsBlockedIP(%s) = true, want allowed", text)
	}

	require.True(t, IsBlockedIP(net.IP{1, 2, 3}), "an unparseable address must fail closed")
}
