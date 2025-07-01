package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type MetricsRecorder interface {
	// IncRSMTotal increments the total number of CRSM resources available on the cluster.
	IncCRSMTotal()

	// DecRSMTotal decrements the total number of CRSM resources available on the cluster.
	DecCRSMTotal()
}

type PrometheusMetricsRecorder struct {
	crsmTotal *prometheus.GaugeVec
}

// NewPrometheusMetricsRecorder creates a new PrometheusMetricsRecorder and registers metrics.
func NewPrometheusMetricsRecorder() *PrometheusMetricsRecorder {
	return newPrometheusMetricsRecorderWithRegistry(metrics.Registry)
}

// newPrometheusMetricsRecorderWithRegistry creates a new PrometheusMetricsRecorder with a custom registry.
func newPrometheusMetricsRecorderWithRegistry(registry prometheus.Registerer) *PrometheusMetricsRecorder {
	recorder := &PrometheusMetricsRecorder{
		crsmTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "crsm_total",
				Help: "Total number of CRSM resources available on the cluster.",
			},
			[]string{},
		),
	}

	// Register metrics with the provided registry
	registry.MustRegister(
		recorder.crsmTotal,
	)

	return recorder
}

// IncCRSMTotal increments the total number of CRSM resources available on the cluster.
func (r *PrometheusMetricsRecorder) IncCRSMTotal() {
	r.crsmTotal.WithLabelValues().Inc()
}

// DecCRSMTotal decrements the total number of CRSM resources available on the cluster.
func (r *PrometheusMetricsRecorder) DecCRSMTotal() {
	r.crsmTotal.WithLabelValues().Dec()
}
