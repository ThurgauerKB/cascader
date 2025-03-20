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

package targets

import (
	"context"
	"fmt"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentSet handles rolling restarts for DeploymentSet resources.
type DeploymentTarget struct {
	namespace  string        // Namespace of the DeploymentSet.
	name       string        // Name of the DeploymentSet.
	kubeClient client.Client // Kubernetes client.
}

func (t *DeploymentTarget) Kind() kinds.Kind        { return kinds.DeploymentKind }
func (t *DeploymentTarget) Name() string            { return t.name }
func (t *DeploymentTarget) Namespace() string       { return t.namespace }
func (t *DeploymentTarget) Resource() client.Object { return &appsv1.Deployment{} }
func (t *DeploymentTarget) ID() string              { return utils.GenerateID(t.Kind(), t.namespace, t.name) }

// NewDeployment creates a new Deployment target
func NewDeployment(namespace, name string, c client.Client) *DeploymentTarget {
	return &DeploymentTarget{
		namespace:  namespace,
		name:       name,
		kubeClient: c,
	}
}

// Trigger updates the "restartedAt" annotation on the Deployment to target a rolling restart.
func (t *DeploymentTarget) Trigger(ctx context.Context) error {
	// Fetch the existing Deployment.
	dep := &appsv1.Deployment{}
	if err := t.kubeClient.Get(ctx, client.ObjectKey{Namespace: t.namespace, Name: t.name}, dep); err != nil {
		return fmt.Errorf("failed to fetch Deployment %s/%s: %w", t.namespace, t.name, err)
	}

	// Prepare a patch to modify the Pod Template's annotations.
	patch := client.MergeFrom(dep.DeepCopy())

	updateRestartedAtAnnotation(&dep.Spec.Template)

	// Apply the patch to the Deployment.
	if err := t.kubeClient.Patch(ctx, dep, patch); err != nil {
		return fmt.Errorf("failed to patch Deployment %s/%s: %w", t.namespace, t.name, err)
	}

	return nil
}
