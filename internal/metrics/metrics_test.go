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

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func withIsolatedPrometheusRegistry(t *testing.T, fn func()) {
	t.Helper()

	origReg := prometheus.DefaultRegisterer
	origGather := prometheus.DefaultGatherer

	// New isolated registry for this test.
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	t.Cleanup(func() {
		prometheus.DefaultRegisterer = origReg
		prometheus.DefaultGatherer = origGather
	})

	fn()
}

func resetAll(r *Registry) {
	r.dependencyCyclesDetected.Reset()
	r.workingTargets.Reset()
	r.restartsPerformed.Reset()
}

func TestRegistryMetrics_AllMethods(t *testing.T) {
	withIsolatedPrometheusRegistry(t, func() {
		r := NewRegistry(nil)
		resetAll(r)
		t.Cleanup(func() { resetAll(r) })

		t.Run("IncDependencyCycleDetected increments", func(t *testing.T) {
			resetAll(r)

			r.SetDependencyCycleDetected("ns1", "demo", "Deployment", 1)
			val := testutil.ToFloat64(r.dependencyCyclesDetected.WithLabelValues("ns1", "demo", "Deployment"))
			assert.Equal(t, float64(1), val)
		})

		t.Run("IncWorkloadTargets increments", func(t *testing.T) {
			resetAll(r)

			r.SetWorkloadTargets("ns1", "demo", "Deployment", 1)
			val := testutil.ToFloat64(r.workingTargets.WithLabelValues("ns1", "demo", "Deployment"))
			assert.Equal(t, float64(1), val)
		})

		t.Run("IncRestartsPerformed increments", func(t *testing.T) {
			resetAll(r)

			r.IncRestartsPerformed("ns1", "demo", "Deployment")
			val := testutil.ToFloat64(r.restartsPerformed.WithLabelValues("ns1", "demo", "Deployment"))
			assert.Equal(t, float64(1), val)
		})
	})
}
