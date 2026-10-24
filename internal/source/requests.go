package source

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// requestSource reports the resource requests from each pod's spec. It reads
// nothing remote, so it needs no client.
type requestSource struct{}

func (requestSource) PodUsage(_ context.Context, pods []corev1.Pod) (map[types.NamespacedName]Usage, error) {
	usage := make(map[types.NamespacedName]Usage, len(pods))
	for _, pod := range pods {
		var u Usage
		for _, container := range pod.Spec.Containers {
			u.CPUmilli += container.Resources.Requests.Cpu().MilliValue()
			u.MemoryBytes += container.Resources.Requests.Memory().Value()
		}
		usage[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = u
	}
	return usage, nil
}

var _ Source = requestSource{}
