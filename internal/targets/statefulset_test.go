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
	"github.com/thurgauerkb/cascader/internal/utils"
	"github.com/thurgauerkb/cascader/test/testutils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatefulSetTarget_Methods(t *testing.T) {
	t.Parallel()

	target := NewStatefulSet("default", "test-statefulset", nil)

	t.Run("GetKind", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, kinds.StatefulSetKind, target.Kind(), "GetKind should return the kinds.Kind for StatefulSet")
	})

	t.Run("GetName", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "test-statefulset", target.Name(), "GetName should return the StatefulSet name")
	})

	t.Run("GetNamespace", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "default", target.Namespace(), "GetNamespace should return the StatefulSet namespace")
	})

	t.Run("GetK8sObject", func(t *testing.T) {
		t.Parallel()

		expected := &appsv1.StatefulSet{}
		actual := target.Resource()
		assert.IsType(t, expected, actual, "GetK8sObject should return a StatefulSet object")
	})

	t.Run("GetID", func(t *testing.T) {
		t.Parallel()

		expected := "StatefulSet/default/test-statefulset"
		actual := target.ID()
		assert.Equal(t, expected, actual, "Identifier should return the correct identifier string")
	})
}

func TestStatefulSetTarget_Reload(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Define a valid StatefulSet
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	t.Run("Get Error", func(t *testing.T) {
		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(statefulset).
			Build()

		mockClient := &testutils.MockClientWithError{
			Client: baseClient,
			GetErrorFor: testutils.NamedError{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}

		target := NewStatefulSet(
			"default",
			"test-statefulset",
			mockClient,
		)

		err := target.Trigger(context.TODO())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated get error")
	})

	t.Run("Successful Reload", func(t *testing.T) {
		t.Parallel()

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(statefulset).
			Build()

		target := NewStatefulSet(
			"default",
			"test-statefulset",
			fakeClient,
		)

		err := target.Trigger(context.TODO())
		assert.NoError(t, err)

		updatedStatefulSet := &appsv1.StatefulSet{}
		_ = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: "default", Name: "test-statefulset"}, updatedStatefulSet)

		assert.Contains(t, updatedStatefulSet.Spec.Template.Annotations, utils.RestartedAtKey)
		assert.NotEmpty(t, updatedStatefulSet.Spec.Template.Annotations[utils.RestartedAtKey])
	})

	t.Run("Patch Error", func(t *testing.T) {
		t.Parallel()

		baseClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(statefulset).
			Build()

		mockClient := &testutils.MockClientWithError{
			Client: baseClient,
			PatchErrorFor: testutils.NamedError{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		}

		target := NewStatefulSet(
			"default",
			"test-statefulset",
			mockClient,
		)

		err := target.Trigger(context.TODO())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated patch error")
	})
}
