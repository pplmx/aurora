package api

import (
	"context"
	"time"

	"github.com/pplmx/aurora/internal/metrics"
)

// registerRuntimeGauges exposes what a dashboard needs from a running node: how
// far the ledger has grown, whether anything is parked in the outbox, and how
// long the oldest parked event has waited. A count may be normal on a busy node;
// the age is what says the drainer is stuck. All are read at scrape time, and a
// failed read drops the line for that scrape instead of reporting a stale value.
func registerRuntimeGauges(
	reg *metrics.Registry,
	pending func(context.Context) (int, error),
	blocks func() int,
	oldest func(context.Context) (time.Time, bool, error),
) {
	reg.RegisterGauge("aurora_outbox_pending_events",
		"Audit events parked in the outbox, waiting for a delivery retry.",
		func() (float64, bool) {
			count, err := pending(context.Background())
			if err != nil {
				return 0, false
			}
			return float64(count), true
		})

	reg.RegisterGauge("aurora_ledger_blocks",
		"Blocks on the local ledger, genesis included.",
		func() (float64, bool) {
			if blocks == nil {
				return 0, false
			}
			return float64(blocks()), true
		})

	if oldest == nil {
		return
	}
	reg.RegisterGauge("aurora_outbox_oldest_pending_seconds",
		"Age of the oldest audit event parked in the outbox, in seconds.",
		func() (float64, bool) {
			at, ok, err := oldest(context.Background())
			if err != nil || !ok {
				return 0, false
			}
			age := time.Since(at).Seconds()
			if age < 0 {
				// A clock that stepped backwards is not a negative age.
				age = 0
			}
			return age, true
		})
}

// RegisterBuildInfo exposes the binary's identity as aurora_build_info, the
// Prometheus convention for "which build is this instance": the labels carry
// the version and toolchain, and the build time when the linker injected one.
func (s *Server) RegisterBuildInfo(version, buildTime, goVersion string) {
	labels := map[string]string{"version": version, "go": goVersion}
	if buildTime != "" && buildTime != "unknown" {
		labels["build_time"] = buildTime
	}
	s.metrics.RegisterConstantGauge("aurora_build_info", "Build identity of this binary.", labels, 1)
}
