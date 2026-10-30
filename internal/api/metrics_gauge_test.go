package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pplmx/aurora/internal/metrics"
)

// TestOutboxGaugeReportsThePendingCount pins the exposed queue depth and the
// failure rule: a count error drops the line for that scrape instead of writing
// zero over a real backlog.
func TestOutboxGaugeReportsThePendingCount(t *testing.T) {
	reg := metrics.NewRegistry()
	registerOutboxGauge(reg, func(context.Context) (int, error) { return 4, nil })
	out := reg.Expose()
	assert.Contains(t, out, "# TYPE aurora_outbox_pending_events gauge\n")
	assert.Contains(t, out, "\naurora_outbox_pending_events 4\n")

	failing := metrics.NewRegistry()
	registerOutboxGauge(failing, func(context.Context) (int, error) {
		return 0, errors.New("database is busy")
	})
	assert.NotContains(t, failing.Expose(), "aurora_outbox_pending_events")
}
