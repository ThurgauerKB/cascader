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

package targets

import (
	"context"
	"fmt"
	"time"

	"github.com/thurgauerkb/cascader/internal/flag"
	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentTarget handles rolling restarts for Deployment resources.
type DeploymentTarget struct {
	namespace  string        // Namespace of the Deployment.
	name       string        // Name of the Deployment.
	kubeClient client.Client // Kubernetes client.
}

// NewDeployment creates a Deployment restart target.
func NewDeployment(namespace, name string, c client.Client) *DeploymentTarget {
	return &DeploymentTarget{
		namespace:  namespace,
		name:       name,
		kubeClient: c,
	}
}

// Kind returns the Kubernetes kind managed by the target.
func (t *DeploymentTarget) Kind() kinds.Kind { return kinds.DeploymentKind }

// Name returns the target Deployment name.
func (t *DeploymentTarget) Name() string { return t.name }

// Namespace returns the target Deployment namespace.
func (t *DeploymentTarget) Namespace() string { return t.namespace }

// Resource returns an empty Deployment used to fetch the target.
func (t *DeploymentTarget) Resource() client.Object { return &appsv1.Deployment{} }

// ID returns the target's stable workload identifier.
func (t *DeploymentTarget) ID() string { return utils.GenerateID(t.Kind(), t.namespace, t.name) }

// Trigger updates the "restartedAt" annotation on the Deployment to target a rolling restart.
func (t *DeploymentTarget) Trigger(ctx context.Context) error {
	// Fetch the existing Deployment.
	dep := &appsv1.Deployment{}
	if err := t.kubeClient.Get(ctx, client.ObjectKey{Namespace: t.namespace, Name: t.name}, dep); err != nil {
		return fmt.Errorf("failed to fetch Deployment %s/%s: %w", t.namespace, t.name, err)
	}

	// Update the "restartedAt" annotation on the Deployment.
	if err := utils.PatchPodTemplateAnnotation(
		ctx,
		t.kubeClient,
		dep,
		&dep.Spec.Template,
		flag.LastObservedRestartAnnotation,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("failed to patch Deployment %s/%s: %w", t.namespace, t.name, err)
	}

	return nil
}
