package utils

import "fmt"

// FormatBytes renders a byte count for operators: plain bytes below a KiB and
// one decimal in KiB/MiB/GiB/TiB above it. Files are reported in powers of two
// by `du -h` and file managers, so a database size here matches what the
// operator sees elsewhere.
func FormatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 0 {
		// A negative count means the caller made a mistake, not a tiny file.
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	for i, unit := range units {
		value /= 1024
		if value < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return ""
}
