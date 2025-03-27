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

	appsv1 "k8s.io/api/apps/v1"

	"github.com/thurgauerkb/cascader/test/testutils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	successfullTriggerTargetMsg string = "Successfully triggered reload"
	deploymentAnnotation        string = "cascader.tkb.ch/deployment"
	statefulSetAnnotation       string = "cascader.tkb.ch/statefulset"
	daemonSetAnnotation         string = "cascader.tkb.ch/daemonset"
)

var _ = Describe("Operator in default mode", Ordered, func() {
	var ns string

	BeforeAll(func() {
		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			"--health-probe-bind-address=:8082",
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
		testutils.NSManager.Cleanup(ctx)
	})

	It("Ensure Operator is Running", func() {
		testutils.ContainsLogs("\"Deployment\",\"worker count\":1", 1*time.Minute, 2*time.Second)
	})

	// Deployment-specific tests
	It("Deployment: Single target in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj2Name),
		)

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Rolling update targets target restart after last replica", func(ctx SpecContext) {
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

		ready := 3
		By(fmt.Sprintf("Waiting until %s is fully rolled out", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desiredReplicas), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("Validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Delete single Pod in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")

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
		)
		obj2ID := testutils.GenerateID(obj2)

		By(fmt.Sprintf("deleting pod of %s", obj1ID))
		Expect(testutils.DeleteResourcePods(ctx, obj1)).ToNot(HaveOccurred(), "error deleting pods")

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Single target with Recreate strategy", func(ctx SpecContext) {
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

		obj2 := testutils.CreateDeployment(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Multiple targets in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")
		obj3Name := testutils.GenerateUniqueName("dep3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, fmt.Sprintf("%s,%s", obj2Name, obj3Name)),
		)

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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	// StatefulSet-specific tests
	It("StatefulSet: Single target in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
		)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("StatefulSet: Delete single Pod in same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("sts1")
		obj2Name := testutils.GenerateUniqueName("sts2")

		obj1 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
			testutils.WithStartupProbe(3),
		)
		obj1ID := testutils.GenerateID(obj1)

		obj2 := testutils.CreateStatefulSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		By(fmt.Sprintf("deleting pod of %s", obj1ID))
		Expect(testutils.DeleteResourcePods(ctx, obj1)).ToNot(HaveOccurred(), "error deleting pods")

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("StatefulSet: Multiple targets in different namespaces", func(ctx SpecContext) {
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	// DaemonSet-specific tests
	It("DaemonSet: Single target same namespace", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("DaemonSet: Mixed strategies in different namespaces", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("ds1")
		obj2Name := testutils.GenerateUniqueName("ds2")

		obj1 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(daemonSetAnnotation, obj2Name),
		)

		obj2 := testutils.CreateDaemonSet(
			ctx,
			ns,
			obj2Name,
			testutils.WithStrategy(appsv1.OnDeleteDaemonSetStrategyType),
			testutils.WithStartupProbe(3),
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	// Mixed workload tests
	It("Mixed: Deployment -> StatefulSet -> Deployment", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("sts2")
		obj3Name := testutils.GenerateUniqueName("dep3")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(statefulSetAnnotation, obj2Name),
		)

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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("Mixed: Deployment -> StatefulSet -> DaemonSet", func(ctx SpecContext) {
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("Mixed: Cross-namespace chain", func(ctx SpecContext) {
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("Mixed: Direct cycle detection (Deployment -> Deployment)", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj1Name),
		)
		obj1ID := testutils.GenerateID(obj1)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj1ID))
		expectedError := fmt.Sprintf("direct cycle detected: adding dependency from %s creates a cycle: %s", obj1ID, obj1ID)
		testutils.ContainsLogs(expectedError, 1*time.Minute, 2*time.Second)
	})

	It("Mixed: Indirect cycle detection (Deployment -> StatefulSet -> Deployment)", func(ctx SpecContext) {
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj1ID))
		expectedCycle := fmt.Sprintf("%s -> %s -> %s", obj1ID, obj2ID, obj3ID)
		expectedError := fmt.Sprintf("indirect cycle detected: adding dependency from %s creates a cycle: %s", obj1ID, expectedCycle)
		testutils.ContainsLogs(expectedError, 1*time.Minute, 2*time.Second)
	})

	It("StatefulSet: Rolling update targets target restart after last replica", func(ctx SpecContext) {
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

		ready := 3
		By(fmt.Sprintf("Wait unitl %s is ready", obj1ID))
		testutils.ContainsLogs(fmt.Sprintf("workload is stable: ready=%d, desired=%d", ready, desired), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Scale-up does not trigger target restart", func(ctx SpecContext) {
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
		By(fmt.Sprintf("scaling up %s from %d to %d", obj1ID, obj1.Spec.Replicas, newReplicas))
		Expect(testutils.ScaleResource(ctx, obj1, newReplicas)).To(Succeed())

		testutils.CheckResourceReadiness(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsNotLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("StatefulSet: Scale-down to 0 triggers target restart", func(ctx SpecContext) {
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
		Expect(testutils.ScaleResource(ctx, obj1, newReplicas)).To(Succeed())

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("Deployment: Scale-up from Zero targets target restart", func(ctx SpecContext) {
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
		Expect(testutils.ScaleResource(ctx, obj1, newReplicas)).To(Succeed())

		testutils.CheckResourceReadiness(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
	})

	It("StatefulSet: Scale-up from Zero targets target restart ", func(ctx SpecContext) {
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
		Expect(testutils.ScaleResource(ctx, obj1, newReplicas)).To(Succeed())

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("Invalid annotations", func(ctx SpecContext) {
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
		testutils.ContainsLogs(fmt.Sprintf("dependency cycle check failed: failed to fetch resource Deployment/%s", annotation), 1*time.Minute, 2*time.Second)
	})

	It("Overlapping dependencies", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("obj1")
		obj2Name := testutils.GenerateUniqueName("obj2")
		obj3Name := testutils.GenerateUniqueName("target")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)

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
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("Long dependency chains", func(ctx SpecContext) {
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
		dep4ID := testutils.GenerateID(dep4)

		testutils.RestartResource(ctx, obj1)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", dep4ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, dep4ID), 1*time.Minute, 2*time.Second)
	})

	It("Concurrent updates", func(ctx SpecContext) {
		obj1Name := testutils.GenerateUniqueName("dep1")
		obj2Name := testutils.GenerateUniqueName("dep2")
		obj3Name := testutils.GenerateUniqueName("target")

		obj1 := testutils.CreateDeployment(
			ctx,
			ns,
			obj1Name,
			testutils.WithAnnotation(deploymentAnnotation, obj3Name),
		)

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

		By(fmt.Sprintf("validating cascader handles concurrent updates to %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})
})

var _ = Describe("Operator watching multiple namespaces", func() {
	AfterEach(func(ctx SpecContext) {
		testutils.NSManager.Cleanup(ctx)
	})

	It("Multiple mixed targets in different namespaces", func(ctx SpecContext) {
		ns1 := testutils.NSManager.CreateNamespace(ctx)
		ns2 := testutils.NSManager.CreateNamespace(ctx)
		ns3 := testutils.NSManager.CreateNamespace(ctx)

		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			fmt.Sprintf("--watch-namespace=%s,%s,%s", ns1, ns2, ns3),
			"--health-probe-bind-address=:8084",
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

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj2ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj2ID), 1*time.Minute, 2*time.Second)

		By(fmt.Sprintf("validating cascader fetches the restart of %s", obj3ID))
		testutils.ContainsLogs(fmt.Sprintf("%q,\"targetID\":%q", successfullTriggerTargetMsg, obj3ID), 1*time.Minute, 2*time.Second)
	})

	It("ignores targets outside of watched namespaces", func(ctx SpecContext) {
		ns1 := testutils.NSManager.CreateNamespace(ctx)
		ns2 := testutils.NSManager.CreateNamespace(ctx)
		nsIgnored := testutils.NSManager.CreateNamespace(ctx)

		testutils.StartOperatorWithFlags([]string{
			"--leader-elect=false",
			fmt.Sprintf("--watch-namespace=%s,%s", ns1, ns2), // note: nsIgnored is excluded
			"--health-probe-bind-address=:8085",
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

		obj2 := testutils.CreateStatefulSet(
			ctx,
			nsIgnored,
			obj2Name,
		)
		obj2ID := testutils.GenerateID(obj2)

		testutils.RestartResource(ctx, obj1)

		By("Validating fetching resource error")
		testutils.ContainsLogs(fmt.Sprintf("dependency cycle detected: dependency cycle check failed: failed to fetch resource %s: unable to get: %s/%s because of unknown namespace for the cache", obj2ID, nsIgnored, obj2Name), 10*time.Second, 1*time.Second)
	})
})
