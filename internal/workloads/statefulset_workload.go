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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulSetWorkload implements the workload interface for StatefulSets.
type StatefulSetWorkload struct {
	StatefulSet *appsv1.StatefulSet
}

func (w *StatefulSetWorkload) Resource() client.Object { return w.StatefulSet }
func (w *StatefulSetWorkload) Kind() kinds.Kind        { return kinds.StatefulSetKind }
func (w *StatefulSetWorkload) ID() string {
	return utils.GenerateID(w.Kind(), w.StatefulSet.GetNamespace(), w.StatefulSet.GetName())
}

// Stable checks if the StatefulSet is stable based on its replica status.
func (w *StatefulSetWorkload) Stable() (isStable bool, reason string) {
	sts := w.StatefulSet
	updated := sts.Status.UpdatedReplicas
	ready := sts.Status.ReadyReplicas
	desired := *sts.Spec.Replicas

	if sts.Status.ObservedGeneration < sts.Generation {
		return false, fmt.Sprintf("rollout in progress: observedGeneration=%d, generation=%d", sts.Status.ObservedGeneration, sts.Generation)
	}

	if desired == 0 {
		return true, "scaled to zero replicas" // nolint:goconst
	}

	if updated != desired {
		return false, fmt.Sprintf("not all replicas are updated: updated=%d, ready=%d, desired=%d", updated, ready, desired)
	}

	if ready != desired {
		return false, fmt.Sprintf("not enough ready replicas: ready=%d, desired=%d", ready, desired)
	}

	// StatefulSets don't have an AvailableReplicas field, so we rely on ReadyReplicas.
	return true, fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desired)
}
