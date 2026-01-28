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
	"github.com/thurgauerkb/cascader/internal/workloads"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StatefulSetReconciler reconciles StatefulSets to detect restarts and target reloads.
type StatefulSetReconciler struct {
	BaseReconciler
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create

// Reconcile handles the reconciliation logic when a StatefulSet is updated.
func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the StatefulSet instance
	sts := &appsv1.StatefulSet{}
	if err := r.KubeClient.Get(ctx, req.NamespacedName, sts); err != nil {
		if kerrors.IsNotFound(err) {
			logger.Info("StatefulSet not found; ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.New("failed to fetch StatefulSet")
	}

	return r.ReconcileWorkload(ctx, &workloads.StatefulSetWorkload{StatefulSet: sts})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithEventFilter(predicates.NewPredicate(
			r.AnnotationKindMap,
			predicates.SpecChanged,
			predicates.SingleReplicaPodDeleted,
			predicates.ScaledToZero,
			predicates.ScaledFromZero,
		)).
		Complete(r)
}
