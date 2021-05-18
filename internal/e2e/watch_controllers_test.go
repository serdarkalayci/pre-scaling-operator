package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	"github.com/containersol/prescale-operator/internal/validations"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	ocv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("e2e Test for the main operator functionalities", func() {

	const timeout = time.Second * 70
	const interval = time.Millisecond * 200

	var casenumber = 1
	var namespace corev1.Namespace
	OpenshiftCluster, _ := validations.ClusterCheck()
	var deployment v1.Deployment
	var deploymentconfig ocv1.DeploymentConfig
	var css v1alpha1.ClusterScalingState
	var cssd v1alpha1.ClusterScalingStateDefinition

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "e2e-tests-workloadwatchers" + strconv.Itoa(casenumber),
	}

	BeforeEach(func() {

		css = CreateClusterScalingState("bau")
		cssd = CreateClusterScalingStateDefinition()

		Expect(k8sClient.Create(context.Background(), &cssd)).Should(Succeed())

		Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

		namespace = createNS(key)

		Expect(k8sClient.Create(context.Background(), &namespace)).Should(Succeed())

	})

	AfterEach(func() {
		// Wait until all potential wait-loops in the step scaler are finished.
		time.Sleep(time.Second * 11)
		// Tear down the deployment or deploymentconfig
		if OpenshiftCluster {
			Expect(k8sClient.Delete(context.Background(), &deploymentconfig)).Should(Succeed())
		} else {
			Expect(k8sClient.Delete(context.Background(), &deployment)).Should(Succeed())
		}

		Expect(k8sClient.Delete(context.Background(), &namespace)).Should(Succeed())

		Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &cssd)).Should(Succeed())

		casenumber = casenumber + 1

		key = types.NamespacedName{
			Name:      "test",
			Namespace: "e2e-tests-workloadwatchers" + strconv.Itoa(casenumber),
		}

		time.Sleep(time.Second * 1)
	})

	Context("Deployment in place and modification test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(annotationchange bool, replicachange bool, optinOld bool, optinNew bool, expectedReplicas int) {
				key.Name = "case" + strconv.Itoa(casenumber)
				fetchedDeployment := v1.Deployment{}
				fetchedDeploymentConfig := ocv1.DeploymentConfig{}

				if OpenshiftCluster {

					deploymentconfig = *createDeploymentConfig(key, optinOld, casenumber)
					Expect(k8sClient.Create(context.Background(), &deploymentconfig)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig
					}, timeout, interval).Should(Not(BeNil()))

					if annotationchange {
						fetchedDeploymentConfig = changeAnnotationDC(fetchedDeploymentConfig)
					}

					if replicachange {
						fetchedDeploymentConfig = changeReplicasDC(fetchedDeploymentConfig)
					}

					fetchedDeploymentConfig = changeOptInDC(fetchedDeploymentConfig, optinNew)

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeploymentConfig)).Should(Succeed())

					time.Sleep(time.Second * 5)

					var replicas32 int32 = int32(expectedReplicas)

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig.Status.AvailableReplicas
					}, timeout, interval).Should(Equal(replicas32))

				} else {

					deployment = createDeployment(key, optinOld, casenumber)
					Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment
					}, timeout, interval).Should(Not(BeNil()))

					if annotationchange {
						fetchedDeployment = changeAnnotation(fetchedDeployment)
					}

					if replicachange {
						fetchedDeployment = changeReplicas(fetchedDeployment)
					}

					fetchedDeployment = changeOptIn(fetchedDeployment, optinNew)

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeployment)).Should(Succeed())

					time.Sleep(time.Second * 5)

					var replicas32 int32 = int32(expectedReplicas)
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return *fetchedDeployment.Spec.Replicas
					}, timeout, interval).Should(Equal(replicas32))

				}

			},

				// Default Replica Count from test if oldoptin = true: 3
				// Default Replica Count from test if oldoptin = false: 1
				// Default fallback annotation count: 2
				// bau Annoation (if changed) will change to :4
				// Replica change (that needs to be rectified): 5
				// Structure:  ("Description of the case" , annotationchange, replicachange, oldoptin, newoptin, expectedReplicas)
				table.Entry("CASE 1  | Should scale to 4 | Annotation has changed. ", true, true, true, true, 4),
				table.Entry("CASE 2  | Should go to 5    | Deployment has been disabled and replica has been modified", true, true, true, false, 5),
				table.Entry("CASE 3  | Should scale to 4 | Deployment opted in and annotation changed", true, true, false, true, 4),
				table.Entry("CASE 4  | Should be at 5    | Deployment opted out. Will go to modified replica count (5)", true, true, false, false, 5),
				table.Entry("CASE 5  | Should scale to 4 | Annotation has been modified.", true, false, true, true, 4),
				table.Entry("CASE 6  | Should stay at 3  | Deployment has been disabled", true, false, true, false, 3),
				table.Entry("CASE 7  | Should scale to 4 | Deployment opted in and annotation changed.", true, false, false, true, 4),
				table.Entry("CASE 8  | Should be at 1    | Deployment opted out", true, false, false, false, 1),
				table.Entry("CASE 9  | Should scale to 3 | Replica count has been modified. Rectify back to 'bau'", false, true, true, true, 3),
				table.Entry("CASE 10 | Should go to 5    | Deployment has been disabled and replica has been modified", false, true, true, false, 5),
				table.Entry("CASE 11 | Should scale to 3 | Deployment opted in. Scale to 'bau'", false, true, false, true, 3),
				table.Entry("CASE 12 | Should be at 5    | Deployment opted out. Will go to modified replica count (5)", false, true, false, false, 5),
				table.Entry("CASE 13 | Stay at Bau  	 | Something else on the deployment changed. Don't do anything", false, false, true, true, 3),
				table.Entry("CASE 14 | Should stay at 3  | Deployment has been disabled", false, false, true, false, 3),
				table.Entry("CASE 15 | Should scale to 3 | Deployment opted in. Scale to 'bau'", false, false, false, true, 3),
				table.Entry("CASE 16 | Should be at 1    | Nothing happend", false, false, false, false, 1),
			)
		})
	})

})

func changeAnnotation(deployment v1.Deployment) v1.Deployment {

	deployment.Annotations = map[string]string{
		"scaler/state-bau-replicas":     "4", // That reflects the annotation change and will change replica # to 4
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	return deployment
}

func changeAnnotationDC(deploymentconfig ocv1.DeploymentConfig) ocv1.DeploymentConfig {

	deploymentconfig.Annotations = map[string]string{
		"scaler/state-bau-replicas":     "4", // That reflects the annotation change and will change replica # to 4
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	return deploymentconfig
}

func changeOptIn(deployment v1.Deployment, optIn bool) v1.Deployment {

	deployment.Labels = map[string]string{
		"app":           "random-generator-1",
		"scaler/opt-in": strconv.FormatBool(optIn),
	}
	return deployment
}

func changeOptInDC(deploymentconfig ocv1.DeploymentConfig, optIn bool) ocv1.DeploymentConfig {

	deploymentconfig.Labels = map[string]string{
		"app":           "random-generator-1",
		"scaler/opt-in": strconv.FormatBool(optIn),
	}
	return deploymentconfig
}

// This covers the case when someone external simply edits the replica count. Depending on the opt-in the operator needs to rectify this.
func changeReplicas(deployment v1.Deployment) v1.Deployment {

	var replicas int32 = 5

	deployment.Spec.Replicas = &replicas

	return deployment
}

func changeReplicasDC(deploymentconfig ocv1.DeploymentConfig) ocv1.DeploymentConfig {

	var replicas int32 = 5

	deploymentconfig.Spec.Replicas = replicas

	return deploymentconfig
}

func createDeployment(deploymentInfo types.NamespacedName, optInOld bool, casenumber int) v1.Deployment {
	var replicaCount int32
	if optInOld {
		replicaCount = 3 // Deployment should start with "bau" in the test. Therefore 3
	} else {
		replicaCount = 1
	}

	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"app":           "random-generator-1",
				"scaler/opt-in": strconv.FormatBool(optInOld),
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     "3",
				"scaler/state-default-replicas": "2",
				"scaler/state-peak-replicas":    "7",
			},
		},

		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "random-generator-1",
				},
			},
			Replicas: &replicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "random-generator-1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "chriscmsoft/random-generator:latest",
						Name:  deploymentInfo.Name},
					},
				},
			},
		},
	}
	return *dep
}

func createDeploymentConfig(deploymentInfo types.NamespacedName, optInOld bool, casenumber int) *ocv1.DeploymentConfig {
	var replicaCount int32
	if optInOld {
		replicaCount = 3 // Deployment should start with "bau" in the test. Therefore 3
	} else {
		replicaCount = 1
	}

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"app":           "random-generator-1",
				"scaler/opt-in": strconv.FormatBool(optInOld),
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     "3",
				"scaler/state-default-replicas": "2",
				"scaler/state-peak-replicas":    "7",
			},
		},

		Spec: ocv1.DeploymentConfigSpec{
			Replicas: replicaCount,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "random-generator-1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "chriscmsoft/random-generator:latest",
						Name:  deploymentInfo.Name},
					},
				},
			},
		},
	}
	return deploymentConfig
}

func createNS(deploymentInfo types.NamespacedName) corev1.Namespace {

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentInfo.Namespace,
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	return *ns
}
