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
	"net/http"
	"net/http/pprof"
	"os"

	jitsiv1beta1 "github.com/onmetal/meeting-operator/apis/jitsi/v1beta1"
	etherpadcontroller "github.com/onmetal/meeting-operator/controllers/etherpad"
	jitsi "github.com/onmetal/meeting-operator/controllers/jitsi"
	jascontroller "github.com/onmetal/meeting-operator/controllers/jitsiautoscaler"
	boardcontroller "github.com/onmetal/meeting-operator/controllers/whiteboard"

	ethv1alpha2 "github.com/onmetal/meeting-operator/apis/etherpad/v1alpha2"
	jasv1alpha1 "github.com/onmetal/meeting-operator/apis/jitsiautoscaler/v1alpha1"

	boardv1alpha1 "github.com/onmetal/meeting-operator/apis/whiteboard/v1alpha2"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	addToScheme()
	var metricsAddr string
	var enableLeaderElection, profiling bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&profiling, "profiling", false, "Enabling this will activate profiling that will be listen on :8080")
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
		LeaderElectionID:       "4642dc8b.meeting.ko",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	createReconciles(mgr)
	addHandlers(mgr, profiling)

	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addToScheme() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(jitsiv1beta1.AddToScheme(scheme))
	utilruntime.Must(ethv1alpha2.AddToScheme(scheme))
	utilruntime.Must(boardv1alpha1.AddToScheme(scheme))
	utilruntime.Must(jasv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func createReconciles(mgr ctrl.Manager) {
	var err error
	if err = (&etherpadcontroller.Reconcile{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Etherpad"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Etherpad")
		os.Exit(1)
	}
	if err = (&jitsi.WebReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Web"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Web")
		os.Exit(1)
	}
	if err = (&jitsi.ProsodyReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Prosody"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Prosody")
		os.Exit(1)
	}
	if err = (&jitsi.JicofoReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Jicofo"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jicofo")
		os.Exit(1)
	}
	if err = (&jitsi.JigasiReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Jigasi"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jigasi")
		os.Exit(1)
	}
	if err = (&jitsi.JibriReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Jibri"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Jibri")
		os.Exit(1)
	}
	if err = (&boardcontroller.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("WhiteBoard"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "WhiteBoard")
		os.Exit(1)
	}
	if err = (&jascontroller.Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AutoScaler"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AutoScaler")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder
}

func addHandlers(mgr ctrl.Manager, profiling bool) {
	if healthErr := mgr.AddHealthzCheck("healthz", healthz.Ping); healthErr != nil {
		setupLog.Error(healthErr, "unable to set up health check")
		os.Exit(1)
	}
	if readyErr := mgr.AddReadyzCheck("readyz", healthz.Ping); readyErr != nil {
		setupLog.Error(readyErr, "unable to set up ready check")
		os.Exit(1)
	}
	if profiling {
		err := mgr.AddMetricsExtraHandler("/debug/pprof/", http.HandlerFunc(pprof.Index))
		if err != nil {
			setupLog.Error(err, "unable to attach pprof to webserver")
			os.Exit(1)
		}
		err = mgr.AddMetricsExtraHandler("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		if err != nil {
			setupLog.Error(err, "unable to attach cpu pprof to webserver")
			os.Exit(1)
		}
		setupLog.Info("profiling activated")
	}
}
