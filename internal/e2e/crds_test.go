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

var _ = Describe("e2e Test for the crd controllers", func() {

	const timeout = time.Second * 20
	const interval = time.Millisecond * 500

	var casenumber = 1
	OpenshiftCluster, _ := validations.ClusterCheck()
	var deploymentconfig1 ocv1.DeploymentConfig
	var deploymentconfig2 ocv1.DeploymentConfig
	var deployment1 v1.Deployment
	var deployment2 v1.Deployment
	var namespace1 corev1.Namespace
	var namespace2 corev1.Namespace
	var css v1alpha1.ClusterScalingState
	var ss v1alpha1.ScalingState
	var cssd v1alpha1.ClusterScalingStateDefinition

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "e2e-tests-crds" + strconv.Itoa(casenumber),
	}

	BeforeEach(func() {

		cssd = CreateClusterScalingStateDefinition()

		Expect(k8sClient.Create(context.Background(), &cssd)).Should(Succeed())

		namespace1 = CrdCreateNS(key, "1")
		namespace2 = CrdCreateNS(key, "2")

		Expect(k8sClient.Create(context.Background(), &namespace1)).Should(Succeed())
		Expect(k8sClient.Create(context.Background(), &namespace2)).Should(Succeed())

		key = types.NamespacedName{
			Name:      "test",
			Namespace: namespace1.Name,
		}

		if OpenshiftCluster {
			deploymentconfig1 = CreateDeploymentConfigRQ(key, "true", casenumber, "1")
			deploymentconfig2 = CreateDeploymentConfigRQ(key, "false", casenumber, "2")

			Expect(k8sClient.Create(context.Background(), &deploymentconfig1)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), &deploymentconfig2)).Should(Succeed())
		} else {
			deployment1 = CreateDeploymentRQ(key, "true", casenumber, "1")
			deployment2 = CreateDeploymentRQ(key, "false", casenumber, "2")

			Expect(k8sClient.Create(context.Background(), &deployment1)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), &deployment2)).Should(Succeed())
		}

		key = types.NamespacedName{
			Name:      "test",
			Namespace: namespace2.Name,
		}

		if OpenshiftCluster {
			deploymentconfig1 = CreateDeploymentConfigRQ(key, "false", casenumber, "1")
			deploymentconfig2 = CreateDeploymentConfigRQ(key, "true", casenumber, "2")

			Expect(k8sClient.Create(context.Background(), &deploymentconfig1)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), &deploymentconfig2)).Should(Succeed())

		} else {
			deployment1 = CreateDeploymentRQ(key, "false", casenumber, "1")
			deployment2 = CreateDeploymentRQ(key, "true", casenumber, "2")

			Expect(k8sClient.Create(context.Background(), &deployment1)).Should(Succeed())
			Expect(k8sClient.Create(context.Background(), &deployment2)).Should(Succeed())
		}

	})

	AfterEach(func() {

		Expect(k8sClient.Delete(context.Background(), &namespace1)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &namespace2)).Should(Succeed())

		if casenumber == 1 {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
		} else if casenumber == 2 {
			Expect(k8sClient.Delete(context.Background(), &ss)).Should(Succeed())
		} else {
			Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), &ss)).Should(Succeed())
		}

		Expect(k8sClient.Delete(context.Background(), &cssd)).Should(Succeed())

		casenumber = casenumber + 1

		key = types.NamespacedName{
			Name:      "test",
			Namespace: "e2e-tests-crds" + strconv.Itoa(casenumber),
		}

		time.Sleep(time.Second * 1)
	})

	Context("Deployment in place and modification test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(expectedReplicas1 int, expectedReplicas2 int, expectedReplicas3 int, expectedReplicas4 int) {
				key.Name = "case" + strconv.Itoa(casenumber)
				fetchedDeploymentConfig1 := ocv1.DeploymentConfig{}
				fetchedDeploymentConfig2 := ocv1.DeploymentConfig{}
				fetchedDeploymentConfig3 := ocv1.DeploymentConfig{}
				fetchedDeploymentConfig4 := ocv1.DeploymentConfig{}
				fetchedDeployment1 := v1.Deployment{}
				fetchedDeployment2 := v1.Deployment{}
				fetchedDeployment3 := v1.Deployment{}
				fetchedDeployment4 := v1.Deployment{}

				if casenumber == 1 {
					css = CreateClusterScalingState("bau")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())
				} else if casenumber == 2 {
					ss = CreateScalingState("peak", namespace1.Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				} else if casenumber == 3 {
					css = CreateClusterScalingState("bau")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 10)

					ss = CreateScalingState("peak", namespace1.Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				} else {
					css = CreateClusterScalingState("peak")
					Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

					time.Sleep(time.Second * 10)

					ss = CreateScalingState("bau", namespace2.Name)
					Expect(k8sClient.Create(context.Background(), &ss)).Should(Succeed())
				}

				time.Sleep(time.Second * 10)

				if OpenshiftCluster {

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "1"

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig1)
						return fetchedDeploymentConfig1
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig2)
						return fetchedDeploymentConfig2
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig3)
						return fetchedDeploymentConfig3
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig4)
						return fetchedDeploymentConfig4
					}, timeout, interval).Should(Not(BeNil()))

					time.Sleep(time.Second * 2)

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "1"
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig1)
						return fetchedDeploymentConfig1.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas1)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig2)
						return fetchedDeploymentConfig2.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas2)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig3)
						return fetchedDeploymentConfig3.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas3)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig4)
						return fetchedDeploymentConfig4.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas4)))

				} else {

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "1"

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment1)
						return fetchedDeployment1
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment2)
						return fetchedDeployment2
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment3)
						return fetchedDeployment3
					}, timeout, interval).Should(Not(BeNil()))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment4)
						return fetchedDeployment4
					}, timeout, interval).Should(Not(BeNil()))

					time.Sleep(time.Second * 2)

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "1"
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment1)
						return *fetchedDeployment1.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas1)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment2)
						return *fetchedDeployment2.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas2)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "1"

					key.Namespace = "e2e-tests-crds" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment3)
						return *fetchedDeployment3.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas3)))

					key.Name = "case" + strconv.Itoa(casenumber) + "-" + "2"

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment4)
						return *fetchedDeployment4.Spec.Replicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas4)))
				}

			},
				// Structure:  ("Description of the case" , expectedReplicas)
				table.Entry("CASE 1  | Apply a CSS and affect only opted-in applications", 2, 1, 1, 2),
				table.Entry("CASE 2  | Apply a SS on one namespace", 4, 1, 1, 1),
				table.Entry("CASE 3  | Apply SS with higher prio than an existing CSS", 4, 1, 1, 2),
				table.Entry("CASE 4  | Apply CSS with higher prio than an existing SS", 4, 1, 1, 4),
			)
		})
	})

})

func CreateDeploymentRQ(deploymentInfo types.NamespacedName, optin string, casenumber int, name string) v1.Deployment {
	var replicaCount int32

	replicaCount = 1

	var appName = "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": optin,
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     "2",
		"scaler/state-default-replicas": "1",
		"scaler/state-peak-replicas":    "4",
	}

	matchlabels := map[string]string{
		"app": appName,
	}
	var deploymentname string
	deploymentname = "case" + strconv.Itoa(casenumber) + "-" + name

	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentname,
			Namespace:   deploymentInfo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},

		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Replicas: &replicaCount,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  deploymentInfo.Name,
						},
					},
				},
			},
		},
	}
	return *dep
}

func CreateDeploymentConfigRQ(deploymentInfo types.NamespacedName, optin string, casenumber int, name string) ocv1.DeploymentConfig {
	var replicaCount int32

	replicaCount = 1

	var appName = "random-generator-1" + "-" + name
	labels := map[string]string{
		"deployment-config.name": appName,
		"scaler/opt-in":          optin,
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     "2",
		"scaler/state-default-replicas": "1",
		"scaler/state-peak-replicas":    "4",
	}

	matchlabels := map[string]string{
		"deployment-config.name": appName,
	}
	var deploymentname string
	deploymentname = "case" + strconv.Itoa(casenumber) + "-" + name

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentname,
			Namespace:   deploymentInfo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: ocv1.DeploymentConfigSpec{
			Replicas: replicaCount,
			Selector: matchlabels,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  deploymentInfo.Name,
						},
					},
				},
			},
		},
	}
	return *deploymentConfig
}

func CrdCreateNS(deploymentInfo types.NamespacedName, number string) corev1.Namespace {

	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentInfo.Namespace + "-" + number,
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	return *ns
}
