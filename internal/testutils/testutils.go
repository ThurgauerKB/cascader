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
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamedError stores information about a specific object that should trigger an error.
type NamedError struct {
	Name      string // Name of the object that should trigger an error.
	Namespace string // Namespace of the object that should trigger an error.
}

// MockClientWithError wraps a fake client and simulates errors on specific conditions.
type MockClientWithError struct {
	client.Client            // Fake client that simulates errors on specific conditions.
	PatchErrorFor NamedError // Contains Name and Namespace of the object that should trigger a patch error.
	GetErrorFor   NamedError // Contains Name and Namespace of the object that should trigger a get error.
}

// Patch fails when the object's name/namespace matches `patchErrorFor`.
func (m *MockClientWithError) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if obj.GetName() == m.PatchErrorFor.Name && obj.GetNamespace() == m.PatchErrorFor.Namespace {
		return errors.New("simulated patch error")
	}
	return m.Client.Patch(ctx, obj, patch, opts...)
}

// Get fails when the object's name/namespace matches `getErrorFor`.
func (m *MockClientWithError) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if key.Name == m.GetErrorFor.Name && key.Namespace == m.GetErrorFor.Namespace {
		return errors.New("simulated get error")
	}
	return m.Client.Get(ctx, key, obj, opts...)
}

// Int32Ptr returns a pointer to the given int32 value.
func Int32Ptr(i int32) *int32 {
	return &i
}
