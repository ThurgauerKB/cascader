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
	"fmt"
	"testing"

	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/targets"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckCycle(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("No Cycle", func(t *testing.T) {
		t.Parallel()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backend-service",
				Namespace: "no-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "no-cycle/service-a",
				},
			},
		}
		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service-a",
				Namespace: "no-cycle",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(depA, depB).Build()

		annotationsKindMap := kinds.AnnotationKindMap{
			"cascader.tkb.ch/deployment": kinds.DeploymentKind,
		}

		reconciler := &BaseReconciler{
			KubeClient:        fakeClient,
			AnnotationKindMap: annotationsKindMap,
		}

		srcID := "Deployment/no-cycle/backend-service"
		targetDeps := []targets.Target{
			targets.NewDeployment("no-cycle", "service-a", fakeClient),
		}

		err := reconciler.checkCycle(context.Background(), srcID, targetDeps)
		assert.NoError(t, err)
	})

	t.Run("Direct Cycle", func(t *testing.T) {
		t.Parallel()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backend",
				Namespace: "direct-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "direct-cycle/backend",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(dep).Build()

		reconciler := &BaseReconciler{
			KubeClient: fakeClient,
			AnnotationKindMap: kinds.AnnotationKindMap{
				"cascader.tkb.ch/deployment": kinds.DeploymentKind,
			},
		}

		srcID := "Deployment/direct-cycle/backend"
		targetDeps := []targets.Target{
			targets.NewDeployment("direct-cycle", "backend", fakeClient),
		}

		err := reconciler.checkCycle(context.Background(), srcID, targetDeps)
		assert.Error(t, err)
		assert.EqualError(t, err, "direct cycle detected: adding dependency from Deployment/direct-cycle/backend creates a direct cycle: Deployment/direct-cycle/backend")
	})

	t.Run("Indirect Cycle", func(t *testing.T) {
		t.Parallel()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "indirect-cycle/second",
				},
			},
		}
		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "second",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "indirect-cycle/first",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(depA, depB).Build()

		reconciler := &BaseReconciler{
			KubeClient: fakeClient,
			AnnotationKindMap: kinds.AnnotationKindMap{
				"cascader.tkb.ch/deployment": kinds.DeploymentKind,
			},
		}

		srcID := "Deployment/indirect-cycle/first"
		targetDeps := []targets.Target{
			targets.NewDeployment("indirect-cycle", "second", fakeClient),
		}

		err := reconciler.checkCycle(context.Background(), srcID, targetDeps)
		assert.Error(t, err)
		assert.EqualError(t, err, "indirect cycle detected: adding dependency from Deployment/indirect-cycle/first creates a indirect cycle: Deployment/indirect-cycle/first -> Deployment/indirect-cycle/second -> Deployment/indirect-cycle/first")
	})

	t.Run("Error when fetching resource", func(t *testing.T) {
		t.Parallel()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "second",
				},
			},
		}

		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "second",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "non-existing",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(depA, depB).Build()

		reconciler := &BaseReconciler{
			KubeClient: fakeClient,
			AnnotationKindMap: kinds.AnnotationKindMap{
				"cascader.tkb.ch/deployment": kinds.DeploymentKind,
			},
		}

		srcID := "Deployment/indirect-cycle/first"
		targetDeps := []targets.Target{
			targets.NewDeployment("indirect-cycle", "second", fakeClient),
		}

		err := reconciler.checkCycle(context.Background(), srcID, targetDeps)
		assert.Error(t, err)
		assert.EqualError(t, err, "dependency cycle check failed: failed to fetch resource Deployment/indirect-cycle/non-existing: deployments.apps \"non-existing\" not found")
	})

	t.Run("Error when extracting dependencies", func(t *testing.T) {
		t.Parallel()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "first",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "second",
				},
			},
		}

		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "second",
				Namespace: "indirect-cycle",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "invalid/target/annotation",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(depA, depB).Build()

		reconciler := &BaseReconciler{
			KubeClient: fakeClient,
			AnnotationKindMap: kinds.AnnotationKindMap{
				"cascader.tkb.ch/deployment": kinds.DeploymentKind,
			},
		}

		srcID := "Deployment/indirect-cycle/first"
		targetDeps := []targets.Target{
			targets.NewDeployment("indirect-cycle", "second", fakeClient),
		}

		err := reconciler.checkCycle(context.Background(), srcID, targetDeps)
		assert.Error(t, err)
		assert.EqualError(t, err, "dependency cycle check failed: error extracting dependencies: cannot create target for workload: invalid reference: invalid format: invalid/target/annotation")
	})
}

func TestDetectCycle(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	t.Run("Cycle detected when visiting the same node twice", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		target := targets.NewDeployment("test-namespace", "test-deployment", fakeClient)
		depChain := []string{"root", target.ID()}

		reconciler := &BaseReconciler{}
		hasCycle, updatedChain, err := reconciler.detectCycle(ctx, target, "root", depChain)

		assert.NoError(t, err)
		assert.True(t, hasCycle, "Expected a cycle to be detected")
		assert.Equal(t, []string{"root", "Deployment/test-namespace/test-deployment", "Deployment/test-namespace/test-deployment"}, updatedChain)
	})

	t.Run("Fails to fetch resource", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		target := targets.NewDeployment("test-namespace", "test-deployment", fakeClient)
		depChain := []string{"root"}

		reconciler := &BaseReconciler{
			KubeClient: fakeClient,
		}

		hasCycle, updatedChain, err := reconciler.detectCycle(ctx, target, "root", depChain)

		assert.False(t, hasCycle, "Expected no cycle detected")
		assert.Nil(t, updatedChain, "Expected nil cycle path")
		assert.Error(t, err)
		assert.EqualError(t, err, "failed to fetch resource Deployment/test-namespace/test-deployment: deployments.apps \"test-deployment\" not found")
	})

	t.Run("Indirect cycle detected", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-ns/dep-dep",
				},
			},
		}

		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-ns/test-dep",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(depA, depB).
			Build()

		target := targets.NewDeployment("test-ns", "test-dep", fakeClient)

		annotations := kinds.AnnotationKindMap{
			"cascader.tkb.ch/deployment": kinds.DeploymentKind,
		}

		reconciler := &BaseReconciler{
			KubeClient:        fakeClient,
			AnnotationKindMap: annotations,
		}

		sourceID := fmt.Sprintf("Deployment/%s/%s", depA.GetNamespace(), depA.GetName())
		traversalPath := []string{}
		expectedCycle := []string{
			"Deployment/test-ns/test-dep",
			"Deployment/test-ns/dep-dep",
			"Deployment/test-ns/test-dep",
		}

		hasCycle, cyclePath, err := reconciler.detectCycle(ctx, target, sourceID, traversalPath)

		assert.True(t, hasCycle, "Expected an indirect cycle to be detected")

		assert.Equal(t, expectedCycle, cyclePath, "Expected a valid cycle path")
		assert.NoError(t, err)
	})

	t.Run("Ignore empty target", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-ns/dep-dep",
				},
			},
		}

		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(depA, depB).
			Build()

		target := targets.NewDeployment("test-ns", "test-dep", fakeClient)

		annotationKindMap := kinds.AnnotationKindMap{
			"cascader.tkb.ch/deployment": kinds.DeploymentKind,
		}

		reconciler := &BaseReconciler{
			KubeClient:        fakeClient,
			AnnotationKindMap: annotationKindMap,
		}

		sourceID := fmt.Sprintf("Deployment/%s/%s", depA.GetNamespace(), depA.GetName())
		traversalPath := []string{}

		hasCycle, cyclePath, err := reconciler.detectCycle(ctx, target, sourceID, traversalPath)

		assert.False(t, hasCycle, "Expected no cycle detected")
		assert.Nil(t, cyclePath, "Expected nil cycle path")
		assert.NoError(t, err)
	})

	t.Run("Error during traversal - target not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		depA := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "test-ns/dep-dep",
				},
			},
		}

		depB := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dep-dep",
				Namespace: "test-ns",
				Annotations: map[string]string{
					"cascader.tkb.ch/deployment": "not-existing",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(depA, depB).
			Build()

		target := targets.NewDeployment("test-ns", "test-dep", fakeClient)

		annotationKindMap := kinds.AnnotationKindMap{
			"cascader.tkb.ch/deployment": kinds.DeploymentKind,
		}

		reconciler := &BaseReconciler{
			KubeClient:        fakeClient,
			AnnotationKindMap: annotationKindMap,
		}

		sourceID := fmt.Sprintf("Deployment/%s/%s", depA.GetNamespace(), depA.GetName())
		traversalPath := []string{}

		hasCycle, cyclePath, err := reconciler.detectCycle(ctx, target, sourceID, traversalPath)

		assert.False(t, hasCycle, "Expected no cycle detected")
		assert.Nil(t, cyclePath, "Expected nil cycle path")
		assert.Error(t, err)
		assert.EqualError(t, err, "failed to fetch resource Deployment/test-ns/not-existing: deployments.apps \"not-existing\" not found")
	})
}
