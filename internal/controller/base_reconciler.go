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

	// Detect workload restart
	updated, restartedAt := restartMarkerUpdated(workload.PodTemplateSpec(), utils.RestartedAtKey, b.LastObservedRestartAnnotation)
	if updated {
		log.Info("Restart detected, handling targets", "restartedAt", restartedAt)

		if err := b.patchRestartMarker(ctx, workload, restartedAt); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to patch restart annotation: %w", err)
		}
	}

	// Extract dependent targets from workload annotations.
	targets, err := b.extractTargets(ctx, res)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create targets: %w", err)
	}
	if len(targets) == 0 {
		log.Info("No targets found; skipping reload.")
		return ctrl.Result{}, nil
	}
	if updated {
		// 'updated' indicates a restart; log targets only in that case to avoid redundant logging on unstable workloads
		log.Info("Dependent targets extracted", "targets", targetIDs(targets))
	}
	metrics.WorkloadTargets.WithLabelValues(ns, name, kind).Set(float64(len(targets)))

	// Determine requeue interval.
	dur, err := b.requeueDurationFor(res)
	if err != nil {
		log.Error(err, fmt.Sprintf("Invalid requeue annotation, using default: %s", b.RequeueAfterDefault))
	}

	// Detect and prevent dependency cycles.
	if err := b.checkCycle(ctx, id, targets); err != nil {
		if cycleErr, ok := err.(*CycleError); ok {
			metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(metrics.CycleDetected)
			b.Recorder.Eventf(res, corev1.EventTypeWarning, "CycleDetected", "Dependency cycle detected: %s", cycleErr.Path)
		}
		log.Error(err, "Dependency cycle detected; skipping reload")
		return ctrl.Result{}, nil // Do not return an error to avoid requeuing the workload.
	}
	metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(metrics.CycleNone) // Reset metric

	// Ensure workload stability.
	stable, reason := workload.Stable()
	if !stable {
		log.Info(fmt.Sprintf("Workload not stable. Requeuing after %s.", dur), "reason", reason)
		return ctrl.Result{RequeueAfter: dur}, nil
	}
	log.Info("Workload is stable", "reason", reason)

	// Attempt target reloads.
	succ, fail := b.triggerReloads(ctx, workload, targets)
	if fail > 0 {
		log.Error(errors.New("partial target reload failure"), "Some targets failed to reload", "succeeded", succ, "failed", fail)
		return ctrl.Result{}, nil // Do not return an error to avoid requeuing the workload.
	}

	log.Info("Finished handling targets", "succeeded", succ, "failed", fail)

	return ctrl.Result{}, nil
}

// patchRestartMarker updates the restart annotation in the workload's PodTemplateSpec.
func (b *BaseReconciler) patchRestartMarker(
	ctx context.Context,
	workload workloads.Workload,
	restartedAt string,
) error {
	return utils.PatchPodTemplateAnnotation(
		ctx,
		b.KubeClient,
		workload.Resource(),
		workload.PodTemplateSpec(),
		b.LastObservedRestartAnnotation,
		restartedAt,
	)
}

// extractTargets creates targets for a workload based on annotations.
func (b *BaseReconciler) extractTargets(ctx context.Context, source client.Object) ([]targets.Target, error) {
	var created []targets.Target

	annotations := source.GetAnnotations()
	if annotations == nil {
		return created, nil
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
			created = append(created, t)
		}
	}

	return created, nil
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
