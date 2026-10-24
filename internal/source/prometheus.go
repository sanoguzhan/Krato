package source

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// prometheusSource reads usage from Prometheus with one query for CPU and one
// for memory, covering every pod in the requested namespaces.
type prometheusSource struct {
	api     promv1.API
	timeout time.Duration
}

func newPrometheusSource(endpoint string) (*prometheusSource, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("prometheus endpoint must not be empty")
	}
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("invalid prometheus endpoint %q: %w", endpoint, err)
	}

	apiClient, err := promapi.NewClient(promapi.Config{Address: endpoint})
	if err != nil {
		return nil, fmt.Errorf("failed to build prometheus client: %w", err)
	}

	return &prometheusSource{
		api:     promv1.NewAPI(apiClient),
		timeout: 10 * time.Second,
	}, nil
}

func (p *prometheusSource) PodUsage(ctx context.Context, pods []corev1.Pod) (map[types.NamespacedName]Usage, error) {
	usage := map[types.NamespacedName]Usage{}
	namespaces := namespacesOf(pods)
	if len(namespaces) == 0 {
		return usage, nil
	}
	nsRegex := escapePromLabel(strings.Join(namespaces, "|"))

	cpuQuery := fmt.Sprintf(
		`sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{namespace=~"%s",container!="",container!="POD"}[2m])) * 1000`,
		nsRegex,
	)
	memQuery := fmt.Sprintf(
		`sum by (namespace, pod) (container_memory_working_set_bytes{namespace=~"%s",container!="",container!="POD"})`,
		nsRegex,
	)

	cpuMilli, err := p.queryByPod(ctx, cpuQuery)
	if err != nil {
		return nil, fmt.Errorf("cpu query failed: %w", err)
	}
	memBytes, err := p.queryByPod(ctx, memQuery)
	if err != nil {
		return nil, fmt.Errorf("memory query failed: %w", err)
	}

	for pod, v := range cpuMilli {
		u := usage[pod]
		u.CPUmilli = int64(v)
		usage[pod] = u
	}
	for pod, v := range memBytes {
		u := usage[pod]
		u.MemoryBytes = int64(v)
		usage[pod] = u
	}
	return usage, nil
}

// queryByPod runs an instant query whose result is labelled by namespace and pod.
func (p *prometheusSource) queryByPod(ctx context.Context, query string) (map[types.NamespacedName]float64, error) {
	log := logf.FromContext(ctx)

	qctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	result, warnings, err := p.api.Query(qctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		log.Info("Prometheus query warning", "warning", w, "query", query)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected prometheus result type %T for query %q", result, query)
	}

	values := make(map[types.NamespacedName]float64, len(vector))
	for _, sample := range vector {
		pod := types.NamespacedName{
			Namespace: string(sample.Metric["namespace"]),
			Name:      string(sample.Metric["pod"]),
		}
		values[pod] = float64(sample.Value)
	}
	return values, nil
}

func escapePromLabel(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return v
}

var _ Source = (*prometheusSource)(nil)
