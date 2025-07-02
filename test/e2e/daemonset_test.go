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

var _ = Describe("DaemonSet Workload", Serial, Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--requeue-after-default=1s",
			"--health-probe-bind-address=:3030",
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

	It("Single target same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf(
			"%q,\"workloadID\":%q",
			restartDetectedMsg,
			obj1ID,
		), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)
	},
	)

	It("Restart Pod twice", func(ctx SpecContext) { // nolint:dupl
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf(
			"%q,\"workloadID\":%q",
			restartDetectedMsg,
			obj1ID,
		), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)

		testutils.LogBuffer.Reset()

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf(
			"%q,\"workloadID\":%q",
			restartDetectedMsg,
			obj1ID,
		), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Delete Pod twice", func(ctx SpecContext) { // nolint:dupl
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf(
			"%q,\"workloadID\":%q",
			restartDetectedMsg,
			obj1ID,
		), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)

		testutils.LogBuffer.Reset()

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf(
			"%q,\"workloadID\":%q",
			restartDetectedMsg,
			obj1ID,
		), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Single target in different namespaces", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
			testutils.WithStrategy(appsv1.OnDeleteDaemonSetStrategyType),
			testutils.WithStartupProbe(3),
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)
	})
})
