package api

import (
	"context"

	"github.com/pplmx/aurora/internal/metrics"
)

// registerRuntimeGauges exposes the two numbers a dashboard needs from a
// running node: how far the ledger has grown and whether anything is parked in
// the outbox. Both are read at scrape time; a failed read drops the line for
// that scrape instead of reporting a stale value.
func registerRuntimeGauges(reg *metrics.Registry, pending func(context.Context) (int, error), blocks func() int) {
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
