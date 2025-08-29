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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/thurgauerkb/cascader/internal/kinds"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UniqueAnnotations ensures all provided annotation values are unique.
// Returns an error if the map is empty or if any duplicate values are found.
func UniqueAnnotations(annotations map[string]string) error {
	if len(annotations) == 0 {
		return errors.New("no annotations provided")
	}

	seen := make(map[string]string, len(annotations))
	for k, v := range annotations {
		if dupKey, ok := seen[v]; ok {
			return fmt.Errorf("duplicate annotation value %q found for keys %q and %q", v, dupKey, k)
		}
		seen[v] = k
	}
	return nil
}

// FormatAnnotations returns a sorted, comma-separated string representation of the annotations map.
func FormatAnnotations(annotations map[string]string) string {
	annotationsList := make([]string, 0, len(annotations))
	for key, val := range annotations {
		annotationsList = append(annotationsList, fmt.Sprintf("%s=%s", key, val))
	}
	sort.Strings(annotationsList) // Ensure deterministic ordering
	return strings.Join(annotationsList, ", ")
}

// ToCacheOptions returns cache.Options configured to watch the given namespaces.
// If no namespaces are provided, it returns an empty Options which watches all namespaces.
func ToCacheOptions(watchNamespaces []string) cache.Options {
	if len(watchNamespaces) == 0 {
		return cache.Options{}
	}

	nsMap := make(map[string]cache.Config, len(watchNamespaces))
	for _, ns := range watchNamespaces {
		nsMap[ns] = cache.Config{}
	}

	return cache.Options{
		DefaultNamespaces: nsMap,
	}
}

// ParseTargetRef splits a target reference (e.g. "namespace/name") into its namespace and name.
// If the reference lacks a namespace, defaultNS is used.
func ParseTargetRef(ref, defaultNS string) (namespace, name string, err error) {
	parts := strings.Split(ref, "/")
	switch len(parts) {
	case 1:
		namespace = defaultNS
		name = parts[0]
	case 2:
		namespace = parts[0]
		name = parts[1]
	default:
		err = fmt.Errorf("invalid format: %s", ref)
	}
	return
}

// GenerateID returns a unique identifier for a resource in the format "Kind/namespace/name".
func GenerateID(kind kinds.Kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

// PatchPodTemplateAnnotation updates the given annotation key in the pod template spec
// and patches the parent object using server-side merge.
func PatchPodTemplateAnnotation(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	template *corev1.PodTemplateSpec,
	key, value string,
) error {
	// Make a deep copy of the object before mutating it
	original := obj.DeepCopyObject().(client.Object)

	// Safely get and modify the annotations
	annotations := template.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	template.SetAnnotations(annotations)

	// Apply patch using MergeFrom
	if err := c.Patch(ctx, obj, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch annotation %q=%q: %w", key, value, err)
	}

	return nil
}

// PatchWorkloadAnnotation updates the given annotation key in the pod metatdata
// and patches the parent object using server-side merge.
func PatchWorkloadAnnotation(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	key, value string,
) error {
	// Make a deep copy of the object before mutating it
	original := obj.DeepCopyObject().(client.Object)

	// Safely get and modify the annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	obj.SetAnnotations(annotations)

	// Apply patch using MergeFrom
	if err := c.Patch(ctx, obj, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch annotation %q=%q: %w", key, value, err)
	}

	return nil
}

// DeleteWorkloadAnnotation removes the specified annotation key from the Pods metadata
// and patches the parent object using a server-side strategic merge.
func DeleteWorkloadAnnotation(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	key string,
) error {
	// Deep copy for patch base
	original := obj.DeepCopyObject().(client.Object)

	// Delete annotation by key
	annotations := obj.GetAnnotations()
	delete(annotations, key)
	obj.SetAnnotations(annotations)

	// Apply patch using MergeFrom
	if err := c.Patch(ctx, obj, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to delete annotation %q: %w", key, err)
	}
	return nil
}
