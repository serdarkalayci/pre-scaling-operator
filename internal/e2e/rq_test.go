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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("e2e Test for the resource quotas functionalities", func() {

	const timeout = time.Second * 60
	const interval = time.Millisecond * 500

	var casenumber = 1
	OpenshiftCluster, _ := validations.ClusterCheck()
	var deployment v1.Deployment
	var deploymentconfig ocv1.DeploymentConfig
	var namespace corev1.Namespace
	var rq corev1.ResourceQuota
	var css v1alpha1.ClusterScalingState
	var cssd v1alpha1.ClusterScalingStateDefinition

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "e2e-tests-resourcequotas" + strconv.Itoa(casenumber),
	}

	BeforeEach(func() {

		css = CreateClusterScalingState("bau")
		cssd = CreateClusterScalingStateDefinition()

		Expect(k8sClient.Create(context.Background(), &cssd)).Should(Succeed())
		Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

		namespace = createNSforRQtest(key)
		Expect(k8sClient.Create(context.Background(), &namespace)).Should(Succeed())

		rq = createRQ(key, casenumber)
		Expect(k8sClient.Create(context.Background(), &rq)).Should(Succeed())

		time.Sleep(time.Second * 10)

	})

	AfterEach(func() {
		// Tear down the deployment or deploymentconfig
		if OpenshiftCluster {
			Expect(k8sClient.Delete(context.Background(), &deploymentconfig)).Should(Succeed())
		} else {
			Expect(k8sClient.Delete(context.Background(), &deployment)).Should(Succeed())
		}

		Expect(k8sClient.Delete(context.Background(), &rq)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &namespace)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &css)).Should(Succeed())
		Expect(k8sClient.Delete(context.Background(), &cssd)).Should(Succeed())

		casenumber = casenumber + 1

		key = types.NamespacedName{
			Name:      "test",
			Namespace: "e2e-tests-resourcequotas" + strconv.Itoa(casenumber),
		}

		time.Sleep(time.Second * 1)
	})

	Context("Deployment in place and modification test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(expectedReplicas int) {
				key.Name = "case" + strconv.Itoa(casenumber)
				fetchedDeployment := v1.Deployment{}
				fetchedDeploymentConfig := ocv1.DeploymentConfig{}

				replicaCount, stateReplica := caseSpecifics(casenumber)

				if OpenshiftCluster {

					deploymentconfig = createDeploymentConfigforRQtest(key, "false", casenumber, replicaCount, stateReplica)

					Expect(k8sClient.Create(context.Background(), &deploymentconfig)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig
					}, timeout, interval).Should(Not(BeNil()))

					fetchedDeploymentConfig = changeOptInDCforRQtest(fetchedDeploymentConfig, "true")

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeploymentConfig)).Should(Succeed())

					time.Sleep(time.Second * 2)

					var replicas32 int32 = int32(expectedReplicas)

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig.Spec.Replicas
					}, timeout, interval).Should(Equal(replicas32))

				} else {

					deployment = createDeploymentforRQtest(key, "false", casenumber, replicaCount, stateReplica)
					Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment
					}, timeout, interval).Should(Not(BeNil()))

					fetchedDeployment = changeOptInforRQtest(fetchedDeployment, "true")

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeployment)).Should(Succeed())

					time.Sleep(time.Second * 2)

					var replicas32 int32 = int32(expectedReplicas)

					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return *fetchedDeployment.Spec.Replicas
					}, timeout, interval).Should(Equal(replicas32))
				}

			},
				// Structure:  ("Description of the case" , expectedReplicas)
				table.Entry("CASE 1  | Should scale down from 3 to 2 | Enough Quota to scale down.", 2),
				table.Entry("CASE 2  | Should not scale to 5 and stay at 1 | Quota has exceeded. ", 1),
				table.Entry("CASE 3  | Should stay at 2 | Same replicas, no change.", 2),
				table.Entry("CASE 4  | Should scale from 3 to 4 | Enough quota to scale up", 4),
			)
		})
	})

})

func changeOptInDCforRQtest(deploymentconfig ocv1.DeploymentConfig, optIn string) ocv1.DeploymentConfig {

	deploymentconfig.Labels = map[string]string{
		"deployment-config.name": "random-generator-1",
		"scaler/opt-in":          optIn,
	}

	return deploymentconfig
}

func changeOptInforRQtest(deployment v1.Deployment, optIn string) v1.Deployment {

	deployment.Labels = map[string]string{
		"app":           "random-generator-1",
		"scaler/opt-in": optIn,
	}

	return deployment
}

func createDeploymentforRQtest(deploymentInfo types.NamespacedName, optin string, casenumber int, replicaCount int32, stateReplica string) v1.Deployment {

	REQCPU, _ := resource.ParseQuantity("50m")
	LIMCPU, _ := resource.ParseQuantity("100m")
	REQMEM, _ := resource.ParseQuantity("50Mi")
	LIMMEM, _ := resource.ParseQuantity("100Mi")

	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"app":           "random-generator-1",
				"scaler/opt-in": optin,
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     stateReplica,
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
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  deploymentInfo.Name,
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    LIMCPU,
									corev1.ResourceMemory: LIMMEM,
								},
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    REQCPU,
									corev1.ResourceMemory: REQMEM,
								},
							},
						},
					},
				},
			},
		},
	}
	return *dep
}

func createDeploymentConfigforRQtest(deploymentInfo types.NamespacedName, optin string, casenumber int, replicaCount int32, stateReplica string) ocv1.DeploymentConfig {

	REQCPU, _ := resource.ParseQuantity("50m")
	LIMCPU, _ := resource.ParseQuantity("100m")
	REQMEM, _ := resource.ParseQuantity("50Mi")
	LIMMEM, _ := resource.ParseQuantity("100Mi")

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"deployment-config.name": "random-generator-1",
				"scaler/opt-in":          optin,
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     stateReplica,
				"scaler/state-default-replicas": "2",
				"scaler/state-peak-replicas":    "7",
			},
		},
		Spec: ocv1.DeploymentConfigSpec{
			Replicas: replicaCount,
			Selector: map[string]string{
				"deployment-config.name": "random-generator-1",
			},
			Strategy: ocv1.DeploymentStrategy{
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    REQCPU,
						corev1.ResourceMemory: REQMEM,
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    LIMCPU,
						corev1.ResourceMemory: LIMMEM,
					},
				},
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment-config.name": "random-generator-1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  deploymentInfo.Name,
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    REQCPU,
									corev1.ResourceMemory: REQMEM,
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    LIMCPU,
									corev1.ResourceMemory: LIMMEM,
								},
							},
						},
					},
				},
			},
		},
	}
	return *deploymentConfig
}

func createRQ(deploymentInfo types.NamespacedName, casenumber int) corev1.ResourceQuota {

	HardLimCPU, _ := resource.ParseQuantity("450m")
	HardReqCPU, _ := resource.ParseQuantity("300m")
	HardLimMEM, _ := resource.ParseQuantity("450Mi")
	HardReqMEM, _ := resource.ParseQuantity("300Mi")

	rq := &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceLimitsCPU:      HardLimCPU,
				corev1.ResourceLimitsMemory:   HardLimMEM,
				corev1.ResourceRequestsCPU:    HardReqCPU,
				corev1.ResourceRequestsMemory: HardReqMEM,
			},
		},
	}

	return *rq
}

func createNSforRQtest(deploymentInfo types.NamespacedName) corev1.Namespace {

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

func caseSpecifics(casenumber int) (int32, string) {

	var replicaCount int32
	var stateReplica string

	if casenumber == 1 {
		replicaCount = 3
	} else if casenumber == 2 {
		replicaCount = 1
	} else if casenumber == 3 {
		replicaCount = 2
	} else {
		replicaCount = 3
	}

	if casenumber == 2 {
		stateReplica = "8"
	} else if casenumber == 4 {
		stateReplica = "4"
	} else {
		stateReplica = "2"
	}

	return replicaCount, stateReplica
}
