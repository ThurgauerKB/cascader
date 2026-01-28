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

import "github.com/prometheus/client_golang/prometheus"

type CycleState float64

const (
	CycleNone CycleState = iota
	CycleDetected
)

// Registry provides a typed façade for recording AutoVPA Prometheus metrics.
type Registry struct {
	reg                      prometheus.Registerer
	dependencyCyclesDetected *prometheus.GaugeVec
	workingTargets           *prometheus.GaugeVec
	restartsPerformed        *prometheus.CounterVec
}

// NewRegistry creates and registers all AutoVPA metrics with the provided
// Prometheus registerer, allowing the metrics server to expose them automatically.
func NewRegistry(reg prometheus.Registerer) *Registry {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	dependencyCyclesDetected := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cascader_dependency_cycles_detected",
			Help: "Indicates whether a dependency cycle is currently detected for a specific workload (1 = cycle detected, 0 = no cycle).",
		},
		[]string{"namespace", "name", "resource_kind"},
	)

	workloadTargets := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cascader_workload_targets",
			Help: "Number of dependency targets extracted from a workload's annotations by Cascader.",
		},
		[]string{"namespace", "name", "resource_kind"},
	)

	restartsPerformed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cascader_restarts_performed_total",
			Help: "Total number of restarts performed by Cascader.",
		},
		[]string{"namespace", "name", "resource_kind"},
	)

	reg.MustRegister(dependencyCyclesDetected, workloadTargets, restartsPerformed)

	return &Registry{
		reg:                      reg,
		dependencyCyclesDetected: dependencyCyclesDetected,
		workingTargets:           workloadTargets,
		restartsPerformed:        restartsPerformed,
	}
}

// SetDependencyCycleDetected sets the dependency cycle detected metric for the given workload.
func (r *Registry) SetDependencyCycleDetected(namespace, name, kind string, state CycleState) {
	r.dependencyCyclesDetected.WithLabelValues(namespace, name, kind).Set(float64(state))
}

// SetWorkloadTargets sets the number of dependency targets extracted from a workload's annotations by Cascader.
func (r *Registry) SetWorkloadTargets(namespace, name, kind string, value float64) {
	r.workingTargets.WithLabelValues(namespace, name, kind).Set(value)
}

// IncRestartsPerformed increments the total number of restarts performed by Cascader.
func (r *Registry) IncRestartsPerformed(namespace, name, kind string) {
	r.restartsPerformed.WithLabelValues(namespace, name, kind).Inc()
}
