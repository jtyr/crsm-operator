package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/gomega"
)

func TestCRSMTotal(t *testing.T) {
	// Initiate Gomega
	g := NewWithT(t)

	// Create a custom registry
	registry := prometheus.NewRegistry()
	recorder := newPrometheusMetricsRecorderWithRegistry(registry)

	// Test incrementation and decrementation of the gauge value
	g.Expect(testutil.ToFloat64(recorder.crsmTotal.WithLabelValues())).To(Equal(0.0), "Test crsmTotal initial:")
	recorder.IncCRSMTotal()
	g.Expect(testutil.ToFloat64(recorder.crsmTotal.WithLabelValues())).To(Equal(1.0), "Test crsmTotal increment 1:")
	recorder.IncCRSMTotal()
	g.Expect(testutil.ToFloat64(recorder.crsmTotal.WithLabelValues())).To(Equal(2.0), "Test crsmTotal increment 2:")
	recorder.DecCRSMTotal()
	g.Expect(testutil.ToFloat64(recorder.crsmTotal.WithLabelValues())).To(Equal(1.0), "Test crsmTotal decrement 1:")
	recorder.DecCRSMTotal()
	g.Expect(testutil.ToFloat64(recorder.crsmTotal.WithLabelValues())).To(Equal(0.0), "Test crsmTotal decrement 2:")
}
