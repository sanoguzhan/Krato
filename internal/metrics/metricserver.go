package metrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricServerCollector struct {
	*CollectorType
	metricsClient metricsclientset.Interface
}

func NewMetricServerCollector(k8sClient client.Client, cfg *rest.Config) (*MetricServerCollector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rest config must not be nil for metrics-server collector")
	}
	mc, err := metricsclientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build metrics clientset: %w", err)
	}

	m := &MetricServerCollector{
		metricsClient: mc,
		CollectorType: &CollectorType{
			source: "metrics-server",
			listPods: func(ctx context.Context, namespace string, selector *metav1.LabelSelector) ([]corev1.Pod, error) {
				return retrieveSelectedPods(ctx, k8sClient, namespace, selector)
			},
		},
	}
	m.usageFn = m.getPodUsage
	return m, nil
}

func (m *MetricServerCollector) getPodUsage(ctx context.Context, pod corev1.Pod) (MetricResult, error) {
	podMetrics, err := m.metricsClient.MetricsV1beta1().
		PodMetricses(pod.Namespace).
		Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		return MetricResult{}, fmt.Errorf("failed to get pod metrics for %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	usage := MetricResult{PodCount: 1}
	for _, container := range podMetrics.Containers {
		usage.CPUmilli += container.Usage.Cpu().MilliValue()
		usage.MemoryBytes += container.Usage.Memory().Value()
	}
	return usage, nil
}

var _ MetricsCollector = (*MetricServerCollector)(nil)
