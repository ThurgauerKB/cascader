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
	"time"

	"github.com/thurgauerkb/cascader/internal/testutils"
	"github.com/thurgauerkb/cascader/internal/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sClient is the global Kubernetes client for tests.
var K8sClient client.Client

// CreateDeployment creates a Deployment resource in the specified namespace with the given name.
func CreateDeployment(ctx context.Context, namespace, name string, opts ...Option) *appsv1.Deployment {
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.DeploymentSpec{
		Replicas: testutils.Int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:1.21",
					},
				},
			},
		},
	}

	deployment := &appsv1.Deployment{ObjectMeta: meta, Spec: spec}
	applyOptions(deployment, opts...)
	Expect(K8sClient.Create(ctx, deployment)).To(Succeed())

	waitForResourceReady(ctx, deployment)

	deployment.TypeMeta = metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}
	return deployment
}

// CreateStatefulSet creates a StatefulSet resource in the specified namespace with the given name.
func CreateStatefulSet(ctx context.Context, namespace, name string, opts ...Option) *appsv1.StatefulSet {
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.StatefulSetSpec{
		Replicas: testutils.Int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:1.21",
					},
				},
			},
		},
	}

	statefulSet := &appsv1.StatefulSet{ObjectMeta: meta, Spec: spec}
	applyOptions(statefulSet, opts...)
	Expect(K8sClient.Create(ctx, statefulSet)).To(Succeed())

	waitForResourceReady(ctx, statefulSet)

	statefulSet.TypeMeta = metav1.TypeMeta{
		Kind:       "StatefulSet",
		APIVersion: "apps/v1",
	}
	return statefulSet
}

// CreateDaemonSet creates a DaemonSet resource in the specified namespace with the given name.
func CreateDaemonSet(ctx context.Context, namespace, name string, opts ...Option) *appsv1.DaemonSet {
	meta := metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: map[string]string{},
		Labels:      map[string]string{"app": name},
	}
	spec := appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": name},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:1.21",
					},
				},
			},
		},
	}

	daemonSet := &appsv1.DaemonSet{ObjectMeta: meta, Spec: spec}
	applyOptions(daemonSet, opts...)
	Expect(K8sClient.Create(ctx, daemonSet)).To(Succeed())

	waitForResourceReady(ctx, daemonSet)

	daemonSet.TypeMeta = metav1.TypeMeta{
		Kind:       "DaemonSet",
		APIVersion: "apps/v1",
	}
	return daemonSet
}

// waitForResourceReady waits until the specified resource is ready.
func waitForResourceReady(ctx context.Context, resource client.Object) {
	By(fmt.Sprintf("waiting for %T %s/%s to be ready", resource, resource.GetNamespace(), resource.GetName()))
	Eventually(func() bool {
		// Fetch the resource from Kubernetes
		err := K8sClient.Get(ctx, client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.GetName()}, resource)
		if err != nil {
			return false
		}

		// Check readiness based on the resource type
		switch obj := resource.(type) {
		case *appsv1.Deployment:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.StatefulSet:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.DaemonSet:
			return obj.Status.NumberReady == obj.Status.DesiredNumberScheduled
		default:
			return false // Unsupported resource type
		}
	}, 2*time.Minute, 2*time.Second).Should(BeTrue(), fmt.Sprintf("%T %s/%s is not ready", resource, resource.GetNamespace(), resource.GetName()))
}

// GenerateUniqueName generates a unique name using a base string and a truncated UUID.
func GenerateUniqueName(base string) string {
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:5])
}

// GenerateID generates an ID from Kind/namespace/name
func GenerateID(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind().Kind
	return fmt.Sprintf("%s/%s/%s", gvk, obj.GetNamespace(), obj.GetName())
}

// DeleteNamespaceIfExists deletes a namespace if it exists.
func DeleteNamespaceIfExists(namespace string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	err := K8sClient.Delete(context.Background(), ns)
	if err != nil && !kerrors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred(), "Failed to delete namespace")
	}
}

// DeleteResourcePods deletes all pods associated with a specific Kubernetes resource.
func DeleteResourcePods(ctx context.Context, resource client.Object) error {
	podList, err := ListPods(ctx, resource)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		if err := K8sClient.Delete(ctx, &pod); err != nil {
			return fmt.Errorf("failed to delete pod %s/%s: %w", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}

// ListPods lists all pods associated with a specific Kubernetes resource.
func ListPods(ctx context.Context, resource client.Object) (podList *corev1.PodList, err error) {
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

	podList = &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(resource.GetNamespace()),
		client.MatchingLabelsSelector{Selector: selector},
	}

	if err := K8sClient.List(ctx, podList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list pods for resource %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}
	return podList, err
}

// ScaleResource scales a given Kubernetes resource by updating its replica count.
func ScaleResource(ctx context.Context, resource client.Object, replicas int32) error {
	patch := client.MergeFrom(resource.DeepCopyObject().(client.Object))

	switch r := resource.(type) {
	case *appsv1.Deployment:
		r.Spec.Replicas = &replicas
	case *appsv1.StatefulSet:
		r.Spec.Replicas = &replicas
	case *appsv1.DaemonSet:
		return fmt.Errorf("DaemonSet does not support scaling via replicas")
	default:
		return fmt.Errorf("unsupported resource type: %T", resource)
	}

	return K8sClient.Patch(ctx, resource, patch)
}

// RestartResource restarts a given Kubernetes resource by updating its template annotations.
func RestartResource(ctx context.Context, resource client.Object) {
	By(fmt.Sprintf("restarting %s", resource.GetName()))

	patch := client.MergeFrom(resource.DeepCopyObject().(client.Object))
	now := time.Now().Format(time.RFC3339)

	switch res := resource.(type) {
	case *appsv1.Deployment:
		if res.Spec.Template.Annotations == nil {
			res.Spec.Template.Annotations = map[string]string{}
		}
		res.Spec.Template.Annotations[utils.RestartedAtKey] = now
	case *appsv1.StatefulSet:
		if res.Spec.Template.Annotations == nil {
			res.Spec.Template.Annotations = map[string]string{}
		}
		res.Spec.Template.Annotations[utils.RestartedAtKey] = now
	case *appsv1.DaemonSet:
		if res.Spec.Template.Annotations == nil {
			res.Spec.Template.Annotations = map[string]string{}
		}
		res.Spec.Template.Annotations[utils.RestartedAtKey] = now
	}

	err := K8sClient.Patch(ctx, resource, patch)
	Expect(err).To(Succeed())
}

// CheckResourceReadiness validates that a Kubernetes resource is ready.
func CheckResourceReadiness(ctx context.Context, resource client.Object) {
	By(fmt.Sprintf("validating readiness of %T %s/%s", resource, resource.GetNamespace(), resource.GetName()))
	Eventually(func() bool {
		err := K8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
		Expect(err).NotTo(HaveOccurred())

		// Check readiness based on resource type
		switch obj := resource.(type) {
		case *appsv1.Deployment:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.StatefulSet:
			return obj.Status.ReadyReplicas == *obj.Spec.Replicas
		case *appsv1.DaemonSet:
			return obj.Status.NumberReady == obj.Status.DesiredNumberScheduled
		default:
			Fail(fmt.Sprintf("Unsupported resource type: %T", resource))
			return false
		}
	}, 2*time.Minute, 2*time.Second).Should(BeTrue(), fmt.Sprintf("%T readiness check failed for %s/%s", resource, resource.GetNamespace(), resource.GetName()))
}
