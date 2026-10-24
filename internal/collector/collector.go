// Package collector periodically collects usage for every ResourceAttribution
// and publishes it through the metrics package.
package collector

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
	"github.com/sanoguzhan/krato/internal/metrics"
	"github.com/sanoguzhan/krato/internal/source"
)

// Loop collects usage for all ResourceAttributions every Interval. It is a
// manager.Runnable; since it does not implement LeaderElectionRunnable, the
// manager runs it on the leader only.
type Loop struct {
	Client     client.Client
	RestConfig *rest.Config
	Interval   time.Duration

	// recorded holds the attributions whose series were set on the last pass.
	recorded map[types.NamespacedName]struct{}
}

// Start runs a collection pass immediately and then every Interval, until ctx is cancelled.
func (l *Loop) Start(ctx context.Context) error {
	ctx = logf.IntoContext(ctx, logf.Log.WithName("collector"))
	wait.UntilWithContext(ctx, l.collect, l.Interval)
	return nil
}

func (l *Loop) collect(ctx context.Context) {
	log := logf.FromContext(ctx)

	var list attributionv1alpha1.ResourceAttributionList
	if err := l.Client.List(ctx, &list); err != nil {
		log.Error(err, "Failed to list ResourceAttributions")
		return
	}

	// Group attributions by backend so each backend is queried once per pass.
	groups := map[source.Config][]*attributionv1alpha1.ResourceAttribution{}
	for i := range list.Items {
		ra := &list.Items[i]
		if ra.Spec.Selector == nil {
			continue
		}
		cfg := source.ConfigFor(ra)
		groups[cfg] = append(groups[cfg], ra)
	}

	recorded := map[types.NamespacedName]struct{}{}
	for cfg, attributions := range groups {
		if err := l.collectGroup(ctx, cfg, attributions, recorded); err != nil {
			log.Error(err, "Failed to collect usage", "backend", cfg.Backend, "endpoint", cfg.Endpoint)
		}
	}

	// Drop the series of attributions that were deleted or failed on this pass,
	// so Prometheus does not keep scraping stale values.
	for key := range l.recorded {
		if _, ok := recorded[key]; !ok {
			metrics.Delete(key.Namespace, key.Name)
		}
	}
	l.recorded = recorded
}

// collectGroup collects usage for attributions that share a backend, with a
// single PodUsage call for all their pods.
func (l *Loop) collectGroup(ctx context.Context, cfg source.Config, attributions []*attributionv1alpha1.ResourceAttribution, recorded map[types.NamespacedName]struct{}) error {
	log := logf.FromContext(ctx)

	src, err := source.New(cfg, l.RestConfig)
	if err != nil {
		return err
	}

	selected := make(map[*attributionv1alpha1.ResourceAttribution][]corev1.Pod, len(attributions))
	var allPods []corev1.Pod
	for _, ra := range attributions {
		pods, err := l.selectPods(ctx, ra)
		if err != nil {
			log.Error(err, "Failed to select pods for ResourceAttribution", "namespace", ra.Namespace, "name", ra.Name)
			continue
		}
		selected[ra] = pods
		allPods = append(allPods, pods...)
	}

	usage, err := src.PodUsage(ctx, allPods)
	if err != nil {
		return err
	}

	for ra, pods := range selected {
		var total source.Usage
		for _, pod := range pods {
			u := usage[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}]
			total.CPUmilli += u.CPUmilli
			total.MemoryBytes += u.MemoryBytes
		}
		metrics.Record(ra.Namespace, ra.Name, total.CPUmilli, total.MemoryBytes, len(pods))
		recorded[types.NamespacedName{Namespace: ra.Namespace, Name: ra.Name}] = struct{}{}
	}
	return nil
}

// selectPods lists the pods an attribution selects, skipping pods that have finished.
func (l *Loop) selectPods(ctx context.Context, ra *attributionv1alpha1.ResourceAttribution) ([]corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(ra.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %w", err)
	}

	var list corev1.PodList
	if err := l.Client.List(ctx, &list,
		client.InNamespace(ra.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	pods := list.Items[:0]
	for _, pod := range list.Items {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}
