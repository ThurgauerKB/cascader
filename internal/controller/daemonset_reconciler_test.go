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
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/testutils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestDaemonSetReconciler_SetupWithManager(t *testing.T) {
	t.Parallel()

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{})
	assert.NoError(t, err, "Failed to create manager")

	reconciler := &DaemonSetReconciler{
		BaseReconciler: BaseReconciler{
			KubeClient: fake.NewClientBuilder().WithScheme(mgr.GetScheme()).Build(),
			AnnotationKindMap: kinds.AnnotationKindMap{
				"cascader.tkb.ch/daemonset": kinds.DaemonSetKind,
			},
		},
	}

	err = reconciler.SetupWithManager(mgr)
	assert.NoError(t, err, "SetupWithManager should not return an error")
}

func TestDaemonSetReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("DaemonSet not found", func(t *testing.T) {
		t.Parallel()

		// Fake client with no resources
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler := &DaemonSetReconciler{
			BaseReconciler: BaseReconciler{
				KubeClient: fakeClient,
			},
		}

		req := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "nonexistent-daemonset",
		}}

		result, err := reconciler.Reconcile(context.Background(), req)
		assert.NoError(t, err, "Expected no error when DaemonSet is not found")
		assert.Equal(t, ctrl.Result{}, result, "Expected empty result when DaemonSet is not found")
	})

	t.Run("Error fetching DaemonSet", func(t *testing.T) {
		t.Parallel()

		fakeBaseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeClient := &testutils.MockClientWithError{
			Client:      fakeBaseClient,
			GetErrorFor: testutils.NamedError{Name: "error-daemonset", Namespace: "test-namespace"},
		}

		reconciler := &DaemonSetReconciler{
			BaseReconciler: BaseReconciler{
				KubeClient: fakeClient,
			},
		}

		req := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "error-daemonset",
		}}

		result, err := reconciler.Reconcile(context.Background(), req)
		assert.Error(t, err, "Expected error when Get fails")
		assert.Contains(t, err.Error(), "failed to fetch DaemonSet")
		assert.Equal(t, ctrl.Result{}, result, "Expected empty result when Get fails")
	})

	t.Run("Successful Reconciliation", func(t *testing.T) {
		t.Parallel()

		// Fake DaemonSet object with TypeMeta set
		daemonset := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "test-namespace",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(daemonset).Build()

		reconciler := &DaemonSetReconciler{
			BaseReconciler: BaseReconciler{
				Logger:     &logr.Logger{},
				KubeClient: fakeClient,
			},
		}

		req := ctrl.Request{NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-daemonset",
		}}

		result, err := reconciler.Reconcile(context.Background(), req)
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		assert.Equal(t, ctrl.Result{}, result, "Expected successful result")
	})
}
