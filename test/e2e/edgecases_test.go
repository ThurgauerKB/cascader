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

var _ = Describe("Edge cases", Serial, Ordered, func() {
	AfterEach(func() {
		testutils.StopOperator()
	})

	It("Invalid annotations", func(ctx SpecContext) {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5050",
			"--metrics-enabled=false",
		})

		ns := testutils.NSManager.CreateNamespace(ctx)

		obj1Name := testutils.GenerateUniqueName("dep1")
		annotation := "invalid/reference"

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, annotation),
		)
		obj1ID := testutils.GenerateID(obj1)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader logs invalid annotation for %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("dependency cycle check failed: failed to fetch resource Deployment/%s", annotation),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Overlapping dependencies", func(ctx SpecContext) {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5051",
			"--metrics-enabled=false",
		})

		ns := testutils.NSManager.CreateNamespace(ctx)

		obj1Name := testutils.GenerateUniqueName("obj1")
		obj2Name := testutils.GenerateUniqueName("obj2")
		obj3Name := testutils.GenerateUniqueName("target")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)

		obj3 := testutils.CreateDeployment(
			ctx,
			ns,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		testutils.RestartResource(ctx, obj1)
		testutils.RestartResource(ctx, obj2)

		By(fmt.Sprintf("validating cascader fetches the restart of %s only once", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Long dependency chains", func(ctx SpecContext) {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5052",
			"--metrics-enabled=false",
		})

		ns := testutils.NSManager.CreateNamespace(ctx)

		obj1Name := testutils.GenerateUniqueName("obj1")
		obj2Name := testutils.GenerateUniqueName("obj2")
		obj3Name := testutils.GenerateUniqueName("obj3")
		dep4Name := testutils.GenerateUniqueName("dep4")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
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
			testutils.WithAnnotation(deploymentAnnotation, dep4Name),
		)
		obj3ID := testutils.GenerateID(obj3)

		dep4 := testutils.CreateDeployment(
			ctx,
			ns,
			dep4Name,
		)
		obj4ID := testutils.GenerateID(dep4)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID, obj3ID),
			1*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj4ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID, obj4ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Concurrent updates", func(ctx SpecContext) {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5053",
			"--metrics-enabled=false",
		})

		ns := testutils.NSManager.CreateNamespace(ctx)

		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")
		obj3Name := testutils.GenerateUniqueName("target")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)

		obj3 := testutils.CreateDeployment(
			ctx,
			ns,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		go testutils.RestartResource(ctx, obj1)
		go testutils.RestartResource(ctx, obj2)

		By(fmt.Sprintf("validating cascader handles concurrent updates to %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Detect multiple restarts", func(ctx SpecContext) {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5054",
			"--metrics-enabled=false",
		})

		ns := testutils.NSManager.CreateNamespace(ctx)

		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")
		obj3Name := testutils.GenerateUniqueName("target")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)

		obj3 := testutils.CreateDeployment(
			ctx,
			ns,
			obj3Name,
		)
		obj3ID := testutils.GenerateID(obj3)

		go testutils.RestartResource(ctx, obj1)
		go testutils.RestartResource(ctx, obj2)

		By("validating cascader detects multiple restarts")
		testutils.CountLogOccurrences(restartDetectedMsg, 2, 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader handles concurrent updates to %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Explicit enable HTTP2", func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5055",
			"--metrics-enabled=false",
			"--enable-http2=true",
		})

		By("Validating logs")
		testutils.ContainsNotLogs("disabling HTTP/2 for compatibility", 20*time.Second, 2*time.Second)
	})

	It("Explicit disable HTTP2", func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:5056",
			"--metrics-enabled=false",
			"--enable-http2=false",
		})

		By("Validating logs")
		testutils.ContainsLogs("disabling HTTP/2 for compatibility", 20*time.Second, 2*time.Second)
	})
})
