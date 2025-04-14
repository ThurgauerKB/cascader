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

package e2e

import (
	"fmt"
	"time"

	"github.com/thurgauerkb/cascader/test/testutils"

	. "github.com/onsi/ginkgo/v2" // nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Mixed workloads and dependency chains", Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--requeue-after-default=1s",
			"--health-probe-bind-address=:6060",
			"--metrics-enabled=false",
		})
	})

	AfterAll(func() {
		testutils.StopOperator()
	})

	BeforeEach(func(ctx SpecContext) {
		ns = testutils.NSManager.CreateNamespace(ctx)
	})

	AfterEach(func(ctx SpecContext) {
		testutils.LogBuffer.Reset()
		testutils.NSManager.Cleanup(ctx)
	})

	It("Ensure Operator is Running", func() {
		testutils.CountLogOccurrences("\"worker count\":1", 3, 15*time.Second, 2*time.Second)
	})

	It("Deployment -> StatefulSet -> Deployment", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("dep3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
			testutils.WithReplicas(3),
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
			testutils.WithStartupProbe(2),
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateDeployment(
			ctx,
			ns,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Detect restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID, obj3ID),
			30*time.Second,
			2*time.Second,
		)
	})

	It("Deployment -> StatefulSet -> DaemonSet", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("ds3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(3),
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
			testutils.WithStartupProbe(2),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj3Name),
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Detect restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID, obj3ID),
			30*time.Second,
			2*time.Second,
		)
	})

	It("Cross-namespace chain", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("ds3")

		ns2 := testutils.NSManager.CreateNamespace(ctx)
		ns3 := testutils.NSManager.CreateNamespace(ctx)

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(2),
			testutils.WithStrategy(appsv1.RecreateDeploymentStrategyType),
			testutils.WithAnnotation(statefulSetAnnotation, fmt.Sprintf("%s/%s", ns2, obj2Name)),
			testutils.WithStartupProbe(2),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns2,
			obj2Name,
			testutils.WithReplicas(3),
			testutils.WithAnnotation(daemonSetAnnotation, fmt.Sprintf("%s/%s", ns3, obj3Name)),
			testutils.WithStartupProbe(2),
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateDaemonSet(
			ctx,
			ns3,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Detect restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID, obj3ID),
			30*time.Second,
			2*time.Second,
		)
	})

	It("Multiple mixed targets in different namespaces", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("ds3")

		ns2 := testutils.NSManager.CreateNamespace(ctx)
		ns3 := testutils.NSManager.CreateNamespace(ctx)

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, fmt.Sprintf("%s/%s", ns2, obj2Name)),
			testutils.WithAnnotation(daemonSetAnnotation, fmt.Sprintf("%s/%s", ns3, obj3Name)),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns2,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateDaemonSet(
			ctx,
			ns3,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			30*time.Second,
			2*time.Second,
		)
	})
})
