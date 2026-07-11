package metrics

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)


type PrometheusCollector struct {
	*CollectorType
	api     promv1.API
	timeout time.Duration
}

func NewPrometheusCollector(k8sClient client.Client, endpoint string) (*PrometheusCollector, error) {
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

	p := &PrometheusCollector{
		api:     promv1.NewAPI(apiClient),
		timeout: 10 * time.Second,
		CollectorType: &CollectorType{
			source: "prometheus",
			listPods: func(ctx context.Context, namespace string, selector *metav1.LabelSelector) ([]corev1.Pod, error) {
				return retrieveSelectedPods(ctx, k8sClient, namespace, selector)
			},
		},
	}
	p.usageFn = p.getPodUsage
	return p, nil
}


func (p *PrometheusCollector) getPodUsage(ctx context.Context, pod corev1.Pod) (MetricResult, error) {
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return MetricResult{}, nil
	}

	ns := escapePromLabel(pod.Namespace)
	name := escapePromLabel(pod.Name)

	cpuQuery := fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!="",container!="POD"}[2m])) * 1000`,
		ns, name,
	)
	memQuery := fmt.Sprintf(
		`sum(container_memory_working_set_bytes{namespace="%s",pod="%s",container!="",container!="POD"})`,
		ns, name,
	)

	cpuMilli, err := p.queryScalar(ctx, cpuQuery)
	if err != nil {
		return MetricResult{}, fmt.Errorf("cpu query failed: %w", err)
	}
	memBytes, err := p.queryScalar(ctx, memQuery)
	if err != nil {
		return MetricResult{}, fmt.Errorf("memory query failed: %w", err)
	}

	return MetricResult{
		PodCount:    1,
		CPUmilli:    int64(cpuMilli),
		MemoryBytes: int64(memBytes),
	}, nil
}


func (p *PrometheusCollector) queryScalar(ctx context.Context, query string) (float64, error) {
	log := logf.FromContext(ctx)

	qctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	result, warnings, err := p.api.Query(qctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	for _, w := range warnings {
		log.Info("Prometheus query warning", "warning", w, "query", query)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return 0, fmt.Errorf("unexpected prometheus result type %T for query %q", result, query)
	}
	if len(vector) == 0 {
		return 0, nil
	}
	return float64(vector[0].Value), nil
}


func escapePromLabel(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return v
}

var _ MetricsCollector = (*PrometheusCollector)(nil)
