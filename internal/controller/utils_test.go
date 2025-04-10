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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestRestartMarkerUpdated(t *testing.T) {
	t.Parallel()

	restartedAtKey := "restart"
	lastObservedKey := "last"

	t.Run("No annotations", func(t *testing.T) {
		t.Parallel()

		changed, restartedAt := restartMarkerUpdated(&corev1.PodTemplateSpec{}, restartedAtKey, lastObservedKey)
		assert.False(t, changed)
		assert.Empty(t, restartedAt)
	})

	t.Run("First seen", func(t *testing.T) {
		t.Parallel()

		podTemplate := &corev1.PodTemplateSpec{}
		podTemplate.Annotations = map[string]string{
			restartedAtKey: "now",
		}

		changed, restartedAt := restartMarkerUpdated(podTemplate, restartedAtKey, lastObservedKey)
		assert.True(t, changed)
		assert.Equal(t, "now", restartedAt)
	})

	t.Run("Already observed", func(t *testing.T) {
		t.Parallel()

		podTemplate := &corev1.PodTemplateSpec{}
		podTemplate.Annotations = map[string]string{
			restartedAtKey:  "now",
			lastObservedKey: "now",
		}

		changed, restartedAt := restartMarkerUpdated(podTemplate, restartedAtKey, lastObservedKey)
		assert.False(t, changed)
		assert.Equal(t, "now", restartedAt)
	})

	t.Run("Changed since last observed", func(t *testing.T) {
		t.Parallel()

		podTemplate := &corev1.PodTemplateSpec{}
		podTemplate.Annotations = map[string]string{
			restartedAtKey:  "new",
			lastObservedKey: "old",
		}

		changed, restartedAt := restartMarkerUpdated(podTemplate, restartedAtKey, lastObservedKey)
		assert.True(t, changed)
		assert.Equal(t, "new", restartedAt)
	})
}

func TestRestartChanged(t *testing.T) {
	t.Parallel()

	t.Run("First time seen", func(t *testing.T) {
		t.Parallel()

		assert.True(t, restartMarkerChanged("2024-04-03T10:00:00Z", ""))
	})

	t.Run("Already seen", func(t *testing.T) {
		t.Parallel()

		assert.False(t, restartMarkerChanged("2024-04-03T10:00:00Z", "2024-04-03T10:00:00Z"))
	})

	t.Run("New restart", func(t *testing.T) {
		t.Parallel()

		assert.True(t, restartMarkerChanged("2024-04-03T11:00:00Z", "2024-04-03T10:00:00Z"))
	})

	t.Run("No restart timestamp", func(t *testing.T) {
		t.Parallel()

		assert.False(t, restartMarkerChanged("", ""))
	})
}
