package metrics

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func retrieveSelectedPods(ctx context.Context, k8sClient client.Client, namespace string, selector *metav1.LabelSelector) ([]corev1.Pod, error) {
	selectorExpr, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to build label selector: %w", err)
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selectorExpr},
	}

	if err := k8sClient.List(ctx, podList, listOpts...); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to list pods for metrics collection")
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return podList.Items, nil
}
