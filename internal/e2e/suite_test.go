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
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var k8sManager ctrl.Manager

var css *v1alpha1.ClusterScalingState = CreateClusterScalingState()
var cssd *v1alpha1.ClusterScalingStateDefinition = CreateClusterScalingStateDefinition()

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
	_, err = envtest.InstallCRDs(cfg, options)
	Expect(err).NotTo(HaveOccurred())

	// register controllers
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ClusterScalingStateDefinitionReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterScalingStateDefinition"),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ClusterScalingStateReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ClusterScalingState"),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.ScalingStateReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ScalingState"),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.DeploymentWatcher{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("DeploymentWatcher"),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	Expect(k8sClient.Create(context.Background(), css)).Should(Succeed())

	Expect(k8sClient.Create(context.Background(), cssd)).Should(Succeed())

	// Give some time to startup
	time.Sleep(time.Second * 15)

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	Expect(k8sClient.Delete(context.Background(), cssd)).Should(Succeed())
	Expect(k8sClient.Delete(context.Background(), css)).Should(Succeed())

	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

})

func CreateClusterScalingState() *v1alpha1.ClusterScalingState {

	scalingState := &v1alpha1.ClusterScalingState{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterScalingState",
			APIVersion: "scaling.prescale.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterscalingstate-sample",
		},
		Spec: v1alpha1.ClusterScalingStateSpec{
			State: "bau",
		},
	}

	return scalingState
}

func CreateClusterScalingStateDefinition() *v1alpha1.ClusterScalingStateDefinition {

	states := []v1alpha1.States{
		{
			Name:        "marketing",
			Description: "marketing run",
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
	}

	return scalingState
}

// apiVersion: scaling.prescale.com/v1alpha1
// kind: ClusterScalingStateDefinition
// metadata:
//   name: global-state-definition
// spec:
// - name: peak
//   description: "Maximum scaling settings"
//   priority: 1
// - name: marketing-run
//   description: "Higher expected load after a marketing run. Possibly an email blast or twitter share."
//   priority: 5
// - name: bau
//   description: "Business as usual"
//   priority: 10
