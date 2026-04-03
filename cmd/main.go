package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	sentryv1alpha1 "github.com/agjmills/sentry-operator/api/v1alpha1"
	"github.com/agjmills/sentry-operator/internal/controller"
	"github.com/agjmills/sentry-operator/internal/sentry"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(sentryv1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr         string
		probeAddr           string
		leaderElect         bool
		sentryURL           string
		sentryTokenEnv      string
		defaultOrganization string
		defaultTeam         string
		defaultPlatform     string
		defaultRetainOnDel  bool
		requeueInterval     time.Duration
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&leaderElect, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&sentryURL, "sentry-url", "https://sentry.io", "Base URL of the Sentry instance.")
	flag.StringVar(&sentryTokenEnv, "sentry-token-env", "SENTRY_TOKEN", "Environment variable name containing the Sentry auth token.")
	flag.StringVar(&defaultOrganization, "default-organization", "", "Default Sentry organization slug (used when spec.organization is unset).")
	flag.StringVar(&defaultTeam, "default-team", "", "Default Sentry team slug (used when spec.team is unset).")
	flag.StringVar(&defaultPlatform, "default-platform", "", "Default Sentry platform (used when spec.platform is unset).")
	flag.BoolVar(&defaultRetainOnDel, "default-retain-on-delete", true, "Default retainOnDelete value (true = keep Sentry project when CRD is deleted).")
	flag.DurationVar(&requeueInterval, "requeue-interval", 24*time.Hour, "How often to re-validate existing projects against the Sentry API.")

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	token := os.Getenv(sentryTokenEnv)
	if token == "" {
		setupLog.Error(nil, "Sentry auth token not set", "envVar", sentryTokenEnv)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         leaderElect,
		LeaderElectionID:       "sentry-operator.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	sentryClient := sentry.NewClient(sentryURL, token)

	if err = (&controller.SentryProjectReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		SentryClient: sentryClient,
		Config: controller.Config{
			DefaultOrganization:   defaultOrganization,
			DefaultTeam:           defaultTeam,
			DefaultPlatform:       defaultPlatform,
			DefaultRetainOnDelete: defaultRetainOnDel,
			SentryURL:             sentryURL,
			RequeueInterval:       requeueInterval,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SentryProject")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "sentry-url", sentryURL, "default-org", defaultOrganization)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
