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
	"fmt"
	"net"
	"time"

	"github.com/containeroo/tinyflags"
)

const (
	daemonSetAnnotation           string = "cascader.tkb.ch/daemonset"
	deploymentAnnotation          string = "cascader.tkb.ch/deployment"
	statefulSetAnnotation         string = "cascader.tkb.ch/statefulset"
	lastObservedRestartAnnotation string = "cascader.tkb.ch/last-observed-restart"
	requeueAfterAnnotation        string = "cascader.tkb.ch/requeue-after"
)

// Options holds all configuration options for the application.
type Options struct {
	WatchNamespaces               []string      // Namespaces to watch
	MetricsAddr                   string        // Address for the metrics server
	LeaderElection                bool          // Enable leader election
	ProbeAddr                     string        // Address for health and readiness probes
	SecureMetrics                 bool          // Serve metrics over HTTPS
	EnableHTTP2                   bool          // Enable HTTP/2 for servers
	DeploymentAnnotation          string        // Annotation key for monitored Deployments
	StatefulSetAnnotation         string        // Annotation key for monitored StatefulSets
	DaemonSetAnnotation           string        // Annotation key for monitored DaemonSets
	LastObservedRestartAnnotation string        // Annotation key for last observed restart
	RequeueAfterAnnotation        string        // Annotation key for requeue interval
	RequeueAfterDefault           time.Duration // Default requeue interval
	EnableMetrics                 bool          // Enable or disable metrics
	LogEncoder                    string        // Log format: "json" or "console"
	LogStacktraceLevel            string        // Stacktrace log level
	LogDev                        bool          // Enable development logging mode
}

// ParseArgs parses CLI flags into Options and handles --help/--version output.
func ParseArgs(args []string, version string) (Options, error) {
	options := Options{}

	tf := tinyflags.NewFlagSet("Cascader", tinyflags.ContinueOnError)
	tf.Version(version)

	tf.StringVar(&options.DeploymentAnnotation, "deployment-annotation", deploymentAnnotation, "Annotation key for monitored Deployments").
		Placeholder("ANN").
		Value()
	tf.StringVar(&options.StatefulSetAnnotation, "statefulset-annotation", statefulSetAnnotation, "Annotation key for monitored StatefulSets").
		Placeholder("ANN").
		Value()
	tf.StringVar(&options.DaemonSetAnnotation, "daemonset-annotation", daemonSetAnnotation, "Annotation key for monitored DaemonSets").
		Placeholder("ANN").
		Value()
	tf.StringVar(&options.LastObservedRestartAnnotation, "last-observed-restart-annotation", lastObservedRestartAnnotation, "Annotation key for last observed restart").
		Placeholder("ANN").
		Value()
	tf.StringVar(&options.RequeueAfterAnnotation, "requeue-after-annotation", requeueAfterAnnotation, "Annotation key for requeue interval override").
		Placeholder("ANN").
		Value()

	tf.DurationVar(&options.RequeueAfterDefault, "requeue-after-default", 5*time.Second, "Default requeue interval (Minimum 2 Seconds)").
		Validate(func(d time.Duration) error {
			if d < 2*time.Second {
				return fmt.Errorf("requeue-after-default must be at least 2 seconds")
			}
			return nil
		}).
		Placeholder("DUR").
		Value()

	tf.StringSliceVar(&options.WatchNamespaces, "watch-namespace", nil, "Namespaces to watch (can be repeated or comma-separated)").
		Placeholder("NS").
		Value()

	tf.BoolVar(&options.EnableMetrics, "metrics-enabled", true, "Enable or disable the metrics endpoint").
		Strict().
		Value()

	metricsBindAddress := tf.TCPAddr("metrics-bind-address", &net.TCPAddr{IP: nil, Port: 8443}, "Metrics server address").
		Placeholder("ADDR:PORT").
		Value()
	tf.BoolVar(&options.SecureMetrics, "metrics-secure", true, "Serve metrics over HTTPS").
		Strict().
		Value()

	healthProbeaddress := tf.TCPAddr("health-probe-bind-address", &net.TCPAddr{IP: nil, Port: 8081}, "Health and readiness probe address").
		Placeholder("ADDR:PORT").
		Value()
	tf.BoolVar(&options.EnableHTTP2, "enable-http2", false, "Enable HTTP/2 for servers").
		Value()
	tf.BoolVar(&options.LeaderElection, "leader-elect", true, "Enable leader election").
		Strict().
		Value()

	tf.StringVar(&options.LogEncoder, "log-encoder", "json", "Log format (json, console)").
		Choices("json", "console").
		Value()

	tf.BoolVar(&options.LogDev, "log-devel", false, "Enable development mode logging").Value()
	tf.StringVar(&options.LogStacktraceLevel, "log-stacktrace-level", "panic", "Stacktrace log level").
		Choices("info", "error", "panic").
		Value()

	if err := tf.Parse(args); err != nil {
		return Options{}, err
	}

	options.MetricsAddr = (*metricsBindAddress).String()
	options.ProbeAddr = (*healthProbeaddress).String()

	return options, nil
}
