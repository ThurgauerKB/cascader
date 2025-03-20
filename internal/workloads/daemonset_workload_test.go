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

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDaemonSetWorkload_Methods(t *testing.T) {
	t.Parallel()

	t.Run("Get DaemonSet GetResource", func(t *testing.T) {
		t.Parallel()

		ds := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		}
		ds.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		})
		dsw := DaemonSetWorkload{DaemonSet: ds}

		assert.Equal(t, ds, dsw.Resource(), "GetKind should return the DaemonSet")
	})

	t.Run("Get DaemonSet GetKind", func(t *testing.T) {
		t.Parallel()

		ds := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		}
		ds.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		})
		dsw := DaemonSetWorkload{DaemonSet: ds}

		assert.Equal(t, kinds.DaemonSetKind, dsw.Kind(), "GetKind should return the DaemonSet")
	})

	t.Run("Get DaemonSet GetID", func(t *testing.T) {
		t.Parallel()

		ds := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		}
		ds.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		})
		dsw := DaemonSetWorkload{DaemonSet: ds}

		assert.Equal(t, "DaemonSet/default/test-daemonset", dsw.ID(), "GetID should return the DaemonSet/default/test-daemonset name")
	})
}

func TestDaemonSetWorkload_IsStable(t *testing.T) {
	t.Parallel()

	t.Run("Stable DaemonSet", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				NumberAvailable:        4,
				UpdatedNumberScheduled: 4,
				NumberReady:            4,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 4,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "workload is stable: ready=4, desired=4", msg)
	})

	t.Run("DaemonSet with different Generation", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				NumberAvailable:        4,
				UpdatedNumberScheduled: 4,
				NumberReady:            4,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 4,
				ObservedGeneration:     1,
			},
			ObjectMeta: metav1.ObjectMeta{
				Generation: 2,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "rollout in progress: observedGeneration=1, generation=2", msg)
	})

	t.Run("DaemonSet with Not enough Available Replicas", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				NumberAvailable:        3,
				UpdatedNumberScheduled: 4,
				NumberReady:            4,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 4,
				ObservedGeneration:     2,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not enough available replicas: available=3, desired=4", msg)
	})

	t.Run("DaemonSet with Not All Updated Replicas", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 3,
				NumberReady:            3,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 4,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not all replicas are updated: updated=3, ready=3, desired=4", msg)
	})

	t.Run("DaemonSet with Not Enough Ready Replicas", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 4,
				NumberReady:            3,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 4,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "not enough ready replicas: ready=3, desired=4", msg)
	})

	t.Run("DaemonSet with Unavailable Replicas", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 4,
				NumberReady:            3,
				NumberUnavailable:      1,
				DesiredNumberScheduled: 4,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.False(t, isStable)
		assert.Equal(t, "unavailable replicas: available=1, ready=3, desired=4", msg)
	})

	t.Run("Scaled to Zero DaemonSet", func(t *testing.T) {
		t.Parallel()

		daemonset := &appsv1.DaemonSet{
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: 0,
				NumberReady:            0,
				NumberUnavailable:      0,
				DesiredNumberScheduled: 0,
			},
		}

		workload := DaemonSetWorkload{DaemonSet: daemonset}
		isStable, msg := workload.Stable()

		assert.True(t, isStable)
		assert.Equal(t, "scaled to zero replicas", msg)
	})
}
