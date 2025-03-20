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

func TestStatefulSetWorkload_Methosts(t *testing.T) {
	t.Parallel()

	t.Run("Get StatefulSet GetResource", func(t *testing.T) {
		t.Parallel()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}
		sts.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "StatefulSet",
		})
		stsw := StatefulSetWorkload{StatefulSet: sts}

		assert.Equal(t, sts, stsw.Resource(), "GetKind should return the StatefulSet")
	})

	t.Run("Get StatefulSet GetKind", func(t *testing.T) {
		t.Parallel()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}
		sts.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "StatefulSet",
		})
		stsw := StatefulSetWorkload{StatefulSet: sts}

		assert.Equal(t, kinds.StatefulSetKind, stsw.Kind(), "GetKind should return the StatefulSet")
	})

	t.Run("Get StatefulSet GetID", func(t *testing.T) {
		t.Parallel()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}
		sts.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "StatefulSet",
		})
		stsw := StatefulSetWorkload{StatefulSet: sts}

		assert.Equal(t, "StatefulSet/default/test-statefulset", stsw.ID(), "GetID should return the StatefulSet/default/test-statefulset name")
	})
}

func TestStatefulSetWorkload_IsStable(t *testing.T) {
	t.Parallel()

	t.Run("Stable StatefulSet", func(t *testing.T) {
		t.Parallel()

		statefulset := &appsv1.StatefulSet{
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas: 4,
				ReadyReplicas:   4,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := StatefulSetWorkload{StatefulSet: statefulset}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "workload is stable: ready=4, desired=4", msg)
	})

	t.Run("StatefulSet with different Generation", func(t *testing.T) {
		t.Parallel()

		statefulset := &appsv1.StatefulSet{
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas:    2,
				ReadyReplicas:      4,
				ObservedGeneration: 1,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(4),
			},
			ObjectMeta: metav1.ObjectMeta{
				Generation: 2,
			},
		}
		workload := StatefulSetWorkload{StatefulSet: statefulset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "rollout in progress: observedGeneration=1, generation=2", msg)
	})

	t.Run("StatefulSet with Not All Updated Replicas", func(t *testing.T) {
		t.Parallel()

		statefulset := &appsv1.StatefulSet{
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas: 2,
				ReadyReplicas:   4,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := StatefulSetWorkload{StatefulSet: statefulset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not all replicas are updated: updated=2, ready=4, desired=4", msg)
	})

	t.Run("StatefulSet with Not Enough Ready Replicas", func(t *testing.T) {
		t.Parallel()

		statefulset := &appsv1.StatefulSet{
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas: 4,
				ReadyReplicas:   3,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		workload := StatefulSetWorkload{StatefulSet: statefulset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not enough ready replicas: ready=3, desired=4", msg)
	})

	t.Run("Scaled to Zero StatefulSet", func(t *testing.T) {
		t.Parallel()

		statefulset := &appsv1.StatefulSet{
			Status: appsv1.StatefulSetStatus{
				UpdatedReplicas: 0,
				ReadyReplicas:   0,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: testutils.Int32Ptr(0),
			},
		}
		workload := StatefulSetWorkload{StatefulSet: statefulset}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "scaled to zero replicas", msg)
	})
}
