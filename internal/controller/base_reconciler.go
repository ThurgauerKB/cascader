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
	"fmt"
	"strings"
	"time"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/metrics"
	"github.com/thurgauerkb/cascader/internal/targets"
	"github.com/thurgauerkb/cascader/internal/utils"
	"github.com/thurgauerkb/cascader/internal/workloads"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseReconciler contains shared fields for reconcilers.
type BaseReconciler struct {
	KubeClient                    client.Client           // KubeClient is the Kubernetes API client.
	Logger                        *logr.Logger            // Logger is used for logging reconciliation events.
	Recorder                      record.EventRecorder    // Recorder records Kubernetes events.
	AnnotationKindMap             kinds.AnnotationKindMap // AnnotationKindMap maps annotation keys to workload kinds.
	LastObservedRestartAnnotation string                  // LastObservedRestartAnnotation is the annotation key for last observed restarts.
	RequeueAfterAnnotation        string                  // RequeueAfterAnnotation is the annotation key for requeue intervals.
	RequeueAfterDefault           time.Duration           // RequeueAfterDefault is the default duration for requeuing.
}

// ReconcileWorkload handles the core reconciliation logic for any workload type.
func (b *BaseReconciler) ReconcileWorkload(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	ns, name := obj.GetNamespace(), obj.GetName()

	// Initialize a workload instance from the given Kubernetes object.
	workload, err := workloads.NewWorkload(obj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create workload for %s/%s: %w", ns, name, err)
	}

	res := workload.Resource()
	id := workload.ID()
	kind := workload.Kind().String()

	log := b.Logger.WithValues("workloadID", id) // Append workload ID to logger context

	// If the last-observed-restart annotation is not present, this is the first time the workload is being processed.
	// The annotation will be removed after a successful reconciliation.
	observed := hasAnnotation(res, b.LastObservedRestartAnnotation)
	if !observed {
		now := time.Now().Format(time.RFC3339)
		log.Info("Restart detected, handling targets", "restartedAt", now)
		if err := b.setLastObservedRestartAnnotation(ctx, workload, now); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch restart annotation: %w", err)
		}
	}

	// Extract dependent targets from workload annotations.
	targets, err := b.extractTargets(ctx, res)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create targets: %w", err)
	}
	// Set the number of targets as a metric, even if no targets are found.
	metrics.WorkloadTargets.WithLabelValues(ns, name, kind).Set(float64(len(targets)))

	if len(targets) == 0 {
		log.Info("No targets found; skipping reload.")
		return ctrl.Result{}, nil
	}
	if !observed {
		// Log targets only when restart was just detected.
		log.Info("Dependent targets extracted", "targets", targetIDs(targets))
	}

	// Determine requeue interval.
	dur, err := b.requeueDurationFor(res)
	if err != nil {
		log.Error(err, fmt.Sprintf("Invalid requeue annotation, using default: %s", b.RequeueAfterDefault))
	}

	// Check for and handle circular dependencies among workloads to prevent infinite reload loops.
	if err := b.checkCycle(ctx, id, targets); err != nil {
		if cycleErr, ok := err.(*CycleError); ok {
			metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(metrics.CycleDetected)
			b.Recorder.Eventf(res, corev1.EventTypeWarning, "CycleDetected", "Dependency cycle detected: %s", cycleErr.Path)
		}
		log.Error(err, "Dependency cycle detected; skipping reload")
		return ctrl.Result{}, nil // Do not return an error to avoid requeuing the workload.
	}
	// Reset dependency cycle metric to indicate no cycle was detected.
	metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(metrics.CycleNone)

	// Check if the workload is in a stable state before triggering reloads.
	stable, reason := workload.Stable()
	if !stable {
		log.Info(fmt.Sprintf("Workload not stable. Requeuing after %s.", dur), "reason", reason)
		return ctrl.Result{RequeueAfter: dur}, nil
	}
	log.Info("Workload is stable", "reason", reason)

	// Always remove the restartedAt annotation, even if target reloads will fail.
	if err := b.clearLastObservedRestartAnnotation(ctx, workload); err != nil {
		b.Logger.Error(err, "Failed to delete restartedAt annotation")
	}

	// Trigger reloads on all dependent targets and collect success/failure counts.
	succ, fail := b.triggerReloads(ctx, workload, targets)
	if fail > 0 {
		// Some targets failed to reload. We log the error but do not return it,
		// to avoid requeuing the workload unnecessarily.
		log.Error(errors.New("partial target reload failure"), "Some targets failed to reload", "succeeded", succ, "failed", fail)
		return ctrl.Result{}, nil
	}

	log.Info("Finished handling targets", "succeeded", succ, "failed", fail)

	return ctrl.Result{}, nil
}

// setLastObservedRestartAnnotation sets the last-observed-restart annotation on the given workload.
func (b *BaseReconciler) setLastObservedRestartAnnotation(
	ctx context.Context,
	workload workloads.Workload,
	value string,
) error {
	return utils.PatchWorkloadAnnotation(
		ctx,
		b.KubeClient,
		workload.Resource(),
		b.LastObservedRestartAnnotation,
		value,
	)
}

// clearLastObservedRestartAnnotation removes the last-observed-restart annotation from the given workload.
func (b *BaseReconciler) clearLastObservedRestartAnnotation(
	ctx context.Context,
	workload workloads.Workload,
) error {
	return utils.DeleteWorkloadAnnotation(
		ctx,
		b.KubeClient,
		workload.Resource(),
		b.LastObservedRestartAnnotation,
	)
}

// extractTargets parses annotations to extract dependent workload targets.
func (b *BaseReconciler) extractTargets(ctx context.Context, source client.Object) ([]targets.Target, error) {
	var targetList []targets.Target

	annotations := source.GetAnnotations()
	if annotations == nil {
		return targetList, nil
	}

	for key, kind := range b.AnnotationKindMap {
		val, exists := annotations[key]
		if !exists {
			continue
		}

		// Targets can be specified as a comma-separated list.
		for _, ref := range strings.Split(val, ",") {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				continue
			}

			t, err := targets.NewTarget(ctx, b.KubeClient, kind, ref, source)
			if err != nil {
				return nil, fmt.Errorf("cannot create target for workload: %w", err)
			}
			targetList = append(targetList, t)
		}
	}

	return targetList, nil
}

// requeueDurationFor determines requeue interval from annotations or falls back to default.
func (b *BaseReconciler) requeueDurationFor(obj client.Object) (time.Duration, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return b.RequeueAfterDefault, nil
	}

	val, exists := annotations[b.RequeueAfterAnnotation]
	if !exists || strings.TrimSpace(val) == "" {
		return b.RequeueAfterDefault, nil
	}

	dur, err := time.ParseDuration(val)
	if err != nil {
		return b.RequeueAfterDefault, fmt.Errorf("invalid annotation: %w", err)
	}

	return dur, nil
}

// triggerReloads attempts to trigger reload for each target, returning success and failure counts.
func (b *BaseReconciler) triggerReloads(ctx context.Context, workload workloads.Workload, targets []targets.Target) (succ, fail int) {
	res := workload.Resource()
	workloadID := workload.ID()
	log := b.Logger.WithValues("workloadID", workloadID) // Append workload ID to logger context

	for _, t := range targets {
		targetID := t.ID()
		kind := t.Kind().String()

		if err := t.Trigger(ctx); err != nil {
			log.Error(err, "Failed to trigger reload", "targetID", targetID)
			b.Recorder.Eventf(
				res,
				corev1.EventTypeWarning,
				"ReloadFailed",
				"Cascader failed to trigger reload due to change in %q: %v",
				workloadID, err,
			)
			fail++

			continue
		}

		metrics.RestartsPerformed.WithLabelValues(kind, t.Namespace(), t.Name()).Inc()
		log.Info("Successfully triggered reload", "targetID", targetID)
		b.Recorder.Eventf(
			res,
			corev1.EventTypeNormal,
			"ReloadSucceeded",
			"Cascader triggered reload due to change in %q",
			workloadID,
		)
		succ++
	}

	return succ, fail
}
