package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormatBytes pins the boundaries an operator reads: plain bytes below a
// KiB, one decimal above it, and each unit promoted at 1024.
func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		0:               "0 B",
		512:             "512 B",
		1023:            "1023 B",
		1024:            "1.0 KiB",
		1536:            "1.5 KiB",
		1 << 20:         "1.0 MiB",
		5*(1<<30) + 512: "5.0 GiB",
		3 * (1 << 40):   "3.0 TiB",
		10 * (1 << 40):  "10.0 TiB",
	}
	for size, want := range cases {
		assert.Equalf(t, want, FormatBytes(size), "size %d", size)
	}
}
