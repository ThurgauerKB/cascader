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
	"github.com/thurgauerkb/cascader/internal/testutils"
	"github.com/thurgauerkb/cascader/internal/utils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("Object has desired annotation", func(t *testing.T) {
		t.Parallel()

		obj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"desired-annotation": "value",
				},
			},
		}

		annotationKindMap := kinds.AnnotationKindMap{
			"desired-annotation": "",
		}

		result := hasAnnotation(obj, annotationKindMap)
		assert.True(t, result, "Expected hasAnnotation to return true for matching annotation")
	})

	t.Run("Object does not have desired annotation", func(t *testing.T) {
		t.Parallel()

		obj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"other-annotation": "value",
				},
			},
		}

		annotationKindMap := kinds.AnnotationKindMap{
			"desired-annotation": "",
		}

		result := hasAnnotation(obj, annotationKindMap)
		assert.False(t, result, "Expected hasAnnotation to return false for non-matching annotation")
	})

	t.Run("Object has no annotations", func(t *testing.T) {
		t.Parallel()

		obj := &corev1.Pod{}
		annotationKindMap := kinds.AnnotationKindMap{
			"desired-annotation": "",
		}

		result := hasAnnotation(obj, annotationKindMap)
		assert.False(t, result, "Expected hasAnnotation to return false for objects with no annotations")
	})
}

func TestSpecChanged(t *testing.T) {
	t.Parallel()

	t.Run("Spec changes detected", func(t *testing.T) {
		t.Parallel()

		oldDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.20"},
						},
					},
				},
			},
		}

		newDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.21"},
						},
					},
				},
			},
		}

		result := SpecChanged(oldDep, newDep)
		assert.True(t, result, "Expected isSpecChange to return true for spec changes")
	})

	t.Run("No spec changes", func(t *testing.T) {
		t.Parallel()

		oldDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.21"},
						},
					},
				},
			},
		}

		newDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.21"},
						},
					},
				},
			},
		}

		result := SpecChanged(oldDep, newDep)
		assert.False(t, result, "Expected isSpecChange to return false for no spec changes")
	})

	t.Run("Error in extracting PodTemplate", func(t *testing.T) {
		t.Parallel()

		oldPod := &corev1.Pod{}
		newPod := &corev1.Pod{}

		result := SpecChanged(oldPod, newPod)
		assert.False(t, result, "Expected isSpecChange to return false when template extraction fails")
	})
}

func TestRestartAnnotationChanged(t *testing.T) {
	t.Parallel()

	t.Run("No change in restartedAt", func(t *testing.T) {
		t.Parallel()

		restartedAt := "2025-02-22T12:00:00Z"
		depOld := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.RestartedAtKey: restartedAt,
						},
					},
				},
			},
		}

		// Deep copy to simulate an updated object with the same annotation.
		depNew := depOld.DeepCopy()

		changed := RestartAnnotationChanged(depOld, depNew)
		assert.False(t, changed, "expected false when restartedAt annotation is unchanged")
	})

	t.Run("Changed restartedAt annotation", func(t *testing.T) {
		t.Parallel()

		depOld := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.RestartedAtKey: "2025-02-22T12:00:00Z",
						},
					},
				},
			},
		}

		depNew := depOld.DeepCopy()
		depNew.Spec.Template.Annotations[utils.RestartedAtKey] = "2025-02-22T12:05:00Z"

		changed := RestartAnnotationChanged(depOld, depNew)
		assert.True(t, changed, "expected true when restartedAt annotation has changed")
	})

	t.Run("Missing restartedAt annotation", func(t *testing.T) {
		t.Parallel()

		// Both objects without the restartedAt annotation.
		depOld := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
				},
			},
		}

		depNew := depOld.DeepCopy()

		changed := RestartAnnotationChanged(depOld, depNew)
		assert.False(t, changed, "expected false when restartedAt annotation missing on both")

		// New object now has the restartedAt annotation.
		depNew.Spec.Template.Annotations = map[string]string{
			utils.RestartedAtKey: "2025-02-22T12:05:00Z",
		}

		changed = RestartAnnotationChanged(depOld, depNew)
		assert.True(t, changed, "expected true when restartedAt annotation is added")
	})

	t.Run("Unsupported object type", func(t *testing.T) {
		t.Parallel()

		// Using a Pod, which is unsupported by extractPodTemplate.
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					utils.RestartedAtKey: "2025-02-22T12:00:00Z",
				},
			},
		}
		pod2 := pod.DeepCopy()

		changed := RestartAnnotationChanged(pod, pod2)
		assert.False(t, changed, "expected false for unsupported object type")
	})
}

func TestExtractPodTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Deployment - Successful Extraction", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{},
			},
		}

		template, err := extractPodTemplate(dep)
		assert.NoError(t, err, "Expected no error for valid Deployment")
		assert.NotNil(t, template, "Expected non-nil template for Deployment")
	})

	t.Run("StatefulSet - Successful Extraction", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{},
			},
		}

		template, err := extractPodTemplate(dep)
		assert.NoError(t, err, "Expected no error for valid StatefulSet")
		assert.NotNil(t, template, "Expected non-nil template for StatefulSet")
	})

	t.Run("DaemonSet - Successful Extraction", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.DaemonSet{
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{},
			},
		}

		template, err := extractPodTemplate(dep)
		assert.NoError(t, err, "Expected no error for valid DaemonSet")
		assert.NotNil(t, template, "Expected non-nil template for DaemonSet")
	})

	t.Run("Unsupported Object Type", func(t *testing.T) {
		t.Parallel()

		pod := &corev1.Pod{}

		template, err := extractPodTemplate(pod)
		assert.Error(t, err, "Expected error for unsupported object type")
		assert.Contains(t, err.Error(), "unsupported object type", "Expected specific error message for unsupported type")
		assert.Nil(t, template, "Expected nil template for unsupported type")
	})
}

func TestHashTemplate(t *testing.T) {
	t.Parallel()

	t.Run("Generate hash for valid PodTemplateSpec", func(t *testing.T) {
		t.Parallel()

		template := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"annotation": "value",
				},
			},
		}

		hash, err := hashTemplate(template)
		assert.NoError(t, err, "Expected no error for valid PodTemplateSpec")
		assert.NotEmpty(t, hash, "Expected non-empty hash for valid PodTemplateSpec")
	})

	t.Run("Error on marshaling failure", func(t *testing.T) {
		t.Parallel()

		template := corev1.PodTemplateSpec{}
		template.ObjectMeta.Annotations = nil // Ensure no annotations

		hash, err := hashTemplate(template)
		assert.NoError(t, err, "Expected no error for valid PodTemplateSpec without annotations")
		assert.NotEmpty(t, hash, "Expected non-empty hash for valid PodTemplateSpec without annotations")
	})
}

func TestDaemonSetTransitioning(t *testing.T) {
	t.Parallel()

	t.Run("DaemonSet in transition", func(t *testing.T) {
		t.Parallel()

		ds := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 2,
				DesiredNumberScheduled: 3,
				NumberUnavailable:      1,
			},
		}

		result := DaemonSetTransitioning(ds)
		assert.True(t, result, "Expected isDaemonSetInTransition to return true for a DaemonSet in transition")
	})

	t.Run("DaemonSet not in transition", func(t *testing.T) {
		t.Parallel()

		ds := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 3,
				DesiredNumberScheduled: 3,
				NumberUnavailable:      0,
			},
		}

		result := DaemonSetTransitioning(ds)
		assert.False(t, result, "Expected isDaemonSetInTransition to return false for a stable DaemonSet")
	})

	t.Run("Invalid object type", func(t *testing.T) {
		t.Parallel()

		obj := &corev1.Pod{}
		result := DaemonSetTransitioning(obj)
		assert.False(t, result, "Expected isDaemonSetInTransition to return false for an invalid object type")
	})
}

func TestSingleReplicaPodDeleted(t *testing.T) {
	t.Parallel()

	t.Run("Deployment: Single replica pod deleted", func(t *testing.T) {
		t.Parallel()

		oldDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		}

		newDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     0,
				AvailableReplicas: 0,
			},
		}

		result := SingleReplicaPodDeleted(oldDep, newDep)
		assert.True(t, result, "Expected isSingleReplicaPodDeleted to return true for a deleted pod in a single-replica Deployment")
	})

	t.Run("StatefulSet: Single replica pod deleted", func(t *testing.T) {
		t.Parallel()

		oldDep := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		}

		newDep := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:     0,
				AvailableReplicas: 0,
			},
		}

		result := SingleReplicaPodDeleted(oldDep, newDep)
		assert.True(t, result, "Expected isSingleReplicaPodDeleted to return true for a deleted pod in a single-replica StatefulSet")
	})

	t.Run("No replicas deleted", func(t *testing.T) {
		t.Parallel()

		oldDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		}

		newDep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		}

		result := SingleReplicaPodDeleted(oldDep, newDep)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false when no pods are deleted")
	})

	t.Run("Invalid object type", func(t *testing.T) {
		t.Parallel()

		oldObj := &corev1.Pod{}
		newObj := &corev1.Pod{}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("Invalid new object type (old Deployment)", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.Deployment{}
		newObj := &corev1.Pod{}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("Deployment - invalid replica (nil)", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.Deployment{}
		newObj := &appsv1.Deployment{}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("Deployment - replica != 1", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}
		newObj := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("Invalid new object type (old StatefulSet)", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.StatefulSet{}
		newObj := &corev1.Pod{}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("StatefulSet - invalid replica (nil)", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.StatefulSet{}
		newObj := &appsv1.StatefulSet{}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})

	t.Run("StatefulSet - replica != 1", func(t *testing.T) {
		t.Parallel()

		oldObj := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}
		newObj := &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		result := SingleReplicaPodDeleted(oldObj, newObj)
		assert.False(t, result, "Expected isSingleReplicaPodDeleted to return false for invalid object types")
	})
}

func TestScaledToZero(t *testing.T) {
	t.Parallel()

	t.Run("Deployment scaled to zero", func(t *testing.T) {
		t.Parallel()

		oldDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		newDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}

		assert.True(t, ScaledToZero(oldDeployment, newDeployment))
	})

	t.Run("StatefulSet scaled to zero", func(t *testing.T) {
		t.Parallel()

		oldSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-statefuset", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		newSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-statefuset", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}

		assert.True(t, ScaledToZero(oldSts, newSts))
	})

	t.Run("Deamonset not scaled to zero", func(t *testing.T) {
		t.Parallel()

		oldDS := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ds", Namespace: "default"},
			Spec:       appsv1.DaemonSetSpec{},
			Status: appsv1.DaemonSetStatus{
				NumberReady: 1,
			},
		}

		newDS := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ds", Namespace: "default"},
			Spec:       appsv1.DaemonSetSpec{},
			Status: appsv1.DaemonSetStatus{
				NumberReady: 0,
			},
		}

		assert.False(t, ScaledToZero(oldDS, newDS))
	})

	t.Run("Type mismatch", func(t *testing.T) {
		t.Parallel()

		oldDaemonSet := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-daemonset", Namespace: "default"},
		}

		newSTS := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(2),
			},
		}

		assert.False(t, ScaledToZero(oldDaemonSet, newSTS))
	})
}

func TestScaledFromZero(t *testing.T) {
	t.Parallel()

	t.Run("Deployment scaled from zero", func(t *testing.T) {
		t.Parallel()

		oldDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}

		newDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		assert.True(t, ScaledFromZero(oldDeployment, newDeployment))
	})

	t.Run("StatefulSet scaled from zero", func(t *testing.T) {
		t.Parallel()

		oldSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-statefuset", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}

		newSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-statefuset", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		assert.True(t, ScaledFromZero(oldSts, newSts))
	})

	t.Run("Deamonset not scaled from zero", func(t *testing.T) {
		t.Parallel()

		oldDS := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ds", Namespace: "default"},
			Spec:       appsv1.DaemonSetSpec{},
			Status: appsv1.DaemonSetStatus{
				NumberReady: 0,
			},
		}

		newDS := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ds", Namespace: "default"},
			Spec:       appsv1.DaemonSetSpec{},
			Status: appsv1.DaemonSetStatus{
				NumberReady: 1,
			},
		}

		assert.False(t, ScaledFromZero(oldDS, newDS))
	})

	t.Run("StatefulSet not scaled from zero", func(t *testing.T) {
		t.Parallel()

		oldSTS := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(1),
			},
		}

		newSTS := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(2),
			},
		}

		assert.False(t, ScaledFromZero(oldSTS, newSTS))
	})

	t.Run("Type mismatch", func(t *testing.T) {
		t.Parallel()

		oldRs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-replicaset", Namespace: "default"},
		}

		newSTS := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(2),
			},
		}

		assert.False(t, ScaledFromZero(oldRs, newSTS))
	})
}

func TestGetReplicas(t *testing.T) {
	t.Parallel()

	t.Run("Get replicas from Deployment", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		replicas := getReplicas(deployment)
		assert.Equal(t, int32(3), replicas)
	})

	t.Run("Get replicas from StatefulSet", func(t *testing.T) {
		t.Parallel()

		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(2),
			},
		}

		replicas := getReplicas(statefulSet)
		assert.Equal(t, int32(2), replicas)
	})

	t.Run("Unsupported type", func(t *testing.T) {
		t.Parallel()

		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-replicaset", Namespace: "default"},
		}

		replicas := getReplicas(rs)
		assert.Equal(t, int32(-1), replicas)
	})
}
