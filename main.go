/*
Copyright 2021.

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

package main

import (
	"flag"
	"os"

	c "github.com/containersol/prescale-operator/internal"
	"github.com/containersol/prescale-operator/internal/validations"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	dc "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dc.AddToScheme(scheme))
	utilruntime.Must(scalingv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "14b492ce.prescale.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterScalingStateDefinitionReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ClusterScalingStateDefinition"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clusterscalingstatedefinition-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterScalingStateDefinition")
		os.Exit(1)
	}
	if err = (&controllers.ClusterScalingStateReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ClusterScalingState"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("clusterscalingstate-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterScalingState")
		os.Exit(1)
	}
	if err = (&controllers.ScalingStateReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ScalingState"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("scalingstate-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScalingState")
		os.Exit(1)
	}
	if err = (&controllers.DeploymentWatcher{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("DeploymentWatcher"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("deployment-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DeploymentWatcher")
		os.Exit(1)
	}

	//We need to identify if the operator is running on an Openshift cluster. If yes, we activate the deploymentconfig watcher
	c.OpenshiftCluster, err = validations.ClusterCheck()
	if err != nil {
		setupLog.Error(err, "unable to identify cluster")
	}
	if c.OpenshiftCluster {
		if err = (&controllers.DeploymentConfigWatcher{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("DeploymentConfigWatcher"),
			Scheme:   mgr.GetScheme(),
			Recorder: mgr.GetEventRecorderFor("deploymentconfig-controller"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "DeploymentConfigWatcher")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	z := cron.New()
	z.AddFunc("0 30 * * * *", func() { fmt.Println("Every hour on the half hour") })
	z.Start()
	// s := gocron.NewScheduler(time.UTC)
	// s.Every(1).Minute().Do(r.RectifyDeploymentsInFailureState, mgr.GetClient())
	//gocron.Start()
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
