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
	"testing"

	"github.com/thurgauerkb/cascader/internal/targets"
	"github.com/thurgauerkb/cascader/internal/workloads"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTargetIDs(t *testing.T) {
	t.Parallel()

	t.Run("Extract IDs from multiple targets", func(t *testing.T) {
		t.Parallel()

		c := fake.NewClientBuilder().Build()

		input := []targets.Target{
			targets.NewDeployment("ns1", "dep1", c),
			targets.NewStatefulSet("ns1", "sts1", c),
			targets.NewDaemonSet("ns1", "ds1", c),
		}

		expected := []string{"Deployment/ns1/dep1", "StatefulSet/ns1/sts1", "DaemonSet/ns1/ds1"}
		assert.Equal(t, expected, targetIDs(input))
	})

	t.Run("Empty slice returns empty result", func(t *testing.T) {
		t.Parallel()

		var input []targets.Target
		assert.Empty(t, targetIDs(input))
	})
}

func TestHasAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("Annotation exists", func(t *testing.T) {
		t.Parallel()

		mockResource := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"example.com/annotation": "value",
				},
			},
		}

		workload, err := workloads.NewWorkload(mockResource)
		assert.NoError(t, err)

		assert.True(t, hasAnnotation(workload.Resource(), "example.com/annotation"))
	})

	t.Run("Annotation does not exist", func(t *testing.T) {
		t.Parallel()

		mockResource := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"example.com/other": "value",
				},
			},
		}

		workload, err := workloads.NewWorkload(mockResource)
		assert.NoError(t, err)

		assert.False(t, hasAnnotation(workload.Resource(), "example.com/annotation"))
	})

	t.Run("No annotations present", func(t *testing.T) {
		t.Parallel()

		mockResource := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: nil,
			},
		}

		workload, err := workloads.NewWorkload(mockResource)
		assert.NoError(t, err)

		assert.False(t, hasAnnotation(workload.Resource(), "example.com/annotation"))
	})
}
