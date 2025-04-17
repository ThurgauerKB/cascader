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
	"os/exec"
	"testing"

	"github.com/thurgauerkb/cascader/test/testutils"

	. "github.com/onsi/ginkgo/v2" // nolint:staticcheck
	. "github.com/onsi/gomega"    // nolint:staticcheck

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	testCfg    *rest.Config
	testScheme = runtime.NewScheme()
)

const (
	successfullTriggerTargetMsg string = "Successfully triggered reload"
	restartDetectedMsg          string = "Restart detected, handling targets"
	deploymentAnnotation        string = "cascader.tkb.ch/deployment"
	statefulSetAnnotation       string = "cascader.tkb.ch/statefulset"
	daemonSetAnnotation         string = "cascader.tkb.ch/daemonset"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cascader E2E Suite")
}

var _ = BeforeSuite(func() {
	By("Using existing cluster configuration")
	var err error
	testCfg, err = config.GetConfig()
	Expect(err).NotTo(HaveOccurred())
	Expect(testCfg).NotTo(BeNil())

	By("Adding schemes")
	Expect(appsv1.AddToScheme(testScheme)).To(Succeed())
	Expect(corev1.AddToScheme(testScheme)).To(Succeed())

	// Setup the test environment
	testutils.K8sClient, err = client.New(testCfg, client.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred())

	testutils.NSManager = testutils.NewNamespaceManager()

	// Build the operator
	cmd := exec.Command("go", "build", "-o", "../../bin/cascader", "../../cmd/")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err = cmd.Run()
	Expect(err).ToNot(HaveOccurred(), "operator exited unexpectedly")
})

var _ = AfterSuite(func(ctx SpecContext) {
	By("Deleting namespaces")
	testutils.NSManager.Cleanup(ctx)

	By("Stopping operator manager")
	testutils.StopOperator()
})
