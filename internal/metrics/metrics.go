// Package metrics holds the Prometheus gauges the operator exports for each
// ResourceAttribution. They are served on the manager's /metrics endpoint.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var labels = []string{"namespace", "attribution"}

var (
	cpuCores = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "krato_attribution_cpu_cores",
		Help: "CPU of the pods selected by a ResourceAttribution, in cores, as reported by its metrics backend.",
	}, labels)
	memoryBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "krato_attribution_memory_bytes",
		Help: "Memory of the pods selected by a ResourceAttribution, in bytes, as reported by its metrics backend.",
	}, labels)
	pods = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "krato_attribution_pods",
		Help: "Number of running or pending pods selected by a ResourceAttribution.",
	}, labels)
)

func init() {
	crmetrics.Registry.MustRegister(cpuCores, memoryBytes, pods)
}

// Record sets the gauges of one attribution.
func Record(namespace, name string, cpuMilli, memBytes int64, podCount int) {
	cpuCores.WithLabelValues(namespace, name).Set(float64(cpuMilli) / 1000)
	memoryBytes.WithLabelValues(namespace, name).Set(float64(memBytes))
	pods.WithLabelValues(namespace, name).Set(float64(podCount))
}

// Delete removes the series of one attribution.
func Delete(namespace, name string) {
	cpuCores.DeleteLabelValues(namespace, name)
	memoryBytes.DeleteLabelValues(namespace, name)
	pods.DeleteLabelValues(namespace, name)
}
