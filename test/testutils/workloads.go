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

package testutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/thurgauerkb/cascader/internal/testutils"
	"github.com/thurgauerkb/cascader/internal/utils"

	. "github.com/onsi/ginkgo/v2" // nolint:staticcheck
	. "github.com/onsi/gomega"    // nolint:staticcheck

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultTestImage     string = "nginx:1.21"
	defaultTestImageName string = "nginx"
)

// K8sClient is the shared Kubernetes client used in e2e tests.
var K8sClient client.Client

// CreateDeployment creates and applies a Deployment in the specified namespace.
func CreateDeployment(ctx context.Context, namespace, name string, opts ...Option) *appsv1.Deployment { // nolint:dupl
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.DeploymentSpec{
		Replicas: testutils.Int32Ptr(1),
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: defaultTestImageName, Image: defaultTestImage}},
			},
		},
	}
	deployment := &appsv1.Deployment{ObjectMeta: meta, Spec: spec}
	applyOptions(deployment, opts...)
	Expect(K8sClient.Create(ctx, deployment)).To(Succeed())

	CheckResourceReadiness(ctx, deployment)

	deployment.TypeMeta = metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}
	return deployment
}

// CreateStatefulSet creates and applies a StatefulSet in the specified namespace.
func CreateStatefulSet(ctx context.Context, namespace, name string, opts ...Option) *appsv1.StatefulSet { // nolint:dupl
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.StatefulSetSpec{
		Replicas: testutils.Int32Ptr(1),
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: defaultTestImageName, Image: defaultTestImage}},
			},
		},
	}
	statefulSet := &appsv1.StatefulSet{ObjectMeta: meta, Spec: spec}
	applyOptions(statefulSet, opts...)
	Expect(K8sClient.Create(ctx, statefulSet)).To(Succeed())

	CheckResourceReadiness(ctx, statefulSet)

	statefulSet.TypeMeta = metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"}
	return statefulSet
}

// CreateDaemonSet creates and applies a DaemonSet in the specified namespace.
func CreateDaemonSet(ctx context.Context, namespace, name string, opts ...Option) *appsv1.DaemonSet { // nolint:dupl
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: defaultTestImageName, Image: defaultTestImage}},
			},
		},
	}
	daemonSet := &appsv1.DaemonSet{ObjectMeta: meta, Spec: spec}
	applyOptions(daemonSet, opts...)
	Expect(K8sClient.Create(ctx, daemonSet)).To(Succeed())

	CheckResourceReadiness(ctx, daemonSet)

	daemonSet.TypeMeta = metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"}
	return daemonSet
}

// GenerateUniqueName returns a unique name based on the given base string and a truncated UUID.
func GenerateUniqueName(base string) string {
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:5])
}

// GenerateID returns a string identifier in Kind/namespace/name format for a Kubernetes object.
func GenerateID(obj client.Object) string {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	return fmt.Sprintf("%s/%s/%s", kind, obj.GetNamespace(), obj.GetName())
}

// DeleteNamespaceIfExists deletes the given namespace if it exists.
func DeleteNamespaceIfExists(namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	err := K8sClient.Delete(context.Background(), ns)
	if err != nil && !kerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred(), "failed to delete namespace %q", namespace)
	}
}

// DeleteRandomPod deletes a random pod associated with the given resource.
func DeleteRandomPod(ctx context.Context, resource client.Object) {
	pods, err := ListPods(ctx, resource)
	Expect(err).To(Succeed())
	Expect(pods.Items).ToNot(BeEmpty(), "no pods found to delete")

	pod := pods.Items[rand.Intn(len(pods.Items))]
	By(fmt.Sprintf("Deleting pod %q", pod.Name))
	Expect(K8sClient.Delete(ctx, &pod)).To(Succeed())
}

// ListPods returns all pods associated with a Deployment, StatefulSet, or DaemonSet.
func ListPods(ctx context.Context, resource client.Object) (*corev1.PodList, error) {
	var selector labels.Selector

	switch res := resource.(type) {
	case *appsv1.Deployment:
		selector = labels.SelectorFromSet(res.Spec.Template.Labels)
	case *appsv1.StatefulSet:
		selector = labels.SelectorFromSet(res.Spec.Template.Labels)
	case *appsv1.DaemonSet:
		selector = labels.SelectorFromSet(res.Spec.Template.Labels)
	default:
		return nil, fmt.Errorf("unsupported resource type: %T", resource)
	}

	pods := &corev1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(resource.GetNamespace()),
		client.MatchingLabelsSelector{Selector: selector},
	}

	if err := K8sClient.List(ctx, pods, opts...); err != nil {
		return nil, fmt.Errorf("failed to list pods for %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}

	return pods, nil
}

// ScaleResource updates the replica count of a Deployment or StatefulSet.
func ScaleResource(ctx context.Context, resource client.Object, replicas int32) {
	patch := client.MergeFrom(resource.DeepCopyObject().(client.Object))

	switch res := resource.(type) {
	case *appsv1.Deployment:
		res.Spec.Replicas = &replicas
	case *appsv1.StatefulSet:
		res.Spec.Replicas = &replicas
	case *appsv1.DaemonSet:
		panic("cannot scale a DaemonSet using replicas")
	default:
		panic(fmt.Sprintf("unsupported resource type: %T", res))
	}

	Expect(K8sClient.Patch(ctx, resource, patch)).To(Succeed())
}

// RestartResource sets the restart annotation on the PodTemplateSpec of a resource.
func RestartResource(ctx context.Context, resource client.Object) {
	By(fmt.Sprintf("Restarting %s %s/%s", resource.GetObjectKind().GroupVersionKind().Kind, resource.GetNamespace(), resource.GetName()))

	patch := client.MergeFrom(resource.DeepCopyObject().(client.Object))
	now := time.Now().Format(time.RFC3339)

	var template *corev1.PodTemplateSpec

	switch res := resource.(type) {
	case *appsv1.Deployment:
		template = &res.Spec.Template
	case *appsv1.StatefulSet:
		template = &res.Spec.Template
	case *appsv1.DaemonSet:
		template = &res.Spec.Template
	default:
		panic(fmt.Sprintf("unsupported resource type: %T", res))
	}

	if template.Annotations == nil {
		template.Annotations = map[string]string{}
	}
	template.Annotations[utils.RestartedAtKey] = now

	Expect(K8sClient.Patch(ctx, resource, patch)).To(Succeed())
}

// CheckResourceReadiness waits until a Deployment, StatefulSet, or DaemonSet is ready.
func CheckResourceReadiness(ctx context.Context, resource client.Object) {
	By(fmt.Sprintf("Checking readiness of %T %s/%s", resource, resource.GetNamespace(), resource.GetName()))

	Eventually(func() bool {
		err := K8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
		Expect(err).NotTo(HaveOccurred())

		switch obj := resource.(type) {
		case *appsv1.Deployment:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.StatefulSet:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.DaemonSet:
			return obj.Status.NumberReady == obj.Status.DesiredNumberScheduled
		default:
			Fail(fmt.Sprintf("unsupported resource type: %T", resource))
			return false
		}
	}, 2*time.Minute, 2*time.Second).Should(BeTrue(), fmt.Sprintf("resource %T %s/%s did not become ready", resource, resource.GetNamespace(), resource.GetName()))
}
