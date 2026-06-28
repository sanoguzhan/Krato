package metrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type listPodsFn func(ctx context.Context, namespace string, selector *metav1.LabelSelector) ([]corev1.Pod, error)

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

type CollectorType struct {
	source   string
	listPods listPodsFn
	usageFn  podUsageFn
}

func (c *CollectorType) CollectMetrics(ctx context.Context, namespace string, selector *metav1.LabelSelector) (MetricResult, error) {
	_ = logf.FromContext(ctx)
	if selector == nil {
		return MetricResult{}, fmt.Errorf("selector must not be nil")
	}
	if c.usageFn == nil {
		return MetricResult{}, fmt.Errorf("collector usage function is not configured")
	}
	if c.listPods == nil {
		return MetricResult{}, fmt.Errorf("collector pod listing function is not configured")
	}

	pods, err := c.listPods(ctx, namespace, selector)
	if err != nil {
		return MetricResult{}, fmt.Errorf("failed to retrieve selected pods: %w", err)
	}

	aggregated, err := aggregatePodMetrics(ctx, pods, c.usageFn)
	if err != nil {
		logf.FromContext(ctx).Error(err, "Failed to aggregate pod request metrics")
		return MetricResult{}, fmt.Errorf("failed to aggregate pod metrics: %w", err)
	}

	aggregated.Namespace = namespace
	aggregated.Selector = selector.DeepCopy()
	aggregated.Source = c.source
	aggregated.CollectedAt = metav1.Now()

	return aggregated, nil
}
