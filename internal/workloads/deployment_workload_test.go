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

package workloads

import (
	"testing"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/testutils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDeploymentWorkload_Methods(t *testing.T) {
	t.Parallel()

	t.Run("Get Deployment GetResource", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}
		dep.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})
		depw := DeploymentWorkload{Deployment: dep}

		assert.Equal(t, dep, depw.Resource(), "GetKind should return the Deployment")
	})

	t.Run("Get Deployment GetKind", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}
		dep.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})
		depw := DeploymentWorkload{Deployment: dep}

		assert.Equal(t, kinds.DeploymentKind, depw.Kind(), "GetKind should return the Deployment")
	})

	t.Run("Get Deployment GetID", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}
		dep.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})
		depw := DeploymentWorkload{Deployment: dep}

		assert.Equal(t, "Deployment/default/test-deployment", depw.ID(), "GetID should return the Deployment/default/test-deployment name")
	})
}

func TestDeploymentWorkload_IsStable(t *testing.T) {
	t.Parallel()

	t.Run("Stable Deployment", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     4,
				ReadyReplicas:       4,
				UnavailableReplicas: 0,
				AvailableReplicas:   4,
				ObservedGeneration:  2,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "workload is stable: ready=4, desired=4", msg)
	})

	t.Run("Deployment with different Generation", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     4,
				ReadyReplicas:       2,
				UnavailableReplicas: 2,
				ObservedGeneration:  1,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
			ObjectMeta: metav1.ObjectMeta{
				Generation: 2,
			},
		}

		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "rollout in progress: observedGeneration=1, generation=2", msg)
	})

	t.Run("Deployment with not enought Available Replicas", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:   3,
				UpdatedReplicas:     4,
				ReadyReplicas:       4,
				UnavailableReplicas: 0,
				ObservedGeneration:  2,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not enough available replicas: available=3, desired=4", msg)
	})

	t.Run("Deployment with Unavailable Replicas", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     4,
				ReadyReplicas:       2,
				UnavailableReplicas: 2,
				ObservedGeneration:  2,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "unavailable replicas: unavailable=2, ready=2, desired=4", msg)
	})

	t.Run("Deployment with Not All Updated Replicas", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     2,
				ReadyReplicas:       2,
				UnavailableReplicas: 0,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not all replicas are updated: updated=2, ready=2, desired=4", msg)
	})

	t.Run("Deployment with Not Enough Ready Replicas", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     4,
				ReadyReplicas:       3,
				UnavailableReplicas: 0,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not enough ready replicas: ready=3, desired=4", msg)
	})

	t.Run("Scaled to Zero Replicas", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				UpdatedReplicas:     0,
				ReadyReplicas:       0,
				UnavailableReplicas: 0,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}
		workload := DeploymentWorkload{Deployment: deployment}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "scaled to zero replicas", msg)
	})
}
