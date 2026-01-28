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

// DeploymentWorkload implements the workload interface for Deployments.
type DeploymentWorkload struct {
	Deployment *appsv1.Deployment
}

func (w *DeploymentWorkload) GetName() string         { return w.Deployment.GetName() }
func (w *DeploymentWorkload) GetNamespace() string    { return w.Deployment.GetNamespace() }
func (w *DeploymentWorkload) Resource() client.Object { return w.Deployment }
func (w *DeploymentWorkload) Kind() kinds.Kind        { return kinds.DeploymentKind }
func (w *DeploymentWorkload) ID() string {
	return utils.GenerateID(w.Kind(), w.Deployment.GetNamespace(), w.Deployment.GetName())
}

func (w *DeploymentWorkload) PodTemplateSpec() *corev1.PodTemplateSpec {
	return &w.Deployment.Spec.Template
}

// Stable checks if the Deployment is stable based on replica status.
func (w *DeploymentWorkload) Stable() (isStable bool, reason string) {
	dep := w.Deployment
	available := dep.Status.AvailableReplicas
	updated := dep.Status.UpdatedReplicas
	ready := dep.Status.ReadyReplicas
	unavailable := dep.Status.UnavailableReplicas
	desired := *dep.Spec.Replicas

	if dep.Status.ObservedGeneration < dep.Generation {
		return false, fmt.Sprintf("rollout in progress: observedGeneration=%d, generation=%d", dep.Status.ObservedGeneration, dep.Generation)
	}

	if desired == 0 {
		return true, "scaled to zero replicas" // nolint:goconst
	}

	if unavailable > 0 {
		return false, fmt.Sprintf("unavailable replicas: unavailable=%d, ready=%d, desired=%d", unavailable, ready, desired)
	}

	if updated != desired {
		return false, fmt.Sprintf("not all replicas are updated: updated=%d, ready=%d, desired=%d", updated, ready, desired)
	}

	if ready != desired {
		return false, fmt.Sprintf("not enough ready replicas: ready=%d, desired=%d", ready, desired)
	}

	if available != desired {
		return false, fmt.Sprintf("not enough available replicas: available=%d, desired=%d", available, desired)
	}

	return true, fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desired)
}
