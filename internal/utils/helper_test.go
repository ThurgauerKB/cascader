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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thurgauerkb/cascader/internal/kinds"
)

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
