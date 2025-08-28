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
	"testing"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/test/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateTarget(t *testing.T) {
	t.Parallel()

	mockClient := new(testutils.MockClientWithError)

	t.Run("Invalid target reference", func(t *testing.T) {
		t.Parallel()

		origin := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}

		target, err := NewTarget(context.TODO(), mockClient, kinds.DeploymentKind, "not/valid/reference", origin)

		assert.Nil(t, target, "Expected nil target for for invalid reference")
		require.Error(t, err, "Expected error for invalid reference")
	})

	t.Run("Valid Deployment Target", func(t *testing.T) {
		t.Parallel()

		origin := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		}

		target, err := NewTarget(context.TODO(), mockClient, kinds.DeploymentKind, "default/test-deployment", origin)

		assert.NoError(t, err, "Expected no error for Deployment target creation")
		assert.NotNil(t, target, "Expected a non-nil target for Deployment")
	})

	t.Run("Valid StatefulSet Target", func(t *testing.T) {
		t.Parallel()

		origin := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}

		target, err := NewTarget(context.TODO(), mockClient, kinds.StatefulSetKind, "default/test-statefulset", origin)

		assert.NoError(t, err, "Expected no error for StatefulSet target creation")
		assert.NotNil(t, target, "Expected a non-nil target for StatefulSet")
	})

	t.Run("Valid DaemonSet Target", func(t *testing.T) {
		t.Parallel()

		origin := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		}

		target, err := NewTarget(context.TODO(), mockClient, kinds.DaemonSetKind, "default/test-daemonset", origin)

		assert.NoError(t, err, "Expected no error for DaemonSet target creation")
		assert.NotNil(t, target, "Expected a non-nil target for DaemonSet")
	})

	t.Run("Unsupported Workload Type", func(t *testing.T) {
		t.Parallel()

		origin := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replicaset",
				Namespace: "default",
			},
		}

		target, err := NewTarget(context.TODO(), mockClient, "ReplicaSet", "default/test-replicaset", origin)

		require.Error(t, err, "Expected error for unsupported workload type")
		assert.Nil(t, target, "Expected a nil target for unsupported workload type")
		assert.EqualError(t, err, "unsupported target kind: ReplicaSet")
	})
}
