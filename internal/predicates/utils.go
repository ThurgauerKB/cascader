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

package predicates

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/thurgauerkb/cascader/internal/kinds"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// hasAnnotation returns true if obj contains any of the specified annotations.
func hasAnnotation(obj client.Object, annotations kinds.AnnotationKindMap) bool {
	objAnnots := obj.GetAnnotations()
	if objAnnots == nil {
		return false
	}
	for a := range annotations {
		if _, ok := objAnnots[a]; ok {
			return true
		}
	}
	return false
}

// SpecChanged returns true if the PodTemplateSpec differs between old and new objects.
func SpecChanged(oldObj, newObj client.Object) bool {
	oldTpl, err := extractPodTemplate(oldObj)
	if err != nil {
		return false
	}
	newTpl, err := extractPodTemplate(newObj)
	if err != nil {
		return false
	}

	oldHash, err := hashTemplate(*oldTpl)
	if err != nil {
		return false
	}
	newHash, err := hashTemplate(*newTpl)
	if err != nil {
		return false
	}

	return oldHash != newHash
}

// extractPodTemplate extracts the PodTemplateSpec from a supported resource.
func extractPodTemplate(obj client.Object) (*corev1.PodTemplateSpec, error) {
	switch res := obj.(type) {
	case *appsv1.Deployment:
		return &res.Spec.Template, nil
	case *appsv1.StatefulSet:
		return &res.Spec.Template, nil
	case *appsv1.DaemonSet:
		return &res.Spec.Template, nil
	default:
		return nil, fmt.Errorf("unsupported object type: %T", obj)
	}
}

// hashTemplate computes a 64-bit FNV-1 hash of a PodTemplateSpec.
func hashTemplate(t corev1.PodTemplateSpec) (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to serialize PodTemplateSpec: %w", err)
	}
	h := fnv.New64a()
	if _, err := h.Write(data); err != nil {
		return "", fmt.Errorf("failed to hash data: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}

// getReplicas returns the number of replicas for an object, or -1 if unsupported.
func getReplicas(obj client.Object) int32 {
	switch res := obj.(type) {
	case *appsv1.Deployment:
		if res.Spec.Replicas != nil {
			return *res.Spec.Replicas
		}
	case *appsv1.StatefulSet:
		if res.Spec.Replicas != nil {
			return *res.Spec.Replicas
		}
	case *appsv1.DaemonSet:
		return res.Status.DesiredNumberScheduled // DaemonSets don't have .Spec.Replicas
	}
	return -1
}

// ScaledToZero returns true if replicas dropped to zero.
func ScaledToZero(oldObj, newObj client.Object) bool {
	return getReplicas(oldObj) > 0 && getReplicas(newObj) == 0
}

// ScaledFromZero returns true if replicas increased from zero.
func ScaledFromZero(oldObj, newObj client.Object) bool {
	return getReplicas(oldObj) == 0 && getReplicas(newObj) > 0
}

// SingleReplicaPodDeleted returns true if a single-replica workload lost its pod.
func SingleReplicaPodDeleted(oldObj, newObj client.Object) bool {
	switch res := oldObj.(type) {
	case *appsv1.Deployment:
		dep, ok := newObj.(*appsv1.Deployment)
		if !ok || res.Spec.Replicas == nil || dep.Spec.Replicas == nil || *res.Spec.Replicas != 1 {
			return false
		}
		return res.Status.ReadyReplicas == 1 && dep.Status.ReadyReplicas == 0 &&
			res.Status.AvailableReplicas == 1 && dep.Status.AvailableReplicas == 0

	case *appsv1.StatefulSet:
		sts, ok := newObj.(*appsv1.StatefulSet)
		if !ok || res.Spec.Replicas == nil || sts.Spec.Replicas == nil || *res.Spec.Replicas != 1 {
			return false
		}
		return res.Status.ReadyReplicas == 1 && sts.Status.ReadyReplicas == 0
	}
	return false
}

// DaemonSetTransitioning returns true if a DaemonSet is updating pods or has unavailable pods.
func DaemonSetTransitioning(obj client.Object) bool {
	ds, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return false
	}
	return ds.Status.UpdatedNumberScheduled != ds.Status.DesiredNumberScheduled || ds.Status.NumberUnavailable > 0
}
