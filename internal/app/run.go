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
		if errors.Is(err, &config.HelpError{}) {
			fmt.Fprintln(out, err.Error()) // nolint:errcheck
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

	// Configure HTTP/2 settings
	tlsOpts := []func(*tls.Config){}
	if !cfg.EnableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			setupLog.Info("Disabling HTTP/2 for compatibility")
			c.NextProtos = []string{"http/1.1"}
		})
	}

	// Set up webhook server
	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// Configure metrics server
	var metricsServerOptions metricsserver.Options
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

	// Create and initialize the manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		Logger:                 logger,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.LeaderElection,
		LeaderElectionID:       "fc1fdccd.cascader.tkb.ch",
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	// Validate annotation uniqueness
	configuredAnnotations := map[string]string{
		"Deployment":   cfg.DeploymentAnnotation,
		"StatefulSet":  cfg.StatefulSetAnnotation,
		"DaemonSet":    cfg.DaemonSetAnnotation,
		"RequeueAfter": cfg.RequeueAfterAnnotation,
	}
	if err := utils.UniqueAnnotations(configuredAnnotations); err != nil {
		return fmt.Errorf("annotation values must be unique: %w", err)
	}

	// Log configured annotations
	setupLog.Info("starting cascader", "version", version)
	setupLog.Info("configured annotations", "annotations", utils.FormatAnnotations(configuredAnnotations))

	// Define resource annotations with their kinds
	annotationKindMap := kinds.AnnotationKindMap{
		cfg.DeploymentAnnotation:  kinds.DeploymentKind,
		cfg.StatefulSetAnnotation: kinds.StatefulSetKind,
		cfg.DaemonSetAnnotation:   kinds.DaemonSetKind,
	}

	// Set up DeploymentReconciler
	if err := (&controller.DeploymentReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                 &logger,
			KubeClient:             mgr.GetClient(),
			Recorder:               mgr.GetEventRecorderFor("deployment-controller"),
			AnnotationKindMap:      annotationKindMap,
			RequeueAfterAnnotation: cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:    cfg.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Deployment controller: %w", err)
	}

	// Set up StatefulSetReconciler
	if err := (&controller.StatefulSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                 &logger,
			KubeClient:             mgr.GetClient(),
			Recorder:               mgr.GetEventRecorderFor("statefulset-controller"),
			AnnotationKindMap:      annotationKindMap,
			RequeueAfterAnnotation: cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:    cfg.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create StatefulSet controller: %w", err)
	}

	// Set up DaemonSetReconciler
	if err := (&controller.DaemonSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                 &logger,
			KubeClient:             mgr.GetClient(),
			Recorder:               mgr.GetEventRecorderFor("daemonset-controller"),
			AnnotationKindMap:      annotationKindMap,
			RequeueAfterAnnotation: cfg.RequeueAfterAnnotation,
			RequeueAfterDefault:    cfg.RequeueAfterDefault,
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
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("manager encountered an error while running: %w", err)
	}
	return nil
}
