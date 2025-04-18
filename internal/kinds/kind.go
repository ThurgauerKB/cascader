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

package kinds

// AnnotationKindMap maps annotation keys to workload kinds.
type AnnotationKindMap map[string]Kind

// Kind represents the type of a Kubernetes resource.
type Kind string

const (
	DaemonSetKind   Kind = "DaemonSet"   // Represents a Kubernetes DaemonSet resource.
	DeploymentKind  Kind = "Deployment"  // Represents a Kubernetes Deployment resource.
	StatefulSetKind Kind = "StatefulSet" // Represents a Kubernetes StatefulSet resource.
)

// String converts the Kind to its string representation.
func (k Kind) String() string {
	return string(k)
}
