package source

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// metricServerSource reads usage from the metrics.k8s.io API with one List
// per namespace.
type metricServerSource struct {
	metricsClient metricsclientset.Interface
}

func newMetricServerSource(cfg *rest.Config) (*metricServerSource, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rest config must not be nil for metrics-server source")
	}
	mc, err := metricsclientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build metrics clientset: %w", err)
	}
	return &metricServerSource{metricsClient: mc}, nil
}

func (m *metricServerSource) PodUsage(ctx context.Context, pods []corev1.Pod) (map[types.NamespacedName]Usage, error) {
	usage := map[types.NamespacedName]Usage{}
	for _, ns := range namespacesOf(pods) {
		list, err := m.metricsClient.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list pod metrics in namespace %q: %w", ns, err)
		}
		for _, podMetrics := range list.Items {
			var u Usage
			for _, container := range podMetrics.Containers {
				u.CPUmilli += container.Usage.Cpu().MilliValue()
				u.MemoryBytes += container.Usage.Memory().Value()
			}
			usage[types.NamespacedName{Namespace: podMetrics.Namespace, Name: podMetrics.Name}] = u
		}
	}
	return usage, nil
}

var _ Source = (*metricServerSource)(nil)
