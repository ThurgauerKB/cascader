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
	"fmt"
	"io"

	"github.com/containeroo/tinyflags"

	"github.com/thurgauerkb/cascader/internal/controller"
	"github.com/thurgauerkb/cascader/internal/flag"
	"github.com/thurgauerkb/cascader/internal/kinds"
	"github.com/thurgauerkb/cascader/internal/logging"
	internalmetrics "github.com/thurgauerkb/cascader/internal/metrics"
	"github.com/thurgauerkb/cascader/internal/utils"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Run is the main function of the application.
func Run(ctx context.Context, version string, args []string, w io.Writer) error {
	// Parse and validate command-line arguments
	flags, err := flag.ParseArgs(args, version)

	logger, lErr := logging.InitLogging(flags, w)
	setupLog := logger.WithName("setup")
	setupLog.Info("initializing cascader", "version", version)

	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(w, err.Error()) // nolint:errcheck
			return nil
		}
		setupLog.Error(err, "error setting up logger")
		return err
	}

	// Setup logger immediately so startup errors are correctly logged.
	if lErr != nil {
		setupLog.Error(lErr, "error setting up logger")
		return err
	}

	// Validate annotation uniqueness
	configuredAnnotations := map[string]string{
		"DaemonSet":           flags.DaemonSetAnnotation,
		"Deployment":          flags.DeploymentAnnotation,
		"StatefulSet":         flags.StatefulSetAnnotation,
		"LastObservedRestart": flags.LastObservedRestartAnnotation,
		"RequeueAfter":        flags.RequeueAfterAnnotation,
	}
	if err := utils.UniqueAnnotations(configuredAnnotations); err != nil {
		setupLog.Error(err, "annotation values must be unique")
		return err
	}
	setupLog.Info("configured annotations", "values", utils.FormatAnnotations(configuredAnnotations))

	// Configure HTTP/2 settings
	tlsOpts := []func(*tls.Config){}
	if !flags.EnableHTTP2 {
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
	metricsReg := internalmetrics.NewRegistry(crmetrics.Registry)

	metricsServerOptions := metricsserver.Options{
		BindAddress: "0", // disable listener by default
	}
	if flags.EnableMetrics {
		metricsServerOptions = metricsserver.Options{
			BindAddress:   flags.MetricsAddr,
			SecureServing: flags.SecureMetrics,
			TLSOpts:       tlsOpts,
		}
		if flags.SecureMetrics {
			metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
		}
	}

	// Create Cache Options
	cacheOpts := utils.ToCacheOptions(flags.WatchNamespaces)

	// Create and initialize the manager
	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get Kubernetes REST config")
		return err
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		Logger:                 logger,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: flags.ProbeAddr,
		LeaderElection:         flags.LeaderElection,
		LeaderElectionID:       "fc1fdccd.cascader.tkb.ch",
		Cache:                  cacheOpts,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		return err
	}

	// Log watching namespaces
	if len(flags.WatchNamespaces) == 0 {
		setupLog.Info("namespace scope", "mode", "cluster-wide")
	} else {
		setupLog.Info("namespace scope", "mode", "namespaced", "namespaces", flags.WatchNamespaces)
	}

	// Define resource annotations with their kinds
	annotationKindMap := kinds.AnnotationKindMap{
		flags.DaemonSetAnnotation:   kinds.DaemonSetKind,
		flags.DeploymentAnnotation:  kinds.DeploymentKind,
		flags.StatefulSetAnnotation: kinds.StatefulSetKind,
	}

	// Setup Deployment controller
	if err := (&controller.DeploymentReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("deployment-controller"),
			Metrics:                       metricsReg,
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: flags.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        flags.RequeueAfterAnnotation,
			RequeueAfterDefault:           flags.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Deployment controller")
		return err
	}

	// Setup StatefulSet controller
	if err := (&controller.StatefulSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("statefulset-controller"),
			Metrics:                       metricsReg,
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: flags.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        flags.RequeueAfterAnnotation,
			RequeueAfterDefault:           flags.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create StatefulSet controller")
		return err
	}

	// Setup DaemonSet controller
	if err := (&controller.DaemonSetReconciler{
		BaseReconciler: controller.BaseReconciler{
			Logger:                        &logger,
			KubeClient:                    mgr.GetClient(),
			Recorder:                      mgr.GetEventRecorderFor("daemonset-controller"),
			Metrics:                       metricsReg,
			AnnotationKindMap:             annotationKindMap,
			LastObservedRestartAnnotation: flags.LastObservedRestartAnnotation,
			RequeueAfterAnnotation:        flags.RequeueAfterAnnotation,
			RequeueAfterDefault:           flags.RequeueAfterDefault,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create DaemonSet controller")
		return err
	}

	// Register health and readiness checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "failed to set up ready check")
		return err
	}

	// Start the manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "manager encountered an error while running")
		return err
	}

	return nil
}
