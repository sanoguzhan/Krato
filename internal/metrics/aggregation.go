package metrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type podUsageFn func(ctx context.Context, pod corev1.Pod) (MetricResult, error)

func aggregatePodMetrics(ctx context.Context, pods []corev1.Pod, getPodUsage podUsageFn) (MetricResult, error) {
	var totalCpu int64
	var totalMemory int64
	var totalPodCount int32

	for _, pod := range pods {
		podMetrics, err := getPodUsage(ctx, pod)
		if err != nil {
			return MetricResult{}, fmt.Errorf("failed to get pod usage for aggregation: %w", err)
		}
		totalCpu += podMetrics.CPUmilli
		totalMemory += podMetrics.MemoryBytes
		totalPodCount += podMetrics.PodCount
	}

	return MetricResult{
		PodCount:    totalPodCount,
		CPUmilli:    totalCpu,
		MemoryBytes: totalMemory,
	}, nil

}
