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

	"github.com/sanoguzhan/krato/internal/source"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
)

// conditionReady reports whether the spec is valid. Usage itself is collected
// by the collector loop and exported as Prometheus metrics, not written here.
const conditionReady = "Ready"

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

	condition := metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Configured",
		Message:            "Spec is valid; usage is collected by the collector loop",
		ObservedGeneration: resourceAttribution.Generation,
	}
	if err := r.validate(&resourceAttribution); err != nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "InvalidSpec"
		condition.Message = err.Error()
	}

	if !meta.SetStatusCondition(&resourceAttribution.Status.Conditions, condition) {
		return ctrl.Result{}, nil
	}
	if err := r.Status().Update(ctx, &resourceAttribution); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Updated ResourceAttribution status", "condition", condition.Type, "status", condition.Status, "reason", condition.Reason)
	return ctrl.Result{}, nil
}

// validate checks what the collector loop needs: a usable selector and a
// metrics backend that can be built.
func (r *ResourceAttributionReconciler) validate(ra *attributionv1alpha1.ResourceAttribution) error {
	if ra.Spec.Selector == nil {
		return fmt.Errorf("spec.selector must be set")
	}
	if _, err := metav1.LabelSelectorAsSelector(ra.Spec.Selector); err != nil {
		return fmt.Errorf("invalid spec.selector: %w", err)
	}
	if _, err := source.New(source.ConfigFor(ra), r.Config); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceAttributionReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&attributionv1alpha1.ResourceAttribution{}).
		Named("resourceattribution").
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}
