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
	"testing"
	"time"

	"github.com/containeroo/tinyflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpRequested(t *testing.T) {
	t.Parallel()

	t.Run("show version", func(t *testing.T) {
		t.Parallel()

		_, err := ParseArgs([]string{"--version"}, "1.2.3")
		assert.Error(t, err)
		assert.True(t, tinyflags.IsVersionRequested(err))
		assert.EqualError(t, err, "1.2.3")
	})

	t.Run("show help", func(t *testing.T) {
		t.Parallel()

		_, err := ParseArgs([]string{"--help"}, "0.0.0")
		assert.Error(t, err)
		assert.True(t, tinyflags.IsHelpRequested(err))
		usage := `Usage: Cascader [flags]
Flags:
        --deployment-annotation ANNOTATION             Annotation key for monitored Deployments (Default: cascader.tkb.ch/deployment)
        --statefulset-annotation ANNOTATION            Annotation key for monitored StatefulSets (Default: cascader.tkb.ch/statefulset)
        --daemonset-annotation ANNOTATION              Annotation key for monitored DaemonSets (Default: cascader.tkb.ch/daemonset)
        --last-observed-restart-annotation ANNOTATION  Annotation key for last observed restart (Default: cascader.tkb.ch/last-observed-restart)
        --requeue-after-annotation ANNOTATION          Annotation key for requeue interval override (Default: cascader.tkb.ch/requeue-after)
        --requeue-after-default DURATION               Default requeue interval (Minimum 1 Second) (Default: 5s)
        --watch-namespace NAMESPACE                    Namespaces to watch (can be repeated or comma-separated)
        --metrics-enabled <true|false>                 Enable or disable the metrics endpoint (Default: true)
        --metrics-bind-address ADDR:PORT               Metrics server address (Default: :8443)
        --metrics-secure <true|false>                  Serve metrics over HTTPS (Default: true)
        --health-probe-bind-address ADDR:PORT          Health and readiness probe address (Default: :8081)
        --enable-http2 <true|false>                    Enable HTTP/2 for servers (Default: false)
        --leader-elect <true|false>                    Enable leader election (Default: true)
        --log-encoder <json|console>                   Log format (json, console) (Default: json)
        --log-devel                                    Enable development mode logging
        --log-stacktrace-level <info|error|panic>      Stacktrace log level (Default: panic)
    -h, --help                                         show help
        --version                                      show version
`
		assert.EqualError(t, err, usage)
	})
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	t.Run("Default values", func(t *testing.T) {
		t.Parallel()

		args := []string{}
		opts, err := ParseArgs(args, "0.0.0")

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
			"--leader-elect=true",
			"--metrics-enabled=false",
			"--metrics-secure=false",
			"--enable-http2=false",
			"--log-encoder", "console",
			"--log-stacktrace-level", "info",
			"--log-devel",
		}

		opts, err := ParseArgs(args, "0.0.0")

		require.NoError(t, err)
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
		assert.False(t, opts.EnableHTTP2)
		assert.Equal(t, "console", opts.LogEncoder)
		assert.Equal(t, "info", opts.LogStacktraceLevel)
		assert.True(t, opts.LogDev)
	})

	t.Run("Invalid flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"--invalid-flag"}
		_, err := ParseArgs(args, "0.0.0")

		assert.Error(t, err)
		assert.EqualError(t, err, "unknown flag: --invalid-flag")
	})

	t.Run("Test Usage", func(t *testing.T) {
		t.Parallel()

		args := []string{"--help"}
		_, err := ParseArgs(args, "0.0.0")

		assert.Error(t, err)
		assert.True(t, tinyflags.IsHelpRequested(err))
	})

	t.Run("Test Version", func(t *testing.T) {
		t.Parallel()

		args := []string{"--version"}
		_, err := ParseArgs(args, "0.0.0")

		assert.Error(t, err)
		assert.True(t, tinyflags.IsVersionRequested(err))
	})

	t.Run("Multiple namespaces", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace=ns1", "--watch-namespace", "ns2", "--watch-namespace=ns3,ns4"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 4)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
		assert.Equal(t, "ns3", opts.WatchNamespaces[2])
		assert.Equal(t, "ns4", opts.WatchNamespaces[3])
	})

	t.Run("Multiple namespaces, comma separated", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1,ns2"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 2)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
	})

	t.Run("Multiple namespaces, mixed", func(t *testing.T) {
		t.Parallel()

		args := []string{"--watch-namespace", "ns1", "--watch-namespace", "ns2,ns3"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Len(t, opts.WatchNamespaces, 3)
		assert.Equal(t, "ns1", opts.WatchNamespaces[0])
		assert.Equal(t, "ns2", opts.WatchNamespaces[1])
		assert.Equal(t, "ns3", opts.WatchNamespaces[2])
	})

	t.Run("Valid metrics listen address (:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":8080"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (127.0.0.1:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "127.0.0.1:8080"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1:8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (localhost:8080)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", "localhost:8080"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1:8080", opts.MetricsAddr)
	})

	t.Run("Valid metrics listen address (:80)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":80"}
		opts, err := ParseArgs(args, "0.0.0")

		assert.NoError(t, err)
		assert.Equal(t, ":80", opts.MetricsAddr)
	})

	t.Run("Invalid metrics listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--metrics-bind-address", ":invalid"}
		_, err := ParseArgs(args, "0.0.0")
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid value for flag --metrics-bind-address: invalid TCP address \":invalid\": lookup tcp/invalid: unknown port.")
	})

	t.Run("Invalid probes listen address (invalid)", func(t *testing.T) {
		t.Parallel()

		args := []string{"--health-probe-bind-address", ":invalid"}
		_, err := ParseArgs(args, "0.0.0")
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid value for flag --health-probe-bind-address: invalid TCP address \":invalid\": lookup tcp/invalid: unknown port.")
	})
}
