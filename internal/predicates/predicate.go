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

package predicates

import (
	"github.com/thurgauerkb/cascader/internal/kinds"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// UpdateCheck defines a function type for additional update checks.
type UpdateCheck func(oldObj, newObj client.Object) bool

// SingleObjectCheck defines a function type for single-object checks.
type SingleObjectCheck func(obj client.Object) bool

// WrapSingleObjectCheck adapts a single-object check into an UpdateCheck.
func WrapSingleObjectCheck(check SingleObjectCheck) UpdateCheck {
	return func(_, newObj client.Object) bool {
		return check(newObj)
	}
}

// NewPredicate creates a predicate with default behavior and allows adding custom update checks.
func NewPredicate(annotations kinds.AnnotationKindMap, updateChecks ...UpdateCheck) predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Check for required annotations.
			if !hasAnnotation(e.ObjectNew, annotations) {
				return false
			}

			// Run additional update checks.
			for _, check := range updateChecks {
				if check(e.ObjectOld, e.ObjectNew) {
					return true
				}
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Check only for required annotations.
			return e.Object != nil && hasAnnotation(e.Object, annotations)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// Skip create events to avoid unnecessary reloads during resource creation.
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Generic events are not used in this reconciler.
			return false
		},
	}
}
