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

	"github.com/thurgauerkb/cascader/internal/utils"
	"github.com/thurgauerkb/cascader/test/testutils"

	. "github.com/onsi/ginkgo/v2" // nolint:staticcheck
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Deployment workload", Serial, Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--requeue-after-default=1s",
			"--health-probe-bind-address=:4040",
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
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithStartupProbe(5),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)
	})

	It("Restart Pod twice", func(ctx SpecContext) { // nolint:dupl
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithStartupProbe(5),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)

		testutils.LogBuffer.Reset()

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)
	})

	It("Delete Pod twice", func(ctx SpecContext) { // nolint:dupl
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithAnnotation(utils.LastObservedRestartKey, time.Now().Format(time.RFC3339)),
			testutils.WithStartupProbe(5),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)

		testutils.LogBuffer.Reset()

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)
	})

	It("Delete pod of single replica", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithReplicas(1),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.DeleteRandomPod(ctx, obj1)

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
			2*time.Second)
	})

	It("Delete pod of multiple replicas", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithReplicas(3),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.DeleteRandomPod(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			14*time.Second,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			6*time.Second,
			2*time.Second,
		)
	})

	It("Rolling update triggers target restart after last replica", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		desiredReplicas := int32(3)

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithStrategy(appsv1.RollingUpdateDeploymentStrategyType),
			testutils.WithReplicas(desiredReplicas),
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithStartupProbe(2),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		ready := 3
		By(fmt.Sprintf("Waiting until %s is fully rolled out", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desiredReplicas),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)
	})

	It("Recreate strategy triggers target restart", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(2),
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithStrategy(appsv1.RecreateDeploymentStrategyType),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second)
	})

	It("Multiple targets in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")
		obj3Name := testutils.GenerateUniqueName("dep3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, fmt.Sprintf("%s,%s", obj2Name, obj3Name)),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
			testutils.WithReplicas(3),
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
			1*time.Minute,
			2*time.Second)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj3ID))
		testutils.ContainsLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj3ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Scale-up does not trigger target restart", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(2),
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
			testutils.WithStartupProbe(2),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		newReplicas := int32(4)
		By(fmt.Sprintf("Scaling up %s from %d to %d", obj1ID, obj1.Spec.Replicas, newReplicas))
		testutils.ScaleResource(ctx, obj1, newReplicas)

		testutils.CheckResourceReadiness(ctx, obj1)

		By(fmt.Sprintf("Detect restart of %s", obj1ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q", restartDetectedMsg, obj1ID),
			1*time.Minute,
			2*time.Second,
		)

		By(fmt.Sprintf("Fetches restart of %s", obj2ID))
		testutils.ContainsNotLogs(
			fmt.Sprintf("%q,\"workloadID\":%q,\"targetID\":%q", successfullTriggerTargetMsg, obj1ID, obj2ID),
			1*time.Minute,
			2*time.Second,
		)
	})

	It("Scale-up from Zero triggers target restart", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithReplicas(0),
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
			testutils.WithStartupProbe(2),
		)
		obj2ID := testutils.GenerateID(obj2)

		newReplicas := int32(3)
		By(fmt.Sprintf("scaling up %s from %d to %d", obj1ID, obj1.Spec.Replicas, newReplicas))
		testutils.ScaleResource(ctx, obj1, newReplicas)

		testutils.CheckResourceReadiness(ctx, obj1)

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
