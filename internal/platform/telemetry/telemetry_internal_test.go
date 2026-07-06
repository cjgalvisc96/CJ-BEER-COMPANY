package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// failingRegisterer makes the exporter's only error path reachable.
type failingRegisterer struct{}

func (failingRegisterer) Register(prometheus.Collector) error {
	return assert.AnError
}
func (failingRegisterer) MustRegister(...prometheus.Collector) {}
func (failingRegisterer) Unregister(prometheus.Collector) bool { return false }

func TestInitMetricsSurfacesRegistererFailures(t *testing.T) {
	_, err := initMetrics("svc", "test", failingRegisterer{}, prometheus.NewRegistry())

	assert.Error(t, err)
}
