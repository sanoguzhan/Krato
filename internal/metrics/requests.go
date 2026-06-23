package metrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type RequestCollector struct {
	client client.Client
}

func NewRequestCollector(k8sClient client.Client) *RequestCollector {
	return &RequestCollector{client: k8sClient}
}

func (r *RequestCollector) CollectMetrics(ctx context.Context, namespace string, selector *metav1.LabelSelector) (MetricResult, error) {
	_ = logf.FromContext(ctx)
	if selector == nil {
		return MetricResult{}, fmt.Errorf("selector must not be nil")
	}

	selectorExpr, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return MetricResult{}, fmt.Errorf("failed to build label selector: %w", err)
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selectorExpr},
	}

	if err := r.client.List(ctx, podList, listOpts...); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to list pods for metrics collection")
		return MetricResult{}, fmt.Errorf("failed to list pods: %w", err)
	}

	aggregated, err := aggregatePodMetrics(ctx, podList.Items, r.GetPodUsage)
	if err != nil {
		logf.FromContext(ctx).Error(err, "Failed to aggregate pod request metrics")
		return MetricResult{}, fmt.Errorf("failed to aggregate pod metrics: %w", err)
	}

	aggregated.Namespace = namespace
	aggregated.Selector = selector.DeepCopy()
	aggregated.Source = "request"
	aggregated.CollectedAt = metav1.Now()

	return aggregated, nil
}

func (r *RequestCollector) GetPodUsage(ctx context.Context, pod corev1.Pod) (MetricResult, error) {
	_ = ctx

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return MetricResult{}, nil
	}

	podMetrics := MetricResult{}
	for _, container := range pod.Spec.Containers {
		podMetrics.CPUmilli += container.Resources.Requests.Cpu().MilliValue()
		podMetrics.MemoryBytes += container.Resources.Requests.Memory().Value()
	}
	podMetrics.PodCount = 1
	return podMetrics, nil
}

var _ MetricsCollector = (*RequestCollector)(nil)
