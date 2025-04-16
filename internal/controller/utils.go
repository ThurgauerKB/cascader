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
	"time"

	"github.com/thurgauerkb/cascader/internal/targets"

	corev1 "k8s.io/api/core/v1"
)

// targetIDs returns the list of IDs for a slice of targets.
func targetIDs(ts []targets.Target) []string {
	ids := make([]string, len(ts))
	for i, t := range ts {
		ids[i] = t.ID()
	}
	return ids
}

// restartMarkerUpdated reports whether the restart marker differs from the last observed marker.
// If the restart marker is missing entirely, it returns true with the current timestamp.
func restartMarkerUpdated(podTemplate *corev1.PodTemplateSpec, restartedAtKey, lastObservedRestartKey string) (bool, string) {
	annotations := podTemplate.GetAnnotations()
	if annotations == nil {
		return true, time.Now().Format(time.RFC3339)
	}

	restartedAt, lastObservedAt := annotations[restartedAtKey], annotations[lastObservedRestartKey]

	return restartMarkerChanged(restartedAt, lastObservedAt), restartedAt
}

// restartMarkerChanged reports whether restartedAt differs from lastObservedAt.
func restartMarkerChanged(restartedAt, lastObservedAt string) bool {
	return restartedAt != "" && (lastObservedAt == "" || restartedAt != lastObservedAt)
}
