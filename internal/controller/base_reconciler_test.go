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
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/targets"
	"github.com/thurgauerkb/cascader/internal/testutils"
	"github.com/thurgauerkb/cascader/internal/utils"
	"github.com/thurgauerkb/cascader/internal/workloads"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defaultRequeuAfter              time.Duration = 10 * time.Second
	workloadStableMsg               string        = "Workload is stable"
	successfullTriggerTargetMsg     string        = "Successfully triggered reload"
	successfullTriggerAllTargetsMsg string        = "Triggered reloads"
	failedTriggerTargetMsg          string        = "Some targets failed to reload"
)

// Helper function to create a fake BaseReconciler
func createBaseReconciler(objects ...client.Object) *BaseReconciler {
	fakeClient := fake.NewClientBuilder().WithObjects(objects...).Build()
	return &BaseReconciler{
		Logger:                 &logr.Logger{},
		KubeClient:             fakeClient,
		Recorder:               record.NewFakeRecorder(10),
		LastObservedRestartKey: "cascader.tkb.ch/last-observed-restart",
		RequeueAfterAnnotation: "cascader.tkb.ch/requeueAfter",
		RequeueAfterDefault:    defaultRequeuAfter,
		AnnotationKindMap: kinds.AnnotationKindMap{
			"cascader.tkb.ch/deployment":  kinds.DeploymentKind,
			"cascader.tkb.ch/statefulset": kinds.StatefulSetKind,
			"cascader.tkb.ch/daemonset":   kinds.DaemonSetKind,
		},
	}
}

func TestBaseReconciler_ReconcileWorkload(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("Invalid Workload Kind", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicaSet",
				APIVersion: "apps/v1",
			},
		}
		obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "ReplicSet",
		})

		reconciler := createBaseReconciler()

		_, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.Error(t, err, "Expected error for invalid workload kind")
		assert.Contains(t, err.Error(), "unsupported workload type: *v1.ReplicaSet", "Expected error message to indicate invalid ReplicaSet")
	})

	t.Run("Invalid Target annotation", func(t *testing.T) {
		dep1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		dep1.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		targetObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})

		reconciler := createBaseReconciler(dep1, targetObj)

		_, err := reconciler.ReconcileWorkload(context.Background(), dep1)
		assert.Error(t, err, "Expected error on successful reconciliation")
		assert.EqualError(t, err, "failed to create targets: targets cannot be empty")
	})

	t.Run("Successful Reconciliation (use invalid requeue duration)", func(t *testing.T) {
		t.Parallel()

		dep1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment":   "test-namespace/another-deployment",
					"cascader.tkb.ch/requeueAfter": "invalid",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		dep1.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}
		targetObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		})

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer)) // Set up a zap logger that writes to logBuffer

		reconciler := createBaseReconciler(dep1, targetObj)
		reconciler.Logger = &logger

		result, err := reconciler.ReconcileWorkload(context.Background(), dep1)
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{RequeueAfter: defaultRequeuAfter}
		assert.Equal(t, expectedResult, result, "Expected successful result with default requeue duration")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "invalid ", "Expected log to contain message about invalid requeue duration")
	})

	t.Run("Successful Reconciliation (use 0s requeue duration)", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment":   "test-namespace/another-deployment",
					"cascader.tkb.ch/requeueAfter": "0s",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer)) // Set up a zap logger that writes to logBuffer

		reconciler := createBaseReconciler(obj, targetObj)
		reconciler.Logger = &logger

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result with default requeue duration")
	})

	t.Run("No Workload Targets", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
		}

		reconciler := createBaseReconciler(obj)

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.NoError(t, err, "Expected no error when no workload targets are found")
		assert.Equal(t, ctrl.Result{}, result, "Expected empty result when no workload targets are found")
	})

	t.Run("Dependency Cycle Detected - Self-Referential", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-namespace/test-deployment",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		reconciler := createBaseReconciler(obj)

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		expectedResult := ctrl.Result{RequeueAfter: 0}
		assert.Equal(t, expectedResult, result, "Expected successful result with default requeue duration")
		assert.Error(t, err, "Expected error for self-referential dependency cycle")
		assert.EqualError(t, err, "dependency cycle detected: direct cycle detected: adding dependency from Deployment/test-namespace/test-deployment creates a cycle: Deployment/test-namespace/test-deployment")
	})

	t.Run("Successful Reconciliation - Workload is stable", func(t *testing.T) {
		t.Parallel()

		restartedAt := "2024-04-03T12:00:00Z"

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-namespace/another-deployment",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:  4,
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.RestartedAtKey: restartedAt,
						},
					},
				},
			},
		}

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "another-deployment",
				Namespace:  "test-namespace",
				Generation: 6,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:  4,
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer)) // Set up a zap logger that writes to logBuffer

		reconciler := createBaseReconciler(obj, targetObj)
		reconciler.Logger = &logger

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "Restart detected", "Expected log to contain message about restart detected")
		assert.Contains(t, logOutput, "Workload is stable", "Expected log to contain message about stable workload")
		assert.Contains(t, logOutput, successfullTriggerTargetMsg, "Expected log to contain message about successful reload")
		assert.Contains(t, logOutput, successfullTriggerAllTargetsMsg, "Expected log to contain message about successful reload")
	})

	t.Run("Successful Reload of All Targets", func(t *testing.T) {
		t.Parallel()

		sourceObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-namespace/target-deployment",
				},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-deployment",
				Namespace: "test-namespace",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      3,
				UpdatedReplicas:    3,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(3),
			},
		}

		reconciler := createBaseReconciler(sourceObj, targetObj)

		result, err := reconciler.ReconcileWorkload(context.Background(), sourceObj)

		assert.NoError(t, err, "Expected no error during reconciliation")
		expectedResult := ctrl.Result{RequeueAfter: defaultRequeuAfter}
		assert.Equal(t, expectedResult, result, "Expected successful result with default requeue duration")
	})

	t.Run("Partial successfully reloaded targets", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-namespace/another-deployment, test-namespace/notfound-deployment",
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:  4,
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		targetObj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: "test-namespace",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		notfoundDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "notfound-deployment",
				Namespace: "test-namespace",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:  4,
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
			},
		}

		// Create fake client with objects
		baseFakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj, targetObj, notfoundDeployment).Build()
		fakeClient := &testutils.MockClientWithError{
			Client:        baseFakeClient,
			PatchErrorFor: testutils.NamedError{Name: "notfound-deployment", Namespace: "test-namespace"},
		}

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer))

		reconciler := createBaseReconciler()
		reconciler.Logger = &logger
		reconciler.KubeClient = fakeClient

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.NoError(t, err)
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, workloadStableMsg, "Expected log to contain message about stable workload")
		assert.Contains(t, logOutput, successfullTriggerTargetMsg, "Expected log to contain message about successful reload")
		assert.Contains(t, logOutput, failedTriggerTargetMsg, "Expected log to contain failure message for notfound-deployment")
	})

	t.Run("Error patching workload (LastObservedRestart)", func(t *testing.T) {
		t.Parallel()

		restartedAt := "2024-04-03T12:00:00Z"

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "notfound-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-namespace/another-deployment",
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas:  4,
				ReadyReplicas:      4,
				UpdatedReplicas:    4,
				ObservedGeneration: 5,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: testutils.Int32Ptr(4),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.RestartedAtKey: restartedAt,
						},
					},
				},
			},
		}

		// Create fake client with objects
		baseFakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()
		fakeClient := &testutils.MockClientWithError{
			Client:        baseFakeClient,
			PatchErrorFor: testutils.NamedError{Name: "notfound-deployment", Namespace: "test-namespace"},
		}

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer))

		reconciler := createBaseReconciler()
		reconciler.Logger = &logger
		reconciler.KubeClient = fakeClient

		result, err := reconciler.ReconcileWorkload(context.Background(), obj)
		assert.Error(t, err)
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")
		assert.EqualError(t, err, fmt.Sprintf("failed get last restartedAt annotation: failed to patch restart annotation: failed to patch annotation \"cascader.tkb.ch/last-observed-restart\"=%q: simulated patch error", restartedAt))
	})
}

func TestIsNewRestart(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("Restart Detected and Patched", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		restartedAt := "2024-04-03T12:00:00Z"

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
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

		reconciler := createBaseReconciler(dep)

		workload, err := workloads.NewWorkload(dep)
		assert.NoError(t, err)

		changed, seenAt, err := reconciler.isNewRestart(ctx, workload)
		assert.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, restartedAt, seenAt)

		// Confirm annotation was set
		var updated appsv1.Deployment
		err = reconciler.KubeClient.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err)
		observed := updated.Spec.Template.Annotations[reconciler.LastObservedRestartKey]
		assert.NotEmpty(t, observed)
	})

	t.Run("Restart Already Observed", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		timestamp := "2024-04-03T12:00:00Z"

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							utils.RestartedAtKey:                    timestamp,
							"cascader.tkb.ch/last-observed-restart": timestamp,
						},
					},
				},
			},
		}

		reconciler := createBaseReconciler(dep)

		workload, err := workloads.NewWorkload(dep)
		assert.NoError(t, err)

		changed, seenAt, err := reconciler.isNewRestart(ctx, workload)
		assert.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, timestamp, seenAt)
	})

	t.Run("Patch Error Occurs", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		restartedAt := "2024-04-03T12:00:00Z"

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-deployment",
				Namespace: "default",
			},
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

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()
		mockClient := &testutils.MockClientWithError{
			Client:        fakeClient,
			PatchErrorFor: testutils.NamedError{Name: "failing-deployment", Namespace: "default"},
		}

		reconciler := createBaseReconciler(dep)
		reconciler.KubeClient = mockClient

		workload, err := workloads.NewWorkload(dep)
		assert.NoError(t, err)

		changed, seenAt, err := reconciler.isNewRestart(ctx, workload)
		assert.Error(t, err)
		assert.False(t, changed)
		assert.Empty(t, seenAt)
	})
}

func TestExtractTargets(t *testing.T) {
	t.Parallel()

	t.Run("No Annotations", func(t *testing.T) {
		t.Parallel()

		reconciler := createBaseReconciler()

		obj := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}

		targets, err := reconciler.extractTargets(context.Background(), obj)
		assert.Empty(t, targets)
		assert.NoError(t, err)
	})

	t.Run("Valid Annotations", func(t *testing.T) {
		t.Parallel()

		preloadedObject := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-target",
				},
			},
		}
		preloadedTarget := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-target",
				Namespace: "default",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
		}

		reconciler := createBaseReconciler(preloadedObject, preloadedTarget)

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-target,",
				},
			},
		}

		targets, err := reconciler.extractTargets(context.Background(), obj)
		assert.Error(t, err)
		assert.EqualError(t, err, "targets cannot be empty")
		assert.Empty(t, targets)
	})

	t.Run("Empty Annotations", func(t *testing.T) {
		t.Parallel()

		reconciler := createBaseReconciler()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "",
				},
			},
		}

		targets, err := reconciler.extractTargets(context.Background(), obj)

		assert.Error(t, err)
		assert.EqualError(t, err, "targets cannot be empty")
		assert.Empty(t, targets)
	})

	t.Run("No wanted annotations", func(t *testing.T) {
		t.Parallel()

		reconciler := createBaseReconciler()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					"cascder.tkb.ch": "other-annotation",
				},
			},
		}

		targets, err := reconciler.extractTargets(context.Background(), obj)
		assert.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("Invalid annotation reference", func(t *testing.T) {
		t.Parallel()

		reconciler := createBaseReconciler()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "invalid//annotation",
				},
			},
		}

		targets, err := reconciler.extractTargets(context.Background(), obj)
		assert.Error(t, err)
		assert.EqualError(t, err, "cannot create target for workload: invalid reference: invalid format: invalid//annotation")
		assert.Empty(t, targets)
	})
}

func TestTriggerReloads(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("All Reloads Succeed", func(t *testing.T) {
		t.Parallel()

		sts1 := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "statefulset-1",
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
		workload, _ := workloads.NewWorkload(sts1)

		sts2 := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "statefulset-2",
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

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sts1, sts2).Build()

		target1 := targets.NewStatefulSet("default", "statefulset-1", fakeClient)
		target2 := targets.NewStatefulSet("default", "statefulset-2", fakeClient)

		reconciler := createBaseReconciler(sts1, sts2)

		successes, failures := reconciler.triggerReloads(context.Background(), workload, []targets.Target{target1, target2})

		assert.Equal(t, 2, successes, "All reloads should succeed")
		assert.Equal(t, 0, failures, "No failures should occur")
	})

	t.Run("Some Reloads Fail", func(t *testing.T) {
		t.Parallel()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid-statefulset",
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
		workload, _ := workloads.NewWorkload(sts)

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sts).Build()

		validTarget := targets.NewStatefulSet("default", "valid-statefulset", fakeClient)
		invalidTarget := targets.NewStatefulSet("default", "nonexistent-statefulset", fakeClient)

		reconciler := createBaseReconciler(sts)

		successes, failures := reconciler.triggerReloads(context.Background(), workload, []targets.Target{validTarget, invalidTarget})

		assert.Equal(t, 1, successes, "One reload should succeed")
		assert.Equal(t, 1, failures, "One reload should fail")
	})

	t.Run("All Reloads Fail", func(t *testing.T) {
		t.Parallel()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid-statefulset",
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

		workload, _ := workloads.NewWorkload(sts)

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		target1 := targets.NewStatefulSet("default", "nonexistent-1", fakeClient)
		target2 := targets.NewStatefulSet("default", "nonexistent-2", fakeClient)

		reconciler := createBaseReconciler(sts)

		successes, failures := reconciler.triggerReloads(context.Background(), workload, []targets.Target{target1, target2})

		assert.Equal(t, 0, successes, "No reloads should succeed")
		assert.Equal(t, 2, failures, "All reloads should fail")
	})
}

func TestGetRequeueDuration(t *testing.T) {
	t.Parallel()

	reconciler := createBaseReconciler()

	t.Run("No annotations present", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
		}

		duration, err := reconciler.getRequeueDuration(obj)
		assert.NoError(t, err)
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 2 seconds when annotations are missing")
	})

	t.Run("Annotation not found", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"other.annotation": "10s",
				},
			},
		}

		duration, err := reconciler.getRequeueDuration(obj)
		assert.NoError(t, err)
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 2 seconds when annotation is not found")
	})

	t.Run("Empty annotation value", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/requeueAfter": "",
				},
			},
		}

		duration, err := reconciler.getRequeueDuration(obj)
		assert.NoError(t, err)
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 2 seconds when annotation value is empty")
	})

	t.Run("Invalid annotation value", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/requeueAfter": "invalid-duration",
				},
			},
		}

		duration, err := reconciler.getRequeueDuration(obj)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid annotation: time: invalid duration \"invalid-duration\"")
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 2 seconds on parsing error")
	})

	t.Run("Valid annotation value", func(t *testing.T) {
		t.Parallel()

		obj := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/requeueAfter": "10s",
				},
			},
		}

		duration, err := reconciler.getRequeueDuration(obj)
		assert.NoError(t, err)
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 10s")
	})
}
