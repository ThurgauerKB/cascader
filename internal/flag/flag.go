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
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
)

const (
	daemonSetAnnotation           string = "cascader.tkb.ch/daemonset"
	deploymentAnnotation          string = "cascader.tkb.ch/deployment"
	statefulSetAnnotation         string = "cascader.tkb.ch/statefulset"
	lastObservedRestartAnnotation string = "cascader.tkb.ch/last-observed-restart"
	requeueAfterAnnotation        string = "cascader.tkb.ch/requeue-after"
	envPrefix                     string = "CASCADER"
)

// HelpRequested represents a special error type to indicate that help was requested.
type HelpRequested struct {
	Message string
}

// Error returns the error message.
func (e *HelpRequested) Error() string { return e.Message }

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

// Validate checks for invalid or unreachable options like malformed TCP addresses.
func (o Options) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", o.MetricsAddr); err != nil {
		return fmt.Errorf("invalid metrics listen address: %w", err)
	}
	if _, err := net.ResolveTCPAddr("tcp", o.ProbeAddr); err != nil {
		return fmt.Errorf("invalid probe listen address: %w", err)
	}
	return nil
}

// registerFlags binds all application flags to the given FlagSet.
func registerFlags(fs *flag.FlagSet) {
	fs.String("deployment-annotation", deploymentAnnotation, "Annotation key for monitored Deployments")
	fs.String("statefulset-annotation", statefulSetAnnotation, "Annotation key for monitored StatefulSets")
	fs.String("daemonset-annotation", daemonSetAnnotation, "Annotation key for monitored DaemonSets")
	fs.String("last-observed-restart-annotation", lastObservedRestartAnnotation, "Annotation key for last observed restart")
	fs.String("requeue-after-annotation", requeueAfterAnnotation, "Annotation key for requeue interval override")
	fs.Duration("requeue-after-default", 5*time.Second, "Default requeue interval")

	fs.StringSlice("watch-namespace", nil, "Namespaces to watch (can be repeated or comma-separated)")

	fs.Bool("metrics-enabled", true, "Enable or disable the metrics endpoint")
	fs.String("metrics-bind-address", ":8443", "Metrics server address")
	fs.Bool("metrics-secure", true, "Serve metrics over HTTPS")
	fs.String("health-probe-bind-address", ":8081", "Health and readiness probe address")
	fs.Bool("enable-http2", false, "Enable HTTP/2 for servers")
	fs.Bool("leader-elect", true, "Enable leader election")

	fs.String("log-encoder", "json", "Log format (json, console)")
	fs.String("log-stacktrace-level", "panic", "Stacktrace log level (info, error, panic)")
	fs.Bool("log-devel", false, "Enable development mode logging")
}

// ParseArgs parses CLI flags into Options and handles --help/--version output.
func ParseArgs(args []string, w io.Writer, version string) (Options, error) {
	fs := flag.NewFlagSet("Cascader", flag.ContinueOnError)
	fs.SortFlags = false
	fs.SetOutput(w)

	registerFlags(fs)

	var showVersion, showHelp bool
	fs.BoolVar(&showVersion, "version", false, "Show version and exit")
	fs.BoolVarP(&showHelp, "help", "h", false, "Show help and exit")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [flags]\n\nFlags:\n", strings.ToLower(fs.Name())) // nolint:errcheck
		decorateUsageWithEnv(fs, envPrefix)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	if showHelp {
		var buf bytes.Buffer
		fs.SetOutput(&buf)
		fs.Usage()
		return Options{}, &HelpRequested{Message: buf.String()}
	}
	if showVersion {
		return Options{}, &HelpRequested{Message: fmt.Sprintf("%s version %s", fs.Name(), version)}
	}

	return buildOptions(fs)
}

// buildOptions resolves all values from flags, env, or defaults.
func buildOptions(fs *flag.FlagSet) (opts Options, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to parse flags: %v", r)
			opts = Options{}
		}
	}()

	return Options{
		WatchNamespaces:               must(fs.GetStringSlice("watch-namespace")),
		MetricsAddr:                   must(fs.GetString("metrics-bind-address")),
		SecureMetrics:                 must(fs.GetBool("metrics-secure")),
		EnableMetrics:                 must(fs.GetBool("metrics-enabled")),
		LeaderElection:                must(fs.GetBool("leader-elect")),
		ProbeAddr:                     must(fs.GetString("health-probe-bind-address")),
		EnableHTTP2:                   must(fs.GetBool("enable-http2")),
		LogEncoder:                    must(fs.GetString("log-encoder")),
		LogStacktraceLevel:            must(fs.GetString("log-stacktrace-level")),
		LogDev:                        must(fs.GetBool("log-devel")),
		DeploymentAnnotation:          must(fs.GetString("deployment-annotation")),
		StatefulSetAnnotation:         must(fs.GetString("statefulset-annotation")),
		DaemonSetAnnotation:           must(fs.GetString("daemonset-annotation")),
		LastObservedRestartAnnotation: must(fs.GetString("last-observed-restart-annotation")),
		RequeueAfterAnnotation:        must(fs.GetString("requeue-after-annotation")),
		RequeueAfterDefault:           must(fs.GetDuration("requeue-after-default")),
	}, nil
}
