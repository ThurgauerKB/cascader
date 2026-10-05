/*
Copyright 2026 Thurgauer Kantonalbank

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
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseReconciler contains shared fields for reconcilers.
type BaseReconciler struct {
	KubeClient                    client.Client           // KubeClient is the Kubernetes API client.
	Logger                        *logr.Logger            // Logger is used for logging reconciliation events.
	Recorder                      events.EventRecorder    // Recorder records Kubernetes events.
	Metrics                       *metrics.Registry       // Metrics is used for recording metrics.
	AnnotationKindMap             kinds.AnnotationKindMap // AnnotationKindMap maps annotation keys to workload kinds.
	LastObservedRestartAnnotation string                  // LastObservedRestartAnnotation is the annotation key for last observed restarts.
	RequeueAfterAnnotation        string                  // RequeueAfterAnnotation is the annotation key for requeue intervals.
	RequeueAfterDefault           time.Duration           // RequeueAfterDefault is the default duration for requeuing.
}

// ReconcileWorkload handles the core reconciliation logic for any workload type.
func (b *BaseReconciler) ReconcileWorkload(ctx context.Context, workload workloads.Workload) (ctrl.Result, error) {
	log := b.Logger.WithValues("workloadID", workload.ID())

	observed, err := b.observeRestart(ctx, workload, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	targets, err := b.loadTargets(ctx, workload)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(targets) == 0 {
		log.Info("No targets found; skipping reload.")
		return ctrl.Result{}, nil
	}
	if !observed {
		// Log targets only when restart was just detected.
		log.Info("Dependent targets extracted", "targets", targetIDs(targets))
	}

	duration, err := b.requeueDurationFor(workload.Resource())
	if err != nil {
		log.Error(err, fmt.Sprintf("Invalid requeue annotation, using default: %s", b.RequeueAfterDefault))
	}

	cycleDetected, err := b.handleDependencyCycle(ctx, workload, targets, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cycleDetected {
		return ctrl.Result{}, nil
	}

	if result, requeue := b.requeueIfUnstable(workload, duration, log); requeue {
		return result, nil
	}

	succeeded, err := b.completeReloads(ctx, workload, targets)
	if err != nil {
		log.Error(err, "Could not complete target reloads", "succeeded", succeeded)
		return ctrl.Result{}, nil
	}

	log.Info("Finished handling targets", "succeeded", succeeded)
	return ctrl.Result{}, nil
}

// observeRestart records a newly observed restart and reports whether it was already recorded.
func (b *BaseReconciler) observeRestart(ctx context.Context, workload workloads.Workload, log logr.Logger) (bool, error) {
	if hasAnnotation(workload.Resource(), b.LastObservedRestartAnnotation) {
		return true, nil
	}

	now := time.Now().Format(time.RFC3339)
	log.Info("Restart detected, handling targets", "restartedAt", now)
	if err := b.setLastObservedRestartAnnotation(ctx, workload, now); err != nil {
		return false, fmt.Errorf("failed to patch restart annotation: %w", err)
	}

	return false, nil
}

// loadTargets extracts workload targets and records their count.
func (b *BaseReconciler) loadTargets(ctx context.Context, workload workloads.Workload) ([]targets.Target, error) {
	targets, err := b.extractTargets(ctx, workload.Resource())
	if err != nil {
		return nil, fmt.Errorf("failed to create targets: %w", err)
	}

	b.Metrics.SetWorkloadTargets(
		workload.GetNamespace(),
		workload.GetName(),
		workload.Kind().String(),
		float64(len(targets)),
	)
	return targets, nil
}

// handleDependencyCycle records a cycle and reports whether reconciliation should stop.
func (b *BaseReconciler) handleDependencyCycle(
	ctx context.Context,
	workload workloads.Workload,
	targets []targets.Target,
	log logr.Logger,
) (bool, error) {
	err := b.checkCycle(ctx, workload.ID(), targets)
	if err == nil {
		b.Metrics.SetDependencyCycleDetected(workload.GetNamespace(), workload.GetName(), workload.Kind().String(), metrics.CycleNone)
		return false, nil
	}

	cycleErr, ok := err.(*CycleError)
	if !ok {
		return false, fmt.Errorf("check dependency cycle: %w", err)
	}

	b.Metrics.SetDependencyCycleDetected(workload.GetNamespace(), workload.GetName(), workload.Kind().String(), metrics.CycleDetected)
	b.Recorder.Eventf(
		workload.Resource(),
		nil,
		corev1.EventTypeWarning,
		"CycleDetected",
		"CheckDependencyCycle",
		"Dependency cycle detected: %s",
		cycleErr.Path,
	)
	log.Error(err, "Dependency cycle detected; skipping reload")
	return true, nil
}

// requeueIfUnstable returns a requeue result when the workload is not yet stable.
func (b *BaseReconciler) requeueIfUnstable(
	workload workloads.Workload,
	duration time.Duration,
	log logr.Logger,
) (ctrl.Result, bool) {
	stable, reason := workload.Stable()
	if !stable {
		log.Info(fmt.Sprintf("Workload not stable. Requeuing after %s.", duration), "reason", reason)
		return ctrl.Result{RequeueAfter: duration}, true
	}

	log.Info("Workload is stable", "reason", reason)
	return ctrl.Result{}, false
}

// completeReloads clears the restart marker and triggers reloads for every target.
func (b *BaseReconciler) completeReloads(
	ctx context.Context,
	workload workloads.Workload,
	targets []targets.Target,
) (succeeded int, err error) {
	clearErr := b.clearLastObservedRestartAnnotation(ctx, workload)
	succeeded, failed := b.triggerReloads(ctx, workload, targets)

	if failed > 0 {
		return succeeded, errors.Join(clearErr, fmt.Errorf("failed to trigger %d target reloads", failed))
	}

	return succeeded, clearErr
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
				nil,
				corev1.EventTypeWarning,
				"ReloadFailed",
				"TriggerReload",
				"Cascader failed to trigger reload due to change in %q: %v",
				workloadID,
				err,
			)
			fail++

			continue
		}

		b.Metrics.IncRestartsPerformed(t.Namespace(), t.Name(), kind)
		log.Info("Successfully triggered reload", "targetID", targetID)
		b.Recorder.Eventf(
			res,
			nil,
			corev1.EventTypeNormal,
			"ReloadSucceeded",
			"TriggerReload",
			"Cascader triggered reload due to change in %q",
			workloadID,
		)
		succ++
	}

	return succ, fail
}
