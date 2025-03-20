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

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Target represents an abstract target that can be reloaded.
type Target interface {
	Kind() kinds.Kind                  // Kind returns the kind of the target.
	Name() string                      // Name returns the name of the target resource.
	Namespace() string                 // Namespace returns the namespace of the target resource.
	Resource() client.Object           // Resource returns the associated Kubernetes object.
	ID() string                        // ID returns a unique identifier for the target.
	Trigger(ctx context.Context) error // Trigger triggers a reload action for the target.
}

// NewTarget creates a new Target based on the provided reference and source object.
func NewTarget(ctx context.Context, c client.Client, kind kinds.Kind, ref string, source client.Object) (Target, error) {
	ns, name, err := utils.ParseTargetRef(ref, source.GetNamespace())
	if err != nil {
		return nil, fmt.Errorf("invalid reference: %w", err)
	}

	switch kind {
	case kinds.DeploymentKind:
		return NewDeployment(ns, name, c), nil
	case kinds.StatefulSetKind:
		return NewStatefulSet(ns, name, c), nil
	case kinds.DaemonSetKind:
		return NewDaemonSet(ns, name, c), nil
	default:
		return nil, fmt.Errorf("unsupported target kind: %s", kind)
	}
}
