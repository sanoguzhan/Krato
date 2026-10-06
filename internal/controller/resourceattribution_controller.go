/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/sanoguzhan/krato/internal/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
)

type metricsBackend string

const (
	metricsBackendRequests     metricsBackend = "request"
	metricsBackendPrometheus   metricsBackend = "prometheus"
	metricsBackendMetricServer metricsBackend = "metrics-server"
	metricsBackendCustom       metricsBackend = "custom"
)

type metricsSourceConfig struct {
	Backend  metricsBackend
	Endpoint string
}

// ResourceAttributionReconciler reconciles a ResourceAttribution object
type ResourceAttributionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *rest.Config
}

// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ResourceAttribution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ResourceAttributionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var resourceAttribution attributionv1alpha1.ResourceAttribution
	if err := r.Get(ctx, req.NamespacedName, &resourceAttribution); err != nil {
		// If the resource is not found, it might have been deleted after the reconcile request was queued.
		// In this case, we can ignore the error and return.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if resourceAttribution.Spec.Selector == nil {
		return ctrl.Result{}, nil
	}

	config := resolveMetricsSource(resourceAttribution.Spec, resourceAttribution.Status)
	collector, err := selectMetricsCollector(config, r.Client, r.Config)
	if err != nil {
		log.Error(err, "Failed to select metrics collector", "backend", config.Backend, "endpoint", config.Endpoint)
		return ctrl.Result{}, err
	}

	metricsResult, err := collector.CollectMetrics(ctx, resourceAttribution.Namespace, resourceAttribution.Spec.Selector)
	if err != nil {
		log.Error(err, "Failed to collect metrics", "backend", config.Backend, "endpoint", config.Endpoint)
		return ctrl.Result{}, err
	}

	log.Info("Collected workload metrics", "backend", config.Backend, "cpuMilli", metricsResult.CPUmilli, "memoryBytes", metricsResult.MemoryBytes, "podCount", metricsResult.PodCount)

	return ctrl.Result{}, nil
}

func resolveMetricsSource(spec attributionv1alpha1.ResourceAttributionSpec, status attributionv1alpha1.ResourceAttributionStatus) metricsSourceConfig {
	backend := strings.ToLower(strings.TrimSpace(ptrValue(spec.MetricsBackend)))
	endpoint := strings.TrimSpace(ptrValue(spec.MetricsEndpoint))

	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(ptrValue(status.MetricsServerBackend)))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(ptrValue(status.MetricsServerEndpoint))
	}

	if backend == "" {
		return metricsSourceConfig{Backend: metricsBackendRequests}
	}

	return metricsSourceConfig{Backend: metricsBackend(backend), Endpoint: endpoint}
}

func selectMetricsCollector(config metricsSourceConfig, kubeClient client.Client, cfg *rest.Config) (metrics.MetricsCollector, error) {
	switch config.Backend {
	case metricsBackendRequests:
		return metrics.NewRequestCollector(kubeClient), nil
	case metricsBackendMetricServer:
		return metrics.NewMetricServerCollector(kubeClient, cfg)
	case metricsBackendPrometheus:
		if config.Endpoint == "" {
			return nil, fmt.Errorf("metrics backend %q requires spec.metricsEndpoint", config.Backend)
		}
		return metrics.NewPrometheusCollector(kubeClient, config.Endpoint)
	case metricsBackendCustom:
		return nil, fmt.Errorf("metrics backend %q is not implemented yet", config.Backend)
	default:
		return nil, fmt.Errorf("unsupported metrics backend %q", config.Backend)
	}
}

func ptrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceAttributionReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&attributionv1alpha1.ResourceAttribution{}).
		Named("resourceattribution").
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}
