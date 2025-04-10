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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workload defines the interface for Kubernetes workloads, providing methods for stability checks and metadata access.
type Workload interface {
	Resource() client.Object                  // Resource returns the underlying Kubernetes object.
	Kind() kinds.Kind                         // Kind returns the kind of the workload.
	ID() string                               // ID returns a unique identifier for the workload in the format Kind/namespace/name.
	Stable() (isStable bool, reason string)   // Stable checks if the workload is in a stable state.
	PodTemplateSpec() *corev1.PodTemplateSpec // PodTemplateSpec returns the PodTemplateSpec of the workload.
}

// NewWorkload creates a Workload implementation for the given Kubernetes object.
// Returns an error if the object type is not supported.
func NewWorkload(obj client.Object) (Workload, error) {
	switch resource := obj.(type) {
	case *appsv1.Deployment:
		return &DeploymentWorkload{Deployment: resource}, nil
	case *appsv1.StatefulSet:
		return &StatefulSetWorkload{StatefulSet: resource}, nil
	case *appsv1.DaemonSet:
		return &DaemonSetWorkload{DaemonSet: resource}, nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %T", obj)
	}
}
