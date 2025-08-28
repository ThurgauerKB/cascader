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
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadFactory_NewWorkload(t *testing.T) {
	t.Parallel()

	t.Run("Create Deployment Workload", func(t *testing.T) {
		t.Parallel()

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}

		workload, err := NewWorkload(deployment)
		require.NoError(t, err)
		require.NotNil(t, workload)

		assert.IsType(t, &DeploymentWorkload{}, workload)
		assert.Equal(t, kinds.DeploymentKind, workload.Kind())
		assert.Equal(t, "Deployment/default/test-deployment", workload.ID())
	})

	t.Run("Create StatefulSet Workload", func(t *testing.T) {
		t.Parallel()

		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}

		workload, err := NewWorkload(statefulSet)
		require.NoError(t, err)
		require.NotNil(t, workload)

		assert.IsType(t, &StatefulSetWorkload{}, workload)
		assert.Equal(t, kinds.StatefulSetKind, workload.Kind())
		assert.Equal(t, "StatefulSet/default/test-statefulset", workload.ID())
	})

	t.Run("Create DaemonSet Workload", func(t *testing.T) {
		t.Parallel()

		daemonSet := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		}

		workload, err := NewWorkload(daemonSet)
		require.NoError(t, err)
		require.NotNil(t, workload)

		assert.IsType(t, &DaemonSetWorkload{}, workload)
		assert.Equal(t, kinds.DaemonSetKind, workload.Kind())
		assert.Equal(t, "DaemonSet/default/test-daemonset", workload.ID())
	})

	t.Run("Unsupported Workload Type", func(t *testing.T) {
		t.Parallel()

		unsupportedObj := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replicaset",
				Namespace: "default",
			},
		}

		workload, err := NewWorkload(unsupportedObj)
		require.Error(t, err)
		assert.Nil(t, workload)
		assert.Contains(t, err.Error(), "unsupported workload type")
	})
}
