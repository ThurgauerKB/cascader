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
	"fmt"
	"io"
	"net"
	"time"

	flag "github.com/spf13/pflag"
)

const (
	daemonSetAnnotation           string = "cascader.tkb.ch/daemonset"
	deploymentAnnotation          string = "cascader.tkb.ch/deployment"
	statefulSetAnnotation         string = "cascader.tkb.ch/statefulset"
	lastObservedRestartAnnotation string = "cascader.tkb.ch/last-observed-restart"
	requeueAfterAnnotation        string = "cascader.tkb.ch/requeue-after"
)

// HelpRequested represents a special error type to indicate that help was requested.
type HelpRequested struct {
	Message string
}

// Error returns the error message.
func (e *HelpRequested) Error() string { return e.Message }

// Config holds all configuration options for the application.
type Config struct {
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

// Validate validates the configuration.
func (c Config) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", c.MetricsAddr); err != nil {
		return fmt.Errorf("invalid metrics listen address: %w", err)
	}
	if _, err := net.ResolveTCPAddr("tcp", c.ProbeAddr); err != nil {
		return fmt.Errorf("invalid probe listen address: %w", err)
	}
	return nil
}

// ParseArgs parses CLI arguments into a Config struct.
func ParseArgs(args []string, out io.Writer, version string) (Config, error) {
	var cfg Config
	fs := flag.NewFlagSet("Cascader", flag.ContinueOnError)
	fs.SortFlags = false // Preserve order in help output.
	fs.SetOutput(out)

	// Define flags
	fs.StringVar(&cfg.DeploymentAnnotation, "deployment-annotation", deploymentAnnotation, "Annotation key for monitored Deployments")
	fs.StringVar(&cfg.StatefulSetAnnotation, "statefulset-annotation", statefulSetAnnotation, "Annotation key for monitored StatefulSets")
	fs.StringVar(&cfg.DaemonSetAnnotation, "daemonset-annotation", daemonSetAnnotation, "Annotation key for monitored DaemonSets")
	fs.StringVar(&cfg.LastObservedRestartAnnotation, "last-observed-restart-annotation", lastObservedRestartAnnotation, "Annotation key for last observed restart")
	fs.StringVar(&cfg.RequeueAfterAnnotation, "requeue-after-annotation", requeueAfterAnnotation, "Annotation key for requeue interval override")
	fs.DurationVar(&cfg.RequeueAfterDefault, "requeue-after-default", 5*time.Second, "Default requeue interval")

	fs.StringSliceVar(&cfg.WatchNamespaces, "watch-namespace", nil, "Namespaces to watch (can be repeated or comma-separated). Watches all if unset.")

	fs.BoolVar(&cfg.EnableMetrics, "metrics-enabled", true, "Enable or disable the metrics endpoint")
	fs.StringVar(&cfg.MetricsAddr, "metrics-bind-address", ":8443", "Metrics server address (e.g., :8080 for HTTP, :8443 for HTTPS)")
	fs.BoolVar(&cfg.SecureMetrics, "metrics-secure", true, "Serve metrics over HTTPS")

	fs.BoolVar(&cfg.EnableHTTP2, "enable-http2", false, "Enable HTTP/2 for servers")

	fs.StringVar(&cfg.ProbeAddr, "health-probe-bind-address", ":8081", "Health and readiness probe address")

	fs.BoolVar(&cfg.LeaderElection, "leader-elect", true, "Enable leader election")

	fs.StringVar(&cfg.LogEncoder, "log-encoder", "json", "Log format (json, console)")
	fs.StringVar(&cfg.LogStacktraceLevel, "log-stacktrace-level", "panic", "Stacktrace log level (info, error, panic)")
	fs.BoolVar(&cfg.LogDev, "log-devel", false, "Enable development mode logging")

	var showVersion, showHelp bool
	fs.BoolVar(&showVersion, "version", false, "Show version and exit")
	fs.BoolVarP(&showHelp, "help", "h", false, "Show help and exit")

	// Custom usage message
	fs.Usage = func() {
		fs.Output().Write([]byte("Usage:\n")) // nolint:errcheck
		fs.PrintDefaults()
	}

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	// Handle --help and --version
	if showHelp {
		return Config{}, &HelpRequested{Message: captureUsage(fs)}
	}
	if showVersion {
		return Config{}, &HelpRequested{Message: fmt.Sprintf("%s version %s", fs.Name(), version)}
	}

	return cfg, nil
}

// captureUsage captures help output into a string.
func captureUsage(fs *flag.FlagSet) string {
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.Usage()
	return buf.String()
}

// IsHelpRequested checks if the error is a HelpRequested sentinel and prints it.
func IsHelpRequested(err error, w io.Writer) bool {
	var helpErr *HelpRequested
	if errors.As(err, &helpErr) {
		fmt.Fprint(w, helpErr.Error()) // nolint:errcheck
		return true
	}
	return false
}
