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

package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/thurgauerkb/cascader/internal/targets"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CycleKind represents the type of the detected dependency cycle.
// Indicates whether the cycle is direct or indirect.
type CycleKind string

const (
	// DirectKind indicates a cycle where a resource directly depends on itself.
	DirectKind CycleKind = "direct"

	// IndirectKind indicates a cycle formed through a chain of dependencies involving multiple resources.
	IndirectKind CycleKind = "indirect"
)

// CycleError encapsulates details about a detected dependency cycle.
type CycleError struct {
	Kind     CycleKind // Type of the cycle: direct or indirect
	SourceID string    // Identifier of the resource initiating the detection (format: Kind/Namespace/Name)
	Path     string    // Sequence of resources forming the cycle (e.g., "A -> B -> A")
}

// Error returns a descriptive error message for the CycleError.
func (e *CycleError) Error() string {
	return fmt.Sprintf("%s cycle detected: adding dependency from %s creates a cycle: %s", e.Kind, e.SourceID, e.Path)
}

// checkCycle checks for circular dependencies among targets.
// Returns an error if a cycle is found.
func (b *BaseReconciler) checkCycle(ctx context.Context, srcID string, targets []targets.Target) error {
	for _, target := range targets {
		// Direct cycle detection
		if target.ID() == srcID {
			return &CycleError{
				Kind:     DirectKind,
				SourceID: srcID,
				Path:     target.ID(),
			}
		}

		// Recursively detect cycles
		found, cycle, err := b.detectCycle(ctx, target, srcID, []string{srcID})
		if err != nil {
			return fmt.Errorf("dependency cycle check failed: %w", err)
		}

		if found {
			return &CycleError{
				Kind:     IndirectKind,
				SourceID: srcID,
				Path:     strings.Join(cycle, " -> "),
			}
		}
	}

	return nil
}

// detectCycle recursively walks dependencies to detect cycles.
// Returns true if a cycle is found, along with the full path forming the cycle.
func (b *BaseReconciler) detectCycle(ctx context.Context, target targets.Target, srcID string, path []string) (found bool, cycle []string, err error) {
	targetID := target.ID()

	// Detect if target already in path → cycle found
	if slices.Contains(path, targetID) {
		return true, append(path, targetID), nil
	}

	path = append(path, targetID)

	// Fetch target resource
	res := target.Resource()
	if err := b.KubeClient.Get(ctx, client.ObjectKey{Namespace: target.Namespace(), Name: target.Name()}, res); err != nil {
		return false, nil, fmt.Errorf("failed to fetch resource %s: %w", targetID, err)
	}

	// Extract dependencies from resource
	dependencies, err := b.extractTargets(ctx, res)
	if err != nil {
		return false, nil, fmt.Errorf("error extracting dependencies: %w", err)
	}

	// Recursively check dependencies
	for _, dependency := range dependencies {
		if dependency.ID() == srcID {
			// Cycle detected, append the source to complete the cycle path
			return true, append(path, srcID), nil
		}

		// Recursively check dependencies with the updated path
		found, cycle, err = b.detectCycle(ctx, dependency, srcID, path)
		if err != nil {
			return found, nil, err
		}
		if found {
			return true, cycle, nil
		}
	}

	return false, nil, nil
}
