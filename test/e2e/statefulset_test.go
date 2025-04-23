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
)

var _ = Describe("StatefulSet workload", Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--requeue-after-default=1s",
			"--health-probe-bind-address=:9090",
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

	It("Single target in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
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
		)
		obj2ID := testutils.GenerateID(obj2)

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
	})

	It("Restart Pod twice", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
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
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)

		testutils.DeleteRandomPod(ctx, obj1)

		testutils.LogBuffer.Reset()

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
	})

	It("Delete Pod with multiple replicas", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
			testutils.WithReplicas(4),
			testutils.WithStartupProbe(3),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			15*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			15*time.Second,
			2*time.Second,
		)
	})

	It("Multiple targets in different namespaces", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("sts3")

		ns2 := testutils.NSManager.CreateNamespace(ctx)
		ns3 := testutils.NSManager.CreateNamespace(ctx)

		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(3),
			testutils.WithAnnotation(statefulSetAnnotation, fmt.Sprintf("%s/%s,%s/%s", ns2, obj2Name, ns3, obj3Name)),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns2,
			obj2Name,
			testutils.WithReplicas(2),
			testutils.WithStartupProbe(2),
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateStatefulSet(
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
			2*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			2*time.Minute,
			2*time.Second,
		)
	})

	It("Rolling update triggers target restart after last replica", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		desired := int32(3)
		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(desired),
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
			testutils.WithStartupProbe(2),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			30*time.Second,
			2*time.Second,
		)

		ready := 3
		By(fmt.Sprintf("Wait unitl %s is ready", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desired),
			30*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			30*time.Second,
			2*time.Second,
		)
	})

	It("Scale-down to 0 triggers target restart", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
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
		)
		obj2ID := testutils.GenerateID(obj2)

		newReplicas := int32(0)
		By(fmt.Sprintf("scaling down %s from %d to %d", obj1ID, obj1.Spec.Replicas, newReplicas))
		testutils.ScaleResource(ctx, obj1, newReplicas)

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
	})

	It("Scale-up from Zero triggers target restart ", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(0),
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		newReplicas := int32(3)
		By(fmt.Sprintf("scaling up %s from %d to %d", obj1ID, obj1.Spec.Replicas, newReplicas))
		testutils.ScaleResource(ctx, obj1, newReplicas)

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
	})
})
