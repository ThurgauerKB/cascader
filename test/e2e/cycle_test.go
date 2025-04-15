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

var _ = Describe("Cycle Detection", Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--requeue-after-default=1s",
			"--health-probe-bind-address=:2020",
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

	It("Direct cycle detection (Deployment -> Deployment)", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj1Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Fetches restart of %s", obj1ID))
		expectedError := fmt.Sprintf("direct cycle detected: adding dependency from %q creates a direct cycle: %s", obj1ID, obj1ID)
		testutils.ContainsLogs(expectedError, 1*time.Minute, 2*time.Second)
	})

	It("Indirect cycle detection (Deployment -> StatefulSet -> Deployment)", func(ctx SpecContext) {
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
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)
		obj2ID := testutils.GenerateID(obj2)

		obj3 := testutils.CreateDeployment(
			ctx,
			ns,
			obj3Name,
			testutils.WithAnnotation(deploymentAnnotation, obj1Name),
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Fetches restart of %s", obj1ID))
		expectedCycle := fmt.Sprintf("%s -> %s -> %s", obj1ID, obj2ID, obj3ID)
		expectedError := fmt.Sprintf("indirect cycle detected: adding dependency from %q creates a indirect cycle: %s", obj1ID, expectedCycle)
		testutils.ContainsLogs(expectedError, 1*time.Minute, 2*time.Second)
	})
})
