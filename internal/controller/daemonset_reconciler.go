/*
Copyright 2025 Thurgauer Kantonalbank

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
	"errors"

	"github.com/thurgauerkb/cascader/internal/predicates"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DaemonSetReconciler reconciles DaemonSets to detect restarts and target reloads.
type DaemonSetReconciler struct {
	BaseReconciler
}

// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create

// Reconcile handles the reconciliation logic when a DaemonSet is updated.
func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the DaemonSet instance
	ds := &appsv1.DaemonSet{}
	if err := r.KubeClient.Get(ctx, req.NamespacedName, ds); err != nil {
		if kerrors.IsNotFound(err) {
			logger.Info("DaemonSet not found; ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.New("failed to fetch DaemonSet")
	}

	return r.ReconcileWorkload(ctx, ds)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		WithEventFilter(predicates.NewPredicate(
			r.AnnotationKindMap,
			predicates.SpecChanged,
			predicates.WrapSingleObjectCheck(predicates.DaemonSetTransitioning),
			predicates.ScaledToZero,
			predicates.ScaledFromZero,
		)).
		Complete(r)
}
