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

package utils

import (
	"context"
	"testing"
	"time"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/test/testutils"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const lastObservedRestartKey string = "cascader.tkb.ch/last-observed-restart"

func TestUniqueAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("All unique annotations", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{
			"Deployment":   "cascader.tkb.ch/deployment",
			"Statefulset":  "cascader.tkb.ch/statefulset",
			"Daemonset":    "cascader.tkb.ch/daemonset",
			"RequeueAfter": "cascader.tkb.ch/requeue-after",
		}
		err := UniqueAnnotations(annotations)
		assert.NoError(t, err)
	})

	t.Run("Duplicate values", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{
			"Deployment":   "cascader.tkb.ch/deployment",
			"StatefulSet":  "cascader.tkb.ch/deployment",
			"RequeueAfter": "cascader.tkb.ch/requeue-after",
		}
		err := UniqueAnnotations(annotations)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "duplicate annotation 'cascader.tkb.ch/deployment'")
		assert.ErrorContains(t, err, "'Deployment'")
		assert.ErrorContains(t, err, "'StatefulSet'")
	})

	t.Run("Empty map", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{}
		err := UniqueAnnotations(annotations)
		assert.Error(t, err)
		assert.EqualError(t, err, "no annotations provided")
	})

	t.Run("Single annotation", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{
			"Deployment": "cascader.tkb.ch/deployment",
		}
		err := UniqueAnnotations(annotations)
		assert.NoError(t, err)
	})
}

func TestFormatAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("Non-empty annotations map", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{
			"Deployment":   "cascader.tkb.ch/deployment",
			"StatefulSet":  "cascader.tkb.ch/statefulset",
			"DaemonSet":    "cascader.tkb.ch/daemonset",
			"RequeueAfter": "cascader.tkb.ch/requeue-after",
		}
		expected := "DaemonSet=cascader.tkb.ch/daemonset, Deployment=cascader.tkb.ch/deployment, RequeueAfter=cascader.tkb.ch/requeue-after, StatefulSet=cascader.tkb.ch/statefulset"
		result := FormatAnnotations(annotations)

		assert.Equal(t, expected, result)
	})

	t.Run("Empty annotations map", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{}
		expected := ""
		result := FormatAnnotations(annotations)

		assert.Equal(t, expected, result)
	})

	t.Run("Single annotation", func(t *testing.T) {
		t.Parallel()

		annotations := map[string]string{
			"Deployment": "cascader.tkb.ch/deployment",
		}
		expected := "Deployment=cascader.tkb.ch/deployment"
		result := FormatAnnotations(annotations)

		assert.Equal(t, expected, result)
	})
}

func TestToCacheOptions(t *testing.T) {
	t.Parallel()

	t.Run("Empty namespace list returns empty cache options", func(t *testing.T) {
		t.Parallel()

		opts := ToCacheOptions(nil)
		assert.Nil(t, opts.DefaultNamespaces)
	})

	t.Run("Single namespace", func(t *testing.T) {
		t.Parallel()

		opts := ToCacheOptions([]string{"ns1"})
		assert.Len(t, opts.DefaultNamespaces, 1)
		_, exists := opts.DefaultNamespaces["ns1"]
		assert.True(t, exists)
	})

	t.Run("Multiple namespaces", func(t *testing.T) {
		t.Parallel()

		namespaces := []string{"ns1", "ns2", "ns3"}
		opts := ToCacheOptions(namespaces)
		assert.Len(t, opts.DefaultNamespaces, 3)
		for _, ns := range namespaces {
			_, exists := opts.DefaultNamespaces[ns]
			assert.True(t, exists, "expected namespace %q to exist in DefaultNamespaces", ns)
		}
	})
}

func TestParseTargetRef(t *testing.T) {
	t.Parallel()

	t.Run("Only name provided", func(t *testing.T) {
		t.Parallel()

		target := "only-name-target"
		defaultNamespace := "only-name-ns"
		expectedNS := "only-name-ns"
		expectedName := "only-name-target"

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.NoError(t, err)
		assert.Equal(t, expectedNS, ns)
		assert.Equal(t, expectedName, name)
	})

	t.Run("Namespace and name provided", func(t *testing.T) {
		t.Parallel()

		target := "production/ns-name"
		defaultNamespace := "ns-name"
		expectedNS := "production"
		expectedName := "ns-name"

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.NoError(t, err)
		assert.Equal(t, expectedNS, ns)
		assert.Equal(t, expectedName, name)
	})

	t.Run("Invalid target too many slashes", func(t *testing.T) {
		t.Parallel()

		target := "prod/us-west/my-deployment"
		defaultNamespace := "to-many"

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.Error(t, err)
		assert.Empty(t, ns)
		assert.Empty(t, name)
	})

	t.Run("Empty target", func(t *testing.T) {
		t.Parallel()

		target := ""
		defaultNamespace := "empty-target"
		expectedNS := "empty-target"
		expectedName := ""

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.NoError(t, err)
		assert.Equal(t, expectedNS, ns)
		assert.Equal(t, expectedName, name)
	})

	t.Run("Trailing slash", func(t *testing.T) {
		t.Parallel()

		target := "production/"
		defaultNamespace := "default"
		expectedNS := "production"
		expectedName := ""

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.NoError(t, err)
		assert.Equal(t, expectedNS, ns)
		assert.Equal(t, expectedName, name)
	})

	t.Run("Only name provided", func(t *testing.T) {
		t.Parallel()

		target := "/my-deployment"
		defaultNamespace := "default"
		expectedNS := ""
		expectedName := "my-deployment"

		ns, name, err := ParseTargetRef(target, defaultNamespace)

		assert.NoError(t, err)
		assert.Equal(t, expectedNS, ns)
		assert.Equal(t, expectedName, name)
	})
}

func TestGenerateID(t *testing.T) {
	t.Parallel()

	t.Run("should create a unique ID for a resource", func(t *testing.T) {
		t.Parallel()

		expectedID := "Deployment/my-namespace/my-deployment"
		id := GenerateID(kinds.DeploymentKind, "my-namespace", "my-deployment")
		assert.Equal(t, expectedID, id)
	})
}

func TestPatchPodTemplateAnnotation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	now := time.Now().Format(time.RFC3339)

	t.Run("Successful Patch", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

		err := PatchPodTemplateAnnotation(ctx, cl, dep, &dep.Spec.Template, lastObservedRestartKey, now)
		assert.NoError(t, err, "expected no error when patching")

		assert.Equal(t, now, dep.Spec.Template.Annotations[lastObservedRestartKey])

		var patched appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKeyFromObject(dep), &patched)
		assert.NoError(t, err)
		assert.Equal(t, now, patched.Spec.Template.Annotations[lastObservedRestartKey])
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		invalid := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := PatchPodTemplateAnnotation(ctx, cl, invalid, &corev1.PodTemplateSpec{}, lastObservedRestartKey, now)

		assert.Error(t, err, "expected error when patching unsupported object")
	})
}

func TestPatchWorkloadAnnotation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	now := time.Now().Format(time.RFC3339)

	t.Run("Successful Patch", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-deploy",
				Namespace:   "default",
				Annotations: map[string]string{},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

		err := PatchWorkloadAnnotation(ctx, cl, dep, lastObservedRestartKey, now)
		assert.NoError(t, err, "expected no error when patching")

		assert.Equal(t, now, dep.Annotations[lastObservedRestartKey])

		var patched appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKeyFromObject(dep), &patched)
		assert.NoError(t, err)
		assert.Equal(t, now, patched.Annotations[lastObservedRestartKey])
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		invalid := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := PatchWorkloadAnnotation(ctx, cl, invalid, lastObservedRestartKey, now)

		assert.Error(t, err, "expected error when patching unsupported object")
	})
}

func TestDeleteWorkloadAnnotation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	now := time.Now().Format(time.RFC3339)

	t.Run("Successful Delete", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: "default",
				Annotations: map[string]string{
					lastObservedRestartKey: now,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

		err := DeleteWorkloadAnnotation(ctx, cl, dep, lastObservedRestartKey)
		assert.NoError(t, err, "expected no error when deleting annotation")

		var updated appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err, "failed to get updated deployment")

		result, exists := updated.Annotations[lastObservedRestartKey]
		assert.False(t, exists, "annotation should be deleted")
		assert.Equal(t, "", result, "annotation value should be the same as the original")
	})

	t.Run("Successful Delete (no annotation)", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-deploy",
				Namespace:   "default",
				Annotations: map[string]string{},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

		err := DeleteWorkloadAnnotation(ctx, cl, dep, lastObservedRestartKey)
		assert.NoError(t, err, "expected no error when deleting annotation")

		var updated appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err, "failed to get updated deployment")

		result, exists := updated.Annotations[lastObservedRestartKey]
		assert.False(t, exists, "annotation should be deleted")
		assert.Equal(t, "", result, "annotation value should be the same as the original")
	})

	t.Run("Keeps unrelated annotations", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: "default",
				Annotations: map[string]string{
					lastObservedRestartKey: now,
					"other-restartedAtKey": "keep-me",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

		err := DeleteWorkloadAnnotation(ctx, cl, dep, lastObservedRestartKey)
		assert.NoError(t, err, "expected no error when deleting annotation")

		var updated appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKeyFromObject(dep), &updated)
		assert.NoError(t, err, "failed to get updated deployment")

		_, deleted := updated.Annotations[lastObservedRestartKey]
		assert.False(t, deleted, "deleted restartedAtKey should not exist")

		val, kept := updated.Annotations["other-restartedAtKey"]
		assert.True(t, kept, "unrelated annotation should remain")
		assert.Equal(t, "keep-me", val, "unrelated annotation value should be intact")
	})

	t.Run("Patch failure", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deploy",
				Namespace: "default",
				Annotations: map[string]string{
					lastObservedRestartKey: now,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: testutils.DefaultTestImage},
						},
					},
				},
			},
		}

		baseClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()
		mockClient := &testutils.MockClientWithError{
			Client: baseClient,
			PatchErrorFor: testutils.NamedError{
				Name:      "test-deploy",
				Namespace: "default",
			},
		}

		err := DeleteWorkloadAnnotation(ctx, mockClient, dep, lastObservedRestartKey)
		assert.Error(t, err, "expected error when patch fails")
		assert.Contains(t, err.Error(), "failed to delete annotation", "error should include context")
		assert.Contains(t, err.Error(), lastObservedRestartKey, "error should mention the last-observed-restart")
	})
}
