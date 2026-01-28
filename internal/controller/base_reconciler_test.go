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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thurgauerkb/cascader/internal/flag"
	"github.com/thurgauerkb/cascader/internal/kinds"
	internalmetrics "github.com/thurgauerkb/cascader/internal/metrics"
	"github.com/thurgauerkb/cascader/internal/targets"
	"github.com/thurgauerkb/cascader/internal/workloads"
	"github.com/thurgauerkb/cascader/test/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	successfullTriggerAllTargetsMsg string        = "Finished handling targets"
	failedTriggerTargetMsg          string        = "Some targets failed to reload"
	restartDetectedMsg              string        = "Restart detected, handling targets"
)

// Helper function to create a fake BaseReconciler
func createBaseReconciler(objects ...client.Object) *BaseReconciler {
	fakeClient := fake.NewClientBuilder().WithObjects(objects...).Build()
	promReg := prometheus.NewRegistry()
	metricsReg := internalmetrics.NewRegistry(promReg)
	return &BaseReconciler{
		Logger:                        &logr.Logger{},
		KubeClient:                    fakeClient,
		Recorder:                      record.NewFakeRecorder(10),
		Metrics:                       metricsReg,
		LastObservedRestartAnnotation: "cascader.tkb.ch/last-observed-restart",
		RequeueAfterAnnotation:        "cascader.tkb.ch/requeueAfter",
		RequeueAfterDefault:           defaultRequeuAfter,
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

	now := time.Now().Format(time.RFC3339)

	t.Run("Empty Target annotation", func(t *testing.T) {
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

		// Capture logs into a string buffer
		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer)) // Set up a zap logger that writes to logBuffer

		reconciler := createBaseReconciler(dep1, targetObj)
		reconciler.Logger = &logger

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: dep1})
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "No targets found; skipping reload.")
	})

	t.Run("Invalid Target annotation", func(t *testing.T) {
		dep1 := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "invalid/target/annotation",
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: dep1})
		require.Error(t, err)
		assert.EqualError(t, err, "failed to create targets: cannot create target for workload: invalid reference: invalid format: invalid/target/annotation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: dep1})
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
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

		var logBuffer bytes.Buffer
		logger := zap.New(zap.WriteTo(&logBuffer)) // Set up a zap logger that writes to logBuffer

		reconciler := createBaseReconciler(obj)
		reconciler.Logger = &logger

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		expectedLog := "direct cycle detected: adding dependency from Deployment/test-namespace/test-deployment creates a direct cycle: Deployment/test-namespace/test-deployment"
		assert.Contains(t, logOutput, expectedLog, "Expected log to contain message about cycle")
	})

	t.Run("Successful Reconciliation - Workload is stable", func(t *testing.T) {
		t.Parallel()

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
							flag.LastObservedRestartAnnotation: now,
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
		assert.NoError(t, err, "Expected no error on successful reconciliation")
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, restartDetectedMsg, "Expected log to contain message about restart detected")
		assert.Contains(t, logOutput, "Workload is stable", "Expected log to contain message about stable workload")
		assert.Contains(t, logOutput, "Dependent targets extracted")
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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: sourceObj})

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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
		assert.NoError(t, err)
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, workloadStableMsg, "Expected log to contain message about stable workload")
		assert.Contains(t, logOutput, successfullTriggerTargetMsg, "Expected log to contain message about successful reload")
		assert.Contains(t, logOutput, failedTriggerTargetMsg, "Expected log to contain failure message for notfound-deployment")
	})

	t.Run("Error patching workload (Transitioning)", func(t *testing.T) {
		t.Parallel()

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

		result, err := reconciler.ReconcileWorkload(t.Context(), &workloads.DeploymentWorkload{Deployment: obj})
		require.Error(t, err)
		expectedResult := ctrl.Result{}
		assert.Equal(t, expectedResult, result, "Expected successful result")
		assert.ErrorContains(t, err, "failed to patch restart annotation: failed to patch annotation \"cascader.tkb.ch/last-observed-restart\"")
		assert.ErrorContains(t, err, "simulated patch error")
	})
}

func TestSetLastObservedRestartAnnotation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	now := time.Now().Format(time.RFC3339)

	t.Run("Successful patch", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					flag.LastObservedRestartAnnotation: now,
				},
			},
		}

		reconciler := createBaseReconciler(dep)

		err := reconciler.setLastObservedRestartAnnotation(ctx, &workloads.DeploymentWorkload{Deployment: dep}, "true")
		assert.NoError(t, err)

		// Confirm annotation was set
		var updated appsv1.Deployment
		err = reconciler.KubeClient.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err)
		observed := updated.Annotations[reconciler.LastObservedRestartAnnotation]
		assert.NotEmpty(t, observed)
	})
}

func TestClearLastObservedRestartAnnotation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	now := time.Now().Format(time.RFC3339)

	t.Run("Successful patch", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Annotations: map[string]string{
					flag.LastObservedRestartAnnotation: now,
				},
			},
		}

		reconciler := createBaseReconciler(dep)

		err := reconciler.clearLastObservedRestartAnnotation(ctx, &workloads.DeploymentWorkload{Deployment: dep})
		assert.NoError(t, err)

		// Confirm annotation was set
		var updated appsv1.Deployment
		err = reconciler.KubeClient.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err)
		observed := updated.Annotations[reconciler.LastObservedRestartAnnotation]
		assert.Empty(t, observed)
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

		targets, err := reconciler.extractTargets(t.Context(), obj)
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

		targets, err := reconciler.extractTargets(t.Context(), obj)
		assert.NoError(t, err)
		assert.Len(t, targets, 1, "Expected no targets to be extracted")
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

		targets, err := reconciler.extractTargets(t.Context(), obj)

		assert.NoError(t, err)
		assert.Len(t, targets, 0)
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

		targets, err := reconciler.extractTargets(t.Context(), obj)
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

		targets, err := reconciler.extractTargets(t.Context(), obj)
		require.Error(t, err)
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

		successes, failures := reconciler.triggerReloads(
			t.Context(),
			&workloads.StatefulSetWorkload{StatefulSet: sts1},
			[]targets.Target{target1, target2},
		)

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

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sts).Build()

		validTarget := targets.NewStatefulSet("default", "valid-statefulset", fakeClient)
		invalidTarget := targets.NewStatefulSet("default", "nonexistent-statefulset", fakeClient)

		reconciler := createBaseReconciler(sts)

		successes, failures := reconciler.triggerReloads(
			t.Context(),
			&workloads.StatefulSetWorkload{StatefulSet: sts},
			[]targets.Target{validTarget, invalidTarget},
		)

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

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		target1 := targets.NewStatefulSet("default", "nonexistent-1", fakeClient)
		target2 := targets.NewStatefulSet("default", "nonexistent-2", fakeClient)

		reconciler := createBaseReconciler(sts)

		successes, failures := reconciler.triggerReloads(
			t.Context(),
			&workloads.StatefulSetWorkload{StatefulSet: sts},
			[]targets.Target{target1, target2},
		)

		assert.Equal(t, 0, successes, "No reloads should succeed")
		assert.Equal(t, 2, failures, "All reloads should fail")
	})
}

func TestRequeueDurationFor(t *testing.T) {
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

		duration, err := reconciler.requeueDurationFor(obj)
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

		duration, err := reconciler.requeueDurationFor(obj)
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

		duration, err := reconciler.requeueDurationFor(obj)
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

		duration, err := reconciler.requeueDurationFor(obj)
		require.Error(t, err)
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

		duration, err := reconciler.requeueDurationFor(obj)
		assert.NoError(t, err)
		assert.Equal(t, defaultRequeuAfter, duration, "Expected duration to be 10s")
	})
}
