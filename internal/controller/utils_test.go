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
)

func TestHasNewRestart(t *testing.T) {
	t.Parallel()

	t.Run("First time", func(t *testing.T) {
		t.Parallel()

		got := hasNewRestart("2024-04-03T10:00:00Z", "")
		assert.Equal(t, true, got)
	})

	t.Run("Already seen", func(t *testing.T) {
		t.Parallel()

		got := hasNewRestart("2024-04-03T10:00:00Z", "2024-04-03T10:00:00Z")
		assert.Equal(t, false, got)
	})

	t.Run("New restart", func(t *testing.T) {
		t.Parallel()

		got := hasNewRestart("2024-04-03T11:00:00Z", "2024-04-03T10:00:00Z")
		assert.Equal(t, true, got)
	})

	t.Run("No restart", func(t *testing.T) {
		t.Parallel()

		got := hasNewRestart("", "")
		assert.Equal(t, false, got)
	})
}

func TestGetRestartAnnotations(t *testing.T) {
	t.Parallel()

	const restartedAtKey = "a"
	const lastSeenKey = "b"

	t.Run("Nil map", func(t *testing.T) {
		t.Parallel()

		restartedAt, lastSeen := getRestartAnnotations(nil, restartedAtKey, lastSeenKey)
		assert.Empty(t, restartedAt)
		assert.Empty(t, lastSeen)
	})

	t.Run("Empty map", func(t *testing.T) {
		t.Parallel()

		anns := map[string]string{}
		restartedAt, lastSeen := getRestartAnnotations(anns, restartedAtKey, lastSeenKey)
		assert.Empty(t, restartedAt)
		assert.Empty(t, lastSeen)
	})

	t.Run("Only restartedAt", func(t *testing.T) {
		t.Parallel()

		anns := map[string]string{
			restartedAtKey: "now",
		}
		restartedAt, lastSeen := getRestartAnnotations(anns, restartedAtKey, lastSeenKey)
		assert.Equal(t, "now", restartedAt)
		assert.Empty(t, lastSeen)
	})

	t.Run("Both set", func(t *testing.T) {
		t.Parallel()

		anns := map[string]string{
			restartedAtKey: "now",
			lastSeenKey:    "before",
		}
		restartedAt, lastSeen := getRestartAnnotations(anns, restartedAtKey, lastSeenKey)
		assert.Equal(t, "now", restartedAt)
		assert.Equal(t, "before", lastSeen)
	})
}
