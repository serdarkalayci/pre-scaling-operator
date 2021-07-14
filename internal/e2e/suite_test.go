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

package controllers

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	constants "github.com/containersol/prescale-operator/internal"
	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	dc "github.com/openshift/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	scalingv1alpha1 "github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/controllers"
	"github.com/containersol/prescale-operator/internal/validations"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var k8sManager ctrl.Manager

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var useCluster bool = true

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "..", "config", "crd", "bases")},
		UseExistingCluster: &useCluster,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = dc.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = scalingv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = scalingv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = scalingv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//install CRDs
	options := envtest.CRDInstallOptions{
		Paths: testEnv.CRDDirectoryPaths,
		CRDs:  testEnv.CRDs,
	}
	options.MaxTime = time.Minute * 20

	_, err = envtest.InstallCRDs(cfg, options)
	Expect(err).NotTo(HaveOccurred())

	// register controllers
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ClusterScalingStateDefinitionReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ClusterScalingStateDefinition"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("clusterscalingstatedefinition-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ClusterScalingStateReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ClusterScalingState"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("clusterscalingstate-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ScalingStateReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ScalingState"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("scalingstate-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.DeploymentWatcher{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("DeploymentWatcher"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("deployment-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	//OpenshiftCluster, _ := validations.ClusterCheck()
	constants.OpenshiftCluster, err = validations.ClusterCheck()
	if constants.OpenshiftCluster {
		err = (&controllers.DeploymentConfigWatcher{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("DeploymentConfigWatcher"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("deploymentconfig-controller"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	}
	constants.StartTime = time.Now()

	godotenv.Load("../../.env")
	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	// Give some time to startup
	time.Sleep(time.Second * 15)

}, 60)

func CreateClusterScalingState(casenumber int, state string, class string) v1alpha1.ClusterScalingState {

	spec := v1alpha1.ClusterScalingStateSpec{}
	if class != "" {
		spec.State = state
		spec.ScalingClass = class
	} else {
		spec.State = state
	}
	if class == "" {
		class = "default"
	}
	name := "clusterscalingstate-e2e-case-" + strconv.Itoa(casenumber) + "-" + state + "-" + class

	scalingState := &v1alpha1.ClusterScalingState{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterScalingState",
			APIVersion: "scaling.prescale.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
		Config: v1alpha1.ClusterScalingStateConfiguration{
			DryRun: false,
		},
	}

	return *scalingState
}

func CreateScalingState(state, namespace string) v1alpha1.ScalingState {

	scalingState := &v1alpha1.ScalingState{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScalingState",
			APIVersion: "scaling.prescale.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scalingstate-sample",
			Namespace: namespace,
		},
		Spec: v1alpha1.ScalingStateSpec{
			State: state,
		},
		Config: v1alpha1.ScalingStateConfiguration{
			DryRun: false,
		},
	}

	return *scalingState
}

func CreateClusterScalingStateDefinition() v1alpha1.ClusterScalingStateDefinition {

	states := []v1alpha1.States{
		{
			Name:        "peak",
			Description: "Business critical",
			Priority:    1,
		},
		{
			Name:        "marketing",
			Description: "Marketing run",
			Priority:    5,
		},
		{
			Name:        "bau",
			Description: "Business as usual",
			Priority:    10,
		},
	}

	scalingState := &v1alpha1.ClusterScalingStateDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterScalingStateDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "global-state-definition",
		},
		Spec: states,
		Config: v1alpha1.ClusterScalingStateDefinitionConfiguration{
			DryRun: false,
		},
	}

	return *scalingState
}

func returnModifiedStates(state1name string, state2name string, state1prio int32, state2prio int32) []v1alpha1.States {
	return []v1alpha1.States{
		{
			Name:        state1name,
			Description: "Test State 1",
			Priority:    state1prio,
		},
		{
			Name:        state2name,
			Description: "Test State 2",
			Priority:    state2prio,
		},
	}
}

func getModifiedClusterScalingStateDefinition(cssd v1alpha1.ClusterScalingStateDefinition, changeName bool, changePrio bool) v1alpha1.ClusterScalingStateDefinition {

	var statename1 string
	var statename2 string
	var priostate1 int32
	var priostate2 int32

	if changeName {
		statename1 = "mode1"
		statename2 = "mode2"
	} else {
		statename1 = "bau"
		statename2 = "peak"
	}

	if changePrio {
		priostate1 = 1
		priostate2 = 10
	} else {
		priostate1 = 10
		priostate2 = 1
	}

	states := returnModifiedStates(statename1, statename2, priostate1, priostate2)
	cssd.Spec = states

	return cssd
}
