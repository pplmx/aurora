package logger

import (
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// TestInitIsSafeWhileLogging pins the race the detector found during the
// desktop work: Init replaced the package logger (and zerolog's global time
// format) while the oracle scheduler, the outbox drainer and request handlers
// were already logging. Re-initializing under that traffic must be safe - a
// desktop start and a test that calls Init again both do it.
func TestInitIsSafeWhileLogging(t *testing.T) {
	viper.Set("log.level", "info")
	viper.Set("log.path", "")
	t.Cleanup(viper.Reset)
	t.Cleanup(Close)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); Init() }()
		go func() { defer wg.Done(); Info().Msg("probe") }()
	}
	wg.Wait()
}
