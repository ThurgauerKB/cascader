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

var _ = Describe("Operator watching multiple namespaces", Ordered, func() {
	AfterEach(func(ctx SpecContext) {
		testutils.NSManager.Cleanup(ctx)
		testutils.StopOperator()
	})

	It("Multiple mixed targets in different namespaces", func(ctx SpecContext) {
		ns1 := testutils.NSManager.CreateNamespace(ctx)
		ns2 := testutils.NSManager.CreateNamespace(ctx)
		ns3 := testutils.NSManager.CreateNamespace(ctx)

		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			fmt.Sprintf("--watch-namespace=%s,%s,%s", ns1, ns2, ns3),
			"--health-probe-bind-address=:7070",
			"--metrics-enabled=false",
		})

		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("ds3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns1,
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

	It("Ignores targets outside of watched namespaces", func(ctx SpecContext) {
		ns1 := testutils.NSManager.CreateNamespace(ctx)
		ns2 := testutils.NSManager.CreateNamespace(ctx)
		nsIgnored := testutils.NSManager.CreateNamespace(ctx)

		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			fmt.Sprintf("--watch-namespace=%s,%s", ns1, ns2), // note: nsIgnored is excluded
			"--health-probe-bind-address=:7071",
			"--metrics-enabled=false",
		})

		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts-ignored")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns1,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, fmt.Sprintf("%s/%s", nsIgnored, obj2Name)),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			nsIgnored,
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

		By("Validating fetching resource error")
		testutils.ContainsLogs(
			fmt.Sprintf("dependency cycle detected: dependency cycle check failed: failed to fetch resource %s: unable to get: %s/%s because of unknown namespace for the cache", obj2ID, nsIgnored, obj2Name),
			10*time.Second,
			1*time.Second,
		)
	})
})
