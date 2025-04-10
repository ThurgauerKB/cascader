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

package app

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("Smoke", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		args := []string{
			"--leader-elect=false",
			"--health-probe-bind-address", ":8082",
			"--watch-namespace=test-cascader",
			"--metrics-enabled=false",
		}
		out := &bytes.Buffer{}

		errCh := make(chan error, 1)
		go func() {
			errCh <- Run(ctx, "v0.0.0", args, out)
		}()

		time.Sleep(2 * time.Second)
		cancel()

		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Run returned an error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Run did not return within the expected time")
		}
	})

	t.Run("Invalid args", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		args := []string{"--invalid-flag"}
		out := &bytes.Buffer{}

		err := Run(ctx, "v0.0.0", args, out)

		assert.Error(t, err)
		assert.EqualError(t, err, "error parsing arguments: failed to parse arguments: unknown flag: --invalid-flag")
	})

	t.Run("Request Help", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		args := []string{"--version"}
		out := &bytes.Buffer{}

		err := Run(ctx, "v0.0.0", args, out)

		assert.NoError(t, err)
		assert.Equal(t, out.String(), "Cascader version v0.0.0\n")
	})

	t.Run("Logger error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		args := []string{"--log-encoder", "invalid"}
		out := &bytes.Buffer{}

		err := Run(ctx, "v0.0.0", args, out)

		assert.Error(t, err)
		assert.EqualError(t, err, "error setting up logger: invalid log encoder: \"invalid\"")
	})

	t.Run("Leader Election", func(t *testing.T) {
		ctx := context.Background()
		args := []string{
			"--health-probe-bind-address", ":8082",
		}
		out := &bytes.Buffer{}

		err := Run(ctx, "v0.0.0", args, out)

		assert.Error(t, err)
		assert.EqualError(t, err, "unable to create manager: unable to find leader election namespace: not running in-cluster, please specify LeaderElectionNamespace")
	})

	t.Run("Not unique Annotations", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		args := []string{
			"--health-probe-bind-address", ":8085",
			"--leader-elect=false",
			"--last-observed-restart-annotation", "cascader.tkb.ch/last-observed-restart",
			"--deployment-annotation", "cascader.tkb.ch/deployment",
			"--statefulset-annotation", "cascader.tkb.ch/deployment",
			"--daemonset-annotation", "cascader.tkb.ch/daemonset",
		}
		out := &bytes.Buffer{}

		err := Run(ctx, "v0.0.0", args, out)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "annotation values must be unique: duplicate annotation")
	})
}
