package metrics

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetricsCollector interface {
	// CollectMetrics collects resource usage metrics for the given namespace and label selector.
	CollectMetrics(ctx context.Context, namespace string, selector *metav1.LabelSelector) (MetricResult, error)

	// GetPodUsage retrieves the resource usage metrics for a specific pod.
	GetPodUsage(ctx context.Context, pod corev1.Pod) (MetricResult, error)
}

type MetricResult struct {
	Namespace   string
	Selector    *metav1.LabelSelector
	PodCount    int32
	CPUmilli    int64
	MemoryBytes int64
	Source      string // request | prometheus | custom
	CollectedAt metav1.Time
}
