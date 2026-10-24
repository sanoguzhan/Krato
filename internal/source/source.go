package source

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
)

// Backend is the value of spec.metricsBackend.
type Backend string

const (
	BackendRequests      Backend = "request"
	BackendMetricsServer Backend = "metrics-server"
	BackendPrometheus    Backend = "prometheus"
	BackendCustom        Backend = "custom"
)

// Usage is the CPU and memory of a single pod.
type Usage struct {
	CPUmilli    int64
	MemoryBytes int64
}

// Source reads pod usage in bulk: one call covers all the given pods, so the
// number of backend requests does not grow with the number of pods.
type Source interface {
	// PodUsage returns usage keyed by pod. Pods the backend has no data for are
	// missing from the result, and it may contain pods that were not asked for.
	PodUsage(ctx context.Context, pods []corev1.Pod) (map[types.NamespacedName]Usage, error)
}

// Config identifies a metrics backend. It is comparable, so attributions that
// share a backend can be grouped and queried together.
type Config struct {
	Backend  Backend
	Endpoint string
}

// ConfigFor resolves the metrics backend of an attribution. The spec takes
// precedence over the status; with neither set, pod requests are used.
func ConfigFor(ra *attributionv1alpha1.ResourceAttribution) Config {
	backend := strings.ToLower(strings.TrimSpace(ptrValue(ra.Spec.MetricsBackend)))
	endpoint := strings.TrimSpace(ptrValue(ra.Spec.MetricsEndpoint))

	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(ptrValue(ra.Status.MetricsServerBackend)))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(ptrValue(ra.Status.MetricsServerEndpoint))
	}

	if backend == "" {
		return Config{Backend: BackendRequests}
	}

	return Config{Backend: Backend(backend), Endpoint: endpoint}
}

// New returns the Source for cfg.
func New(cfg Config, restConfig *rest.Config) (Source, error) {
	switch cfg.Backend {
	case BackendRequests:
		return requestSource{}, nil
	case BackendMetricsServer:
		return newMetricServerSource(restConfig)
	case BackendPrometheus:
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("metrics backend %q requires spec.metricsEndpoint", cfg.Backend)
		}
		return newPrometheusSource(cfg.Endpoint)
	case BackendCustom:
		return nil, fmt.Errorf("metrics backend %q is not implemented yet", cfg.Backend)
	default:
		return nil, fmt.Errorf("unsupported metrics backend %q", cfg.Backend)
	}
}

// namespacesOf returns the distinct namespaces of pods, sorted.
func namespacesOf(pods []corev1.Pod) []string {
	seen := map[string]struct{}{}
	for _, pod := range pods {
		seen[pod.Namespace] = struct{}{}
	}
	namespaces := make([]string, 0, len(seen))
	for ns := range seen {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	return namespaces
}

func ptrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
