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

package config

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	flag "github.com/spf13/pflag"
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
		cfg, err := ParseArgs(args, &output, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "cascader.tkb.ch/deployment", cfg.DeploymentAnnotation)
		assert.Equal(t, "cascader.tkb.ch/statefulset", cfg.StatefulSetAnnotation)
		assert.Equal(t, "cascader.tkb.ch/daemonset", cfg.DaemonSetAnnotation)
		assert.Equal(t, "cascader.tkb.ch/requeue-after", cfg.RequeueAfterAnnotation)
		assert.Equal(t, "cascader.tkb.ch/last-observed-restart", cfg.LastObservedRestartAnnotation)
		assert.Equal(t, 5*time.Second, cfg.RequeueAfterDefault)
		assert.Equal(t, ":8443", cfg.MetricsAddr)
		assert.Equal(t, ":8081", cfg.ProbeAddr)
		assert.True(t, cfg.LeaderElection)
		assert.True(t, cfg.EnableMetrics)
		assert.True(t, cfg.SecureMetrics)
		assert.False(t, cfg.EnableHTTP2)
		assert.Equal(t, "json", cfg.LogEncoder)
		assert.Equal(t, "panic", cfg.LogStacktraceLevel)
		assert.False(t, cfg.LogDev)
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

		cfg, err := ParseArgs(args, &output, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "custom.deployment", cfg.DeploymentAnnotation)
		assert.Equal(t, "custom.statefulset", cfg.StatefulSetAnnotation)
		assert.Equal(t, "custom.daemonset", cfg.DaemonSetAnnotation)
		assert.Equal(t, "custom.last-observed-restart", cfg.LastObservedRestartAnnotation)
		assert.Equal(t, "custom.requeue-after", cfg.RequeueAfterAnnotation)
		assert.Equal(t, 10*time.Second, cfg.RequeueAfterDefault)
		assert.Equal(t, ":9090", cfg.MetricsAddr)
		assert.Equal(t, ":9091", cfg.ProbeAddr)
		assert.True(t, cfg.LeaderElection)
		assert.False(t, cfg.EnableMetrics)
		assert.False(t, cfg.SecureMetrics)
		assert.True(t, cfg.EnableHTTP2)
		assert.Equal(t, "console", cfg.LogEncoder)
		assert.Equal(t, "panic", cfg.LogStacktraceLevel)
		assert.True(t, cfg.LogDev)
	})

	t.Run("Invalid flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"--invalid-flag"}
		var output strings.Builder
		_, err := ParseArgs(args, &output, "0.0.0")

		assert.Error(t, err)
		assert.EqualError(t, err, "failed to parse arguments: unknown flag: --invalid-flag")
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
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, cfg.WatchNamespaces, 2)
		assert.Equal(t, "ns1", cfg.WatchNamespaces[0])
		assert.Equal(t, "ns2", cfg.WatchNamespaces[1])
	})

	t.Run("Multiple namespaces, comma separated", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1,ns2"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, cfg.WatchNamespaces, 2)
		assert.Equal(t, "ns1", cfg.WatchNamespaces[0])
		assert.Equal(t, "ns2", cfg.WatchNamespaces[1])
	})

	t.Run("Multiple namespaces, mixed", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1", "--watch-namespace", "ns2,ns3"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, cfg.WatchNamespaces, 3)
		assert.Equal(t, "ns1", cfg.WatchNamespaces[0])
		assert.Equal(t, "ns2", cfg.WatchNamespaces[1])
		assert.Equal(t, "ns3", cfg.WatchNamespaces[2])
	})

	t.Run("Valid metrics listen address (:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":8080"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":8080", cfg.MetricsAddr)
	})

	t.Run("Valid metrics listen address (127.0.0.1:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "127.0.0.1:8080"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1:8080", cfg.MetricsAddr)
	})

	t.Run("Valid metrics listen address (localhost:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "localhost:8080"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "localhost:8080", cfg.MetricsAddr)
	})

	t.Run("Valid metrics listen address (:80)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":80"}
		cfg, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":80", cfg.MetricsAddr)
	})

	t.Run("Invalid metrics listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":invalid"}
		_, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.Error(t, err)
		assert.EqualError(t, err, "invalid configuration: invalid metrics listen address: lookup tcp/invalid: unknown port")
	})

	t.Run("Invalid probes listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--health-probe-bind-address", ":invalid"}
		_, err := ParseArgs(args, io.Discard, "0.0.0")

		assert.Error(t, err)
		assert.EqualError(t, err, "invalid configuration: invalid probe listen address: lookup tcp/invalid: unknown port")
	})
}

func TestCaptureUsage(t *testing.T) {
	t.Parallel()

	t.Run("Captures flag usage output", func(t *testing.T) {
		t.Parallel()

		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.String("example-flag", "default", "Example flag description")
		fs.Usage = func() { fs.PrintDefaults() }

		output := captureUsage(fs)

		assert.Contains(t, output, "example-flag", "Expected flag name in usage output")
		assert.Contains(t, output, "Example flag description", "Expected flag description in usage output")
	})

	t.Run("Handles empty flag set", func(t *testing.T) {
		t.Parallel()

		fs := flag.NewFlagSet("empty", flag.ContinueOnError)
		fs.Usage = func() {
			_, _ = fs.Output().Write([]byte("Usage:\n"))
			fs.PrintDefaults()
		}

		output := captureUsage(fs)

		assert.NotEmpty(t, output, "Expected non-empty usage output for an empty flag set")
	})

	t.Run("Multiple flags are captured", func(t *testing.T) {
		t.Parallel()

		fs := flag.NewFlagSet("test-multi", flag.ContinueOnError)
		fs.String("flag-one", "val1", "First flag")
		fs.Int("flag-two", 42, "Second flag")
		fs.Usage = func() { fs.PrintDefaults() }

		output := captureUsage(fs)

		assert.Contains(t, output, "flag-one", "Expected 'flag-one' in usage output")
		assert.Contains(t, output, "First flag", "Expected 'First flag' description")
		assert.Contains(t, output, "flag-two", "Expected 'flag-two' in usage output")
		assert.Contains(t, output, "Second flag", "Expected 'Second flag' description")
	})
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	t.Run("Valid listen addresses (127.0.0.1:8081)", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			MetricsAddr: "localhost:9090",
			ProbeAddr:   "127.0.0.1:8081",
		}

		assert.NoError(t, cfg.Validate())
	})

	t.Run("Invalid metrics address", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			MetricsAddr: ":invalid",
			ProbeAddr:   ":8081",
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid metrics listen address: lookup tcp/invalid: unknown port")
	})

	t.Run("Invalid probe address", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			MetricsAddr: ":9090",
			ProbeAddr:   ":invalid",
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid probe listen address: lookup tcp/invalid: unknown port")
	})
}

func TestIsHelpRequested(t *testing.T) {
	t.Parallel()

	t.Run("returns true and writes message for HelpRequested error", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		helpMsg := "this is the help message\n"
		err := &HelpRequested{Message: helpMsg}

		ok := IsHelpRequested(err, buf)

		assert.True(t, ok)
		assert.Equal(t, helpMsg, buf.String())
	})

	t.Run("returns false and writes nothing for unrelated error", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		err := errors.New("some other error")

		ok := IsHelpRequested(err, buf)

		assert.False(t, ok)
		assert.Equal(t, "", buf.String())
	})
}
