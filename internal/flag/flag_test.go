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

package flag

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHelpRequested(t *testing.T) {
	t.Parallel()

	t.Run("Error Message", func(t *testing.T) {
		t.Parallel()

		err := &HelpRequested{Message: "This is a help message"}
		assert.EqualError(t, err, "This is a help message")
	})

	t.Run("As HelpRequested", func(t *testing.T) {
		t.Parallel()

		err := &HelpRequested{Message: "Help requested"}
		assert.IsType(t, &HelpRequested{}, err, "As() should return true for HelpRequested type")
	})
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	t.Run("Default values", func(t *testing.T) {
		t.Parallel()

		args := []string{}
		var output strings.Builder
		opts, err := ParseArgs(args, &output, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "cascader.tkb.ch/deployment", opts.DeploymentAnnotation)
		assert.Equal(t, "cascader.tkb.ch/statefulset", opts.StatefulSetAnnotation)
		assert.Equal(t, "cascader.tkb.ch/daemonset", opts.DaemonSetAnnotation)
		assert.Equal(t, "cascader.tkb.ch/requeue-after", opts.RequeueAfterAnnotation)
		assert.Equal(t, "cascader.tkb.ch/last-observed-restart", opts.LastObservedRestartAnnotation)
		assert.Equal(t, 5*time.Second, opts.RequeueAfterDefault)
		assert.Equal(t, ":8443", opts.MetricsAddr)
		assert.Equal(t, ":8081", opts.ProbeAddr)
		assert.True(t, opts.LeaderElection)
		assert.True(t, opts.EnableMetrics)
		assert.True(t, opts.SecureMetrics)
		assert.False(t, opts.EnableHTTP2)
		assert.Equal(t, "json", opts.LogEncoder)
		assert.Equal(t, "panic", opts.LogStacktraceLevel)
		assert.False(t, opts.LogDev)
	})

	t.Run("Override values", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--deployment-annotation", "custom.deployment",
			"--statefulset-annotation", "custom.statefulset",
			"--daemonset-annotation", "custom.daemonset",
			"--last-observed-restart-annotation", "custom.last-observed-restart",
			"--requeue-after-annotation", "custom.requeue-after",
			"--requeue-after-default", "10s",
			"--metrics-bind-address", ":9090",
			"--health-probe-bind-address", ":9091",
			"--leader-elect",
			"--metrics-enabled=false",
			"--metrics-secure=false",
			"--enable-http2",
			"--log-encoder", "console",
			"--log-stacktrace-level", "panic",
			"--log-devel",
		}

		var output strings.Builder

		opts, err := ParseArgs(args, &output, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "custom.deployment", opts.DeploymentAnnotation)
		assert.Equal(t, "custom.statefulset", opts.StatefulSetAnnotation)
		assert.Equal(t, "custom.daemonset", opts.DaemonSetAnnotation)
		assert.Equal(t, "custom.last-observed-restart", opts.LastObservedRestartAnnotation)
		assert.Equal(t, "custom.requeue-after", opts.RequeueAfterAnnotation)
		assert.Equal(t, 10*time.Second, opts.RequeueAfterDefault)
		assert.Equal(t, ":9090", opts.MetricsAddr)
		assert.Equal(t, ":9091", opts.ProbeAddr)
		assert.True(t, opts.LeaderElection)
		assert.False(t, opts.EnableMetrics)
		assert.False(t, opts.SecureMetrics)
		assert.True(t, opts.EnableHTTP2)
		assert.Equal(t, "console", opts.LogEncoder)
		assert.Equal(t, "panic", opts.LogStacktraceLevel)
		assert.True(t, opts.LogDev)
	})

	t.Run("Invalid flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"--invalid-flag"}
		var output strings.Builder
		_, err := ParseArgs(args, &output, "0.0.0")

		assert.Error(t, err)
		assert.EqualError(t, err, "unknown flag: --invalid-flag")
	})

	t.Run("Test Usage", func(t *testing.T) {
		t.Parallel()

		args := []string{"--help"}
		var output strings.Builder
		_, err := ParseArgs(args, &output, "0.0.0")

		assert.IsType(t, &HelpRequested{}, err)
		assert.Error(t, err)
	})

	t.Run("Test Version", func(t *testing.T) {
		t.Parallel()

		args := []string{"--version"}
		var output strings.Builder
		_, err := ParseArgs(args, &output, "0.0.0")

		assert.IsType(t, &HelpRequested{}, err)
		assert.Error(t, err)
		assert.EqualError(t, err, "Cascader version 0.0.0")
	})

	t.Run("Multiple namespaces", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1", "--watch-namespace", "ns2"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 2)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
	})

	t.Run("Multiple namespaces, comma separated", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1,ns2"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 2)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
	})

	t.Run("Multiple namespaces, mixed", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1", "--watch-namespace", "ns2,ns3"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 3)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
		assert.Equal(t, "ns3", opts.WatchNamespaces[2])
	})

	t.Run("Valid metrics listen address (:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":8080"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (127.0.0.1:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "127.0.0.1:8080"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1:8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (localhost:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "localhost:8080"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "localhost:8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (:80)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":80"}
		opts, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":80", opts.MetricsAddr)
	})

	t.Run("Invalid metrics listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":invalid"}
		flags, err := ParseArgs(args, io.Discard, "0.0.0")
		assert.NoError(t, err)

		err = flags.Validate()

		assert.Error(t, err)
		assert.EqualError(t, err, "invalid metrics listen address: lookup tcp/invalid: unknown port")
	})

	t.Run("Invalid probes listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--health-probe-bind-address", ":invalid"}
		flags, err := ParseArgs(args, io.Discard, "0.0.0")
		assert.NoError(t, err)

		err = flags.Validate()
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid probe listen address: lookup tcp/invalid: unknown port")
	})
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("Valid listen addresses (127.0.0.1:8081)", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			MetricsAddr: "localhost:9090",
			ProbeAddr:   "127.0.0.1:8081",
		}

		assert.NoError(t, opts.Validate())
	})

	t.Run("Invalid metrics address", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			MetricsAddr: ":invalid",
			ProbeAddr:   ":8081",
		}

		err := opts.Validate()
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid metrics listen address: lookup tcp/invalid: unknown port")
	})

	t.Run("Invalid probe address", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			MetricsAddr: ":9090",
			ProbeAddr:   ":invalid",
		}

		err := opts.Validate()
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid probe listen address: lookup tcp/invalid: unknown port")
	})
}
