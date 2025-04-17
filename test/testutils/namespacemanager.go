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

package testutils

import (
	"context"

	. "github.com/onsi/gomega" // nolint:staticcheck

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespacePrefix string = "cascader-e2e-test-"

var NSManager = NewNamespaceManager()

// NamespaceManager manages test namespaces.
type NamespaceManager struct {
	namespaces []string
}

// NewNamespaceManager initializes a new NamespaceManager.
func NewNamespaceManager() *NamespaceManager {
	return &NamespaceManager{}
}

// CreateNamespace creates a unique namespace for testing.
func (m *NamespaceManager) CreateNamespace(ctx context.Context) string {
	nsName := GenerateUniqueName(namespacePrefix)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	Expect(K8sClient.Create(ctx, ns)).To(Succeed())
	m.namespaces = append(m.namespaces, nsName)

	return nsName
}

// DeleteNamespace deletes a namespace by name.
func (m *NamespaceManager) DeleteNamespace(ctx context.Context, nsName string) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	err := K8sClient.Delete(ctx, ns)
	if err != nil && !kerrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

// Cleanup deletes all namespaces created by the NamespaceManager.
func (m *NamespaceManager) Cleanup(ctx context.Context) {
	for _, ns := range m.namespaces {
		err := K8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
		if err != nil && !kerrors.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}
