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

package workloads

import (
	"fmt"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DaemonSetWorkload implements the Workload interface for DaemonSets.
type DaemonSetWorkload struct {
	DaemonSet *appsv1.DaemonSet
}

func (w *DaemonSetWorkload) GetName() string         { return w.DaemonSet.GetName() }
func (w *DaemonSetWorkload) GetNamespace() string    { return w.DaemonSet.GetNamespace() }
func (w *DaemonSetWorkload) Resource() client.Object { return w.DaemonSet }
func (w *DaemonSetWorkload) Kind() kinds.Kind        { return kinds.DaemonSetKind }
func (w *DaemonSetWorkload) ID() string {
	return utils.GenerateID(w.Kind(), w.DaemonSet.GetNamespace(), w.DaemonSet.GetName())
}

func (w *DaemonSetWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &w.DaemonSet.Spec.Template
}

// Stable checks if the DaemonSet is stable based on its replica status.
func (w *DaemonSetWorkload) Stable() (isStable bool, reason string) {
	ds := w.DaemonSet
	updatedNumberScheduled := ds.Status.UpdatedNumberScheduled
	numberReady := ds.Status.NumberReady
	numberUnavailable := ds.Status.NumberUnavailable
	numberAvailable := ds.Status.NumberAvailable
	desiredNumberScheduled := ds.Status.DesiredNumberScheduled

	if ds.Status.ObservedGeneration < ds.Generation {
		return false, fmt.Sprintf("rollout in progress: observedGeneration=%d, generation=%d", ds.Status.ObservedGeneration, ds.Generation)
	}

	if desiredNumberScheduled == 0 {
		return true, "scaled to zero replicas" // nolint:goconst
	}

	if numberUnavailable > 0 {
		return false, fmt.Sprintf("unavailable replicas: available=%d, ready=%d, desired=%d", numberUnavailable, numberReady, desiredNumberScheduled)
	}

	if updatedNumberScheduled != desiredNumberScheduled {
		return false, fmt.Sprintf("not all replicas are updated: updated=%d, ready=%d, desired=%d", updatedNumberScheduled, numberReady, desiredNumberScheduled)
	}

	if numberReady != desiredNumberScheduled {
		return false, fmt.Sprintf("not enough ready replicas: ready=%d, desired=%d", numberReady, desiredNumberScheduled)
	}

	if numberAvailable != desiredNumberScheduled {
		return false, fmt.Sprintf("not enough available replicas: available=%d, desired=%d", numberAvailable, desiredNumberScheduled)
	}

	return true, fmt.Sprintf("workload is stable: ready=%d, desired=%d", numberReady, desiredNumberScheduled)
}
