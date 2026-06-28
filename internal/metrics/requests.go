package metrics

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RequestCollector struct {
	*CollectorType
}

func NewRequestCollector(k8sClient client.Client) *RequestCollector {
	r := &RequestCollector{
		CollectorType: &CollectorType{
			source: "request",
			listPods: func(ctx context.Context, namespace string, selector *metav1.LabelSelector) ([]corev1.Pod, error) {
				return retrieveSelectedPods(ctx, k8sClient, namespace, selector)
			},
		},
	}
	r.usageFn = r.GetPodUsage
	return r
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
