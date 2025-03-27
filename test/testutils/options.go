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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Option defines a functional option for customizing resources.
type Option func(resource client.Object)

// applyOptions applies functional options to a Kubernetes resource.
func applyOptions(resource client.Object, opts ...Option) {
	for _, opt := range opts {
		opt(resource)
	}
}

// WithAnnotation sets a annotation for a resource.
func WithAnnotation(key, value string) Option {
	return func(resource client.Object) {
		annotations := resource.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		resource.SetAnnotations(annotations)
	}
}

// WithReplicas sets the replicas for a resource.
func WithReplicas(replicas int32) Option {
	return func(resource client.Object) {
		switch obj := resource.(type) {
		case *appsv1.Deployment:
			obj.Spec.Replicas = &replicas
		case *appsv1.StatefulSet:
			obj.Spec.Replicas = &replicas
		}
	}
}

// WithStrategy sets the update strategy for a resource.
func WithStrategy(strategy any) Option {
	return func(resource client.Object) {
		switch obj := resource.(type) {
		case *appsv1.Deployment:
			if s, ok := strategy.(appsv1.DeploymentStrategyType); ok {
				obj.Spec.Strategy = appsv1.DeploymentStrategy{
					Type: s,
				}
			}
		case *appsv1.StatefulSet:
			if s, ok := strategy.(appsv1.StatefulSetUpdateStrategyType); ok {
				obj.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
					Type: s,
				}
			}
		case *appsv1.DaemonSet:
			if s, ok := strategy.(appsv1.DaemonSetUpdateStrategyType); ok {
				obj.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
					Type: s,
				}
			}
		default:
			panic(fmt.Sprintf("Unsupported resource type %T for WithStrategy", resource))
		}
	}
}

// WithStartupProbe sets the readiness probe for a resource.
func WithStartupProbe(delay int32) Option {
	return func(resource client.Object) {
		probe := &corev1.Probe{
			InitialDelaySeconds: delay,
			PeriodSeconds:       5,
			FailureThreshold:    2,
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"sh", "-c", "echo 'Ready'"},
				},
			},
		}

		switch obj := resource.(type) {
		case *appsv1.Deployment:
			obj.Spec.Template.Spec.Containers[0].StartupProbe = probe
		case *appsv1.StatefulSet:
			obj.Spec.Template.Spec.Containers[0].StartupProbe = probe
		case *appsv1.DaemonSet:
			obj.Spec.Template.Spec.Containers[0].StartupProbe = probe
		}
	}
}

// WithMaxUnavailable sets the maximum unavailable replicas for a resource.
func WithMaxUnavailable(maxUnavailable *intstr.IntOrString) Option {
	return func(resource client.Object) {
		switch obj := resource.(type) {
		case *appsv1.Deployment:
			obj.Spec.Strategy.RollingUpdate.MaxUnavailable = maxUnavailable
		case *appsv1.StatefulSet:
			obj.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable = maxUnavailable
		}
	}
}
