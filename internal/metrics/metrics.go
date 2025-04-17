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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	CycleNone = iota
	CycleDetected
)

var (
	// DependencyCyclesDetected reports whether a dependency cycle was detected for a specific workload.
	DependencyCyclesDetected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cascader_dependency_cycles_detected",
			Help: "Indicates whether a dependency cycle is currently detected for a specific workload (1 = cycle detected, 0 = no cycle).",
		},
		[]string{"namespace", "name", "resource_kind"},
	)

	// WorkloadTargets exposes the number of referenced workloads for each annotated workload managed by Cascader.
	WorkloadTargets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cascader_workload_targets",
			Help: "Number of dependency targets extracted from a workload's annotations by Cascader.",
		},
		[]string{"namespace", "name", "resource_kind"},
	)

	// RestartsPerformed tracks the total number of restarts performed by the controller.
	RestartsPerformed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cascader_restarts_performed_total",
			Help: "Total number of restarts performed by Cascader.",
		},
		[]string{"namespace", "name", "resource_kind"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		DependencyCyclesDetected,
		WorkloadTargets,
		RestartsPerformed,
	)
}
