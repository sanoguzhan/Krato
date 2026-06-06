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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceAttributionReconciler reconciles a ResourceAttribution object
type ResourceAttributionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=attribution.krato.io,resources=resourceattributions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ResourceAttribution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ResourceAttributionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var isDefaultMetricsServer bool = false
	var metricsServerBackend, metricsServerEndpoint *string
	var totalCPUUsage, totalMemoryUsage int64
	_ = logf.FromContext(ctx)

	var resourceAttribution attributionv1alpha1.ResourceAttribution
	if err := r.Get(ctx, req.NamespacedName, &resourceAttribution); err != nil {
		// If the resource is not found, it might have been deleted after the reconcile request was queued.
		// In this case, we can ignore the error and return.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	selector, err := metav1.LabelSelectorAsSelector(&resourceAttribution.Spec.Selector)
	if err != nil {
		logf.FromContext(ctx).Error(err, "Failed to convert label selector to selector")
		return ctrl.Result{}, err
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(resourceAttribution.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}	

	
	if resourceAttribution.Status.MetricsServerBackend != "" && resourceAttribution.Status.MetricsServerEndpoint != "" {
			logf.FromContext(ctx).Info("ResourceAttribution has metrics server configured", "backend", resourceAttribution.Status.MetricsServerBackend, "endpoint", resourceAttribution.Status.MetricsServerEndpoint)
			metricsServerBackend = &resourceAttribution.Status.MetricsServerBackend
			metricsServerEndpoint = &resourceAttribution.Status.MetricsServerEndpoint
	} else {
		isDefaultMetricsServer = true
			logf.FromContext(ctx).Info("ResourceAttribution does not have metrics server configured")
	}
	for _, pod := range podList.Items {
		if isDefaultMetricsServer {
	        totalCPUUsage += pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
	        totalMemoryUsage += pod.Spec.Containers[0].Resources.Requests.Memory().Value()
		}			
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceAttributionReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&attributionv1alpha1.ResourceAttribution{}).
		Named("resourceattribution").
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}
