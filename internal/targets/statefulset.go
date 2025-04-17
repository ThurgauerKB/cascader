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

// StatefulSetTarget handles rolling restarts for StatefulSetTarget resources.
type StatefulSetTarget struct {
	namespace  string        // Namespace of the StatefulSet.
	name       string        // Name of the StatefulSet.
	kubeClient client.Client // Kubernetes client.
}

// NewStatefulSet creates a new StatefulSet target
func NewStatefulSet(namespace, name string, c client.Client) *StatefulSetTarget {
	return &StatefulSetTarget{
		namespace:  namespace,
		name:       name,
		kubeClient: c,
	}
}

func (t *StatefulSetTarget) Kind() kinds.Kind        { return kinds.StatefulSetKind }
func (t *StatefulSetTarget) Name() string            { return t.name }
func (t *StatefulSetTarget) Namespace() string       { return t.namespace }
func (t *StatefulSetTarget) Resource() client.Object { return &appsv1.StatefulSet{} }
func (t *StatefulSetTarget) ID() string              { return utils.GenerateID(t.Kind(), t.namespace, t.name) }

// Trigger updates the "restartedAt" annotation on the StatefulSet to target a rolling restart.
func (t *StatefulSetTarget) Trigger(ctx context.Context) error {
	// Fetch the existing StatefulSet.
	sts := &appsv1.StatefulSet{}
	if err := t.kubeClient.Get(ctx, client.ObjectKey{Namespace: t.namespace, Name: t.name}, sts); err != nil {
		return fmt.Errorf("failed to fetch StatefulSet %s/%s: %w", t.namespace, t.name, err)
	}

	// Update the "restartedAt" annotation on the StatefulSetSet.
	if err := utils.PatchPodTemplateAnnotation(
		ctx,
		t.kubeClient,
		sts,
		&sts.Spec.Template,
		utils.RestartedAtKey,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("failed to patch StatefulSet %s/%s: %w", t.namespace, t.name, err)
	}

	return nil
}
