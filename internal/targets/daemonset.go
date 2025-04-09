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
	"time"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DaemonSetTarget handles rolling restarts for DaemonSetTarget resources.
type DaemonSetTarget struct {
	namespace  string        // Namespace of the DaemonSet.
	name       string        // Name of the DaemonSet.
	kubeClient client.Client // Kubernetes client.
}

func (t *DaemonSetTarget) Kind() kinds.Kind        { return kinds.DaemonSetKind }
func (t *DaemonSetTarget) Name() string            { return t.name }
func (t *DaemonSetTarget) Namespace() string       { return t.namespace }
func (t *DaemonSetTarget) Resource() client.Object { return &appsv1.DaemonSet{} }
func (t *DaemonSetTarget) ID() string              { return utils.GenerateID(t.Kind(), t.namespace, t.name) }

// NewDaemonSet creates a new DaemonSet target
func NewDaemonSet(namespace, name string, c client.Client) *DaemonSetTarget {
	return &DaemonSetTarget{
		namespace:  namespace,
		name:       name,
		kubeClient: c,
	}
}

// Trigger updates the "restartedAt" annotation on the DaemonSet to target a rolling restart.
func (t *DaemonSetTarget) Trigger(ctx context.Context) error {
	// Fetch the existing DaemonSet.
	ds := &appsv1.DaemonSet{}
	if err := t.kubeClient.Get(ctx, client.ObjectKey{Namespace: t.namespace, Name: t.name}, ds); err != nil {
		return fmt.Errorf("failed to fetch DaemonSet %s/%s: %w", t.namespace, t.name, err)
	}

	// Update the "restartedAt" annotation on the DaemonSet.
	if err := utils.PatchPodTemplateAnnotation(
		ctx,
		t.kubeClient,
		ds,
		&ds.Spec.Template,
		utils.RestartedAtKey,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("failed to patch DaemonSet %s/%s: %w", t.namespace, t.name, err)
	}

	return nil
}
