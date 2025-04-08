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
	"github.com/thurgauerkb/cascader/internal/workloads"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseReconciler contains shared fields for reconcilers.
type BaseReconciler struct {
	KubeClient             client.Client           // KubeClient is the Kubernetes API client.
	Logger                 *logr.Logger            // Logger is used for logging reconciliation events.
	Recorder               record.EventRecorder    // Recorder records Kubernetes events.
	AnnotationKindMap      kinds.AnnotationKindMap // AnnotationKindMap maps annotation keys to workload kinds.
	RequeueAfterAnnotation string                  // RequeueAfterAnnotation is the annotation key for requeue intervals.
	RequeueAfterDefault    time.Duration           // RequeueAfterDefault is the default duration for requeuing.
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

	// Extract dependent targets from workload annotations.
	targets, err := b.extractTargets(ctx, res)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create targets: %w", err)
	}
	if len(targets) == 0 {
		log.Info("No targets found; skipping reload.")
		return ctrl.Result{}, nil
	}
	metrics.Workloads.WithLabelValues(ns, name, kind).Set(float64(len(targets)))

	// Determine requeue interval.
	dur, err := b.getRequeueDuration(res)
	if err != nil {
		log.Error(err, fmt.Sprintf("Invalid requeue annotation, using default: %s", b.RequeueAfterDefault))
	}

	// Detect and prevent dependency cycles.
	if err := b.checkCycle(ctx, id, targets); err != nil {
		if cycleErr, ok := err.(*CycleError); ok {
			metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(1)
			b.Recorder.Eventf(res, corev1.EventTypeWarning, "CycleDetected", "Dependency cycle detected: %s", cycleErr.DepChain)
		}
		return ctrl.Result{}, fmt.Errorf("dependency cycle detected: %w", err)
	}

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

	metrics.DependencyCyclesDetected.WithLabelValues(ns, name, kind).Set(0) // Reset metric
	log.Info("Triggered reloads", "succeeded", succ, "failed", fail)

	return ctrl.Result{}, nil
}

// extractTargets creates targets for a workload based on annotations.
func (b *BaseReconciler) extractTargets(ctx context.Context, source client.Object) ([]targets.Target, error) {
	var created []targets.Target

	anns := source.GetAnnotations()
	if anns == nil {
		return created, nil
	}

	for ann, kind := range b.AnnotationKindMap {
		val, exists := anns[ann]
		if !exists {
			continue
		}

		// Targets can be specified as a comma-separated list.
		for _, ref := range strings.Split(val, ",") {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				return nil, errors.New("targets cannot be empty")
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

// getRequeueDuration determines requeue interval from annotations or falls back to default.
func (b *BaseReconciler) getRequeueDuration(obj client.Object) (time.Duration, error) {
	anns := obj.GetAnnotations()
	if anns == nil {
		return b.RequeueAfterDefault, nil
	}

	val, exists := anns[b.RequeueAfterAnnotation]
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
