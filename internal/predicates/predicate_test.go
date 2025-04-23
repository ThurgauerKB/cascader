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
	"testing"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/utils"
	"github.com/thurgauerkb/cascader/test/testutils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// TestWrapSingleObjectCheck tests the WrapSingleObjectCheck function.
func TestWrapSingleObjectCheck(t *testing.T) {
	t.Parallel()

	checkNamespace := func(obj client.Object) bool {
		return obj.GetNamespace() == "default"
	}

	updateCheck := WrapSingleObjectCheck(checkNamespace)

	t.Run("UpdateCheck - Matching Namespace", func(t *testing.T) {
		t.Parallel()

		newObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}

		result := updateCheck(nil, newObj)
		assert.True(t, result, "UpdateCheck should return true for matching namespace")
	})

	t.Run("UpdateCheck - Non-Matching Namespace", func(t *testing.T) {
		t.Parallel()

		newObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "non-default",
			},
		}

		result := updateCheck(nil, newObj)
		assert.False(t, result, "UpdateCheck should return false for non-matching namespace")
	})

	t.Run("UpdateCheck - Additional Fields", func(t *testing.T) {
		t.Parallel()

		newObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}

		result := updateCheck(nil, newObj)
		assert.True(t, result, "UpdateCheck should return true for matching namespace even with additional fields")
	})
}

// TestNewPredicate tests the behavior of the NewPredicate function.
func TestNewPredicate(t *testing.T) {
	t.Parallel()

	annotationKindMap := kinds.AnnotationKindMap{
		"cascader.tkb.ch/deployment":  kinds.DeploymentKind,
		"cascader.tkb.ch/statefulset": kinds.StatefulSetKind,
		"cascader.tkb.ch/daemonset":   kinds.DaemonSetKind,
	}

	t.Run("UpdateFunc - Object type not equal", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		oldObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}
		newObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}
		event := event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}

		result := predicate.Update(event)
		assert.False(t, result, "UpdateFunc should return false if object are not equal type")
	})

	t.Run("UpdateFunc - Spec Change Detected", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		oldObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}
		newObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: "latest",
							},
						},
					},
				},
			},
		}
		event := event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}

		result := predicate.Update(event)
		assert.True(t, result, "UpdateFunc should return true for spec change")
	})

	t.Run("UpdateFunc - Rollout Restart Detected", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		oldObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.LastObservedRestartKey: "2025-01-14T12:00:00Z",
						},
					},
				},
			},
		}

		newObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.LastObservedRestartKey: "2025-01-14T12:30:00Z",
						},
					},
				},
			},
		}

		event := event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}

		result := predicate.Update(event)
		assert.True(t, result, "UpdateFunc should return true when rollout restart is detected")
	})

	t.Run("UpdateFunc - Missing Annotations", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		oldObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}
		newObj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}
		event := event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}

		result := predicate.Update(event)
		assert.False(t, result, "UpdateFunc should return false for missing annotations")
	})

	t.Run("DeleteFunc - Matching Annotations", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap)

		obj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
		}
		event := event.DeleteEvent{Object: obj}

		result := predicate.Delete(event)
		assert.True(t, result, "DeleteFunc should return true for matching annotations")
	})

	t.Run("DeleteFunc - Non-Matching Annotations", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap)

		obj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}
		event := event.DeleteEvent{Object: obj}

		result := predicate.Delete(event)
		assert.False(t, result, "DeleteFunc should return false for non-matching annotations")
	})

	t.Run("CreateFunc - Always False", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap)

		obj := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/statefulset": "target",
				},
			},
		}
		event := event.CreateEvent{Object: obj}

		result := predicate.Create(event)
		assert.False(t, result, "CreateFunc should always return false")
	})

	t.Run("UpdateFunc - No changes", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		oldObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "target",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}

		newObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "target",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  testutils.DefaultTestImageName,
								Image: testutils.DefaultTestImage,
							},
						},
					},
				},
			},
		}
		event := event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj}

		result := predicate.Update(event)
		assert.False(t, result, "UpdateFunc should return false when no relevant changes are detected")
	})

	t.Run("GenericFunc - Non-Matching Annotations", func(t *testing.T) {
		t.Parallel()

		predicate := NewPredicate(annotationKindMap, SpecChanged)

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Annotations: map[string]string{
					"unwanted": "target",
				},
			},
			Status: appsv1.DeploymentStatus{ObservedGeneration: 2},
		}
		event := event.GenericEvent{Object: obj}

		result := predicate.Generic(event)
		assert.False(t, result, "GenericEvent should return false")
	})
}
