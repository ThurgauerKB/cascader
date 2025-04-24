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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"

	"github.com/thurgauerkb/cascader/internal/config"
	"github.com/thurgauerkb/cascader/internal/controller"
	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/logging"
	"github.com/thurgauerkb/cascader/internal/utils"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"k8s.io/klog/v2"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Run is the main function of the application.
func Run(ctx context.Context, version string, args []string, out io.Writer) error {
	// Parse and validate command-line arguments
	cfg, err := config.ParseArgs(args, out, version)
	if err != nil {
		var helpErr *config.HelpError
		if errors.As(err, &helpErr) {
			fmt.Fprintln(out, helpErr.Error()) // nolint:errcheck
			return nil
		}
		return fmt.Errorf("error parsing arguments: %w", err)
	}

	// Configure logging
	logger, err := logging.SetupLogger(cfg, out)
	if err != nil {
		return fmt.Errorf("error setting up logger: %w", err)
	}
	log.SetLogger(logger)
	klog.SetLogger(logger) // Redirect klog to use zap
	setupLog := ctrl.Log.WithName("setup")

	setupLog.Info("initializing cascader", "version", version)

	// Configure HTTP/2 settings
	tlsOpts := []func(*tls.Config){}
	if !cfg.EnableHTTP2 {
		setupLog.Info("disabling HTTP/2 for compatibility")
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		})
	}

	// Set up webhook server
	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// Configure metrics server
	metricsServerOptions := metricsserver.Options{BindAddress: "0"} // disable listener by default
	if cfg.EnableMetrics {
		metricsServerOptions = metricsserver.Options{
			BindAddress:   cfg.MetricsAddr,
			SecureServing: cfg.SecureMetrics,
			TLSOpts:       tlsOpts,
		}
		if cfg.SecureMetrics {
			metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
		}
	}

	// Create Cache Options
	cacheOpts := utils.ToCacheOptions(cfg.WatchNamespaces)

	// Create and initialize the manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		Logger:                 logger,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.LeaderElection,
		LeaderElectionID:       "fc1fdccd.cascader.tkb.ch",
		Cache:                  cacheOpts,
	})
	if err != nil {
		return fmt.Errorf("unable to create manager: %w", err)
	}

	// Log watching namespaces
	if len(cfg.WatchNamespaces) == 0 {
		setupLog.Info("namespace scope", "mode", "cluster-wide")
	} else {
		setupLog.Info("namespace scope", "mode", "namespaced", "namespaces", cfg.WatchNamespaces)
	}

	// Validate annotation uniqueness
	configuredAnnotations := map[string]string{
		"DaemonSet":           cfg.DaemonSetAnnotation,
		"Deployment":          cfg.DeploymentAnnotation,
		"StatefulSet":         cfg.StatefulSetAnnotation,
		"LastObservedRestart": cfg.LastObservedRestartAnnotation,
		"RequeueAfter":        cfg.RequeueAfterAnnotation,
	}
	if err := utils.UniqueAnnotations(configuredAnnotations); err != nil {
		return fmt.Errorf("annotation values must be unique: %w", err)
	}

	// Log configured annotations
	setupLog.Info("configured annotations", "values", utils.FormatAnnotations(configuredAnnotations))

	// Define resource annotations with their kinds
	annotationKindMap := kinds.AnnotationKindMap{
		cfg.DaemonSetAnnotation:   kinds.DaemonSetKind,
		cfg.DeploymentAnnotation:  kinds.DeploymentKind,
		cfg.StatefulSetAnnotation: kinds.StatefulSetKind,
	}

	// Setup Deployment controller
	if err := (&controller.DeploymentReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("deployment-controller"),
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: cfg.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:           cfg.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Deployment controller: %w", err)
	}

	// Setup StatefulSet controller
	if err := (&controller.StatefulSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("statefulset-controller"),
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: cfg.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:           cfg.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create StatefulSet controller: %w", err)
	}

	// Setup DaemonSet controller
	if err := (&controller.DaemonSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("daemonset-controller"),
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: cfg.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:           cfg.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create DaemonSet controller: %w", err)
	}

	// Register health and readiness checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("failed to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("failed to set up ready check: %w", err)
	}

	// Start the manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("manager encountered an error while running: %w", err)
	}

	return nil
}
