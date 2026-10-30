package api

import (
	"context"

	"github.com/pplmx/aurora/internal/metrics"
)

// registerOutboxGauge exposes the audit outbox backlog: the one queue an
// operator acts on, because a parked event means a delivery is failing while
// the ledger keeps recording. A count error omits the gauge for that scrape
// instead of reporting a stale zero.
func registerOutboxGauge(reg *metrics.Registry, pending func(context.Context) (int, error)) {
	reg.RegisterGauge("aurora_outbox_pending_events",
		"Audit events parked in the outbox, waiting for a delivery retry.",
		func() (float64, bool) {
			count, err := pending(context.Background())
			if err != nil {
				return 0, false
			}
			return float64(count), true
		})
}
