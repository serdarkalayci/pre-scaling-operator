package controllers

import (
	"context"
	"strconv"
	"time"

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

var _ = Describe("e2e Test for the main operator functionalities", func() {

	const timeout = time.Second * 25
	const interval = time.Millisecond * 500

	var casenumber = 1
	OpenshiftCluster, _ := validations.ClusterCheck()
	var deployment v1.Deployment
	var deploymentconfig ocv1.DeploymentConfig
	var rq corev1.ResourceQuota

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "e2e-tests",
	}

	BeforeEach(func() {

	})

	AfterEach(func() {
		// Tear down the deployment or deploymentconfig
		if OpenshiftCluster {
			Expect(k8sClient.Delete(context.Background(), &deploymentconfig)).Should(Succeed())
		} else {
			Expect(k8sClient.Delete(context.Background(), &deployment)).Should(Succeed())
		}

		Expect(k8sClient.Delete(context.Background(), &rq)).Should(Succeed())

		casenumber = casenumber + 1
		time.Sleep(time.Second * 3)
	})

	Context("Deployment in place and modification test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then the deployment gets modified..", func(replicachange bool, expectedReplicas int) {
				key.Name = "case" + strconv.Itoa(casenumber)
				fetchedDeployment := v1.Deployment{}
				fetchedDeploymentConfig := ocv1.DeploymentConfig{}

				if OpenshiftCluster {

					deploymentconfig = *CreateDeploymentConfigRQ(key, casenumber)

					Expect(k8sClient.Create(context.Background(), &deploymentconfig)).Should(Succeed())

					rq = CreateRQ(key, casenumber)

					Expect(k8sClient.Create(context.Background(), &rq)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig
					}, timeout, interval).Should(Not(BeNil()))

					if replicachange {
						fetchedDeploymentConfig = ChangeReplicasDC(fetchedDeploymentConfig)
					}

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

					deployment = CreateDeploymentRQ(key, casenumber)
					Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())

					rq = CreateRQ(key, casenumber)

					Expect(k8sClient.Create(context.Background(), &rq)).Should(Succeed())

					time.Sleep(time.Second * 2)

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment
					}, timeout, interval).Should(Not(BeNil()))

					if replicachange {
						fetchedDeployment = ChangeReplicas(fetchedDeployment)
					}

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

				// Default Replica Count from test if oldoptin = true: 3
				// Default Replica Count from test if oldoptin = false: 1
				// Default fallback annotation count: 2
				// bau Annoation (if changed) will change to :4
				// Replica change (that needs to be rectified): 5
				// Structure:  ("Description of the case" , annotationchange, replicachange, oldoptin, newoptin, expectedReplicas)
				table.Entry("CASE 1  | Should not scale to 5 | Quota has exceeded. ", false, 1),
				table.Entry("CASE 2  | Should scale to 3 | Enough Quota to scale down.", false, 3),
				table.Entry("CASE 3  | Should be at 3 | Same replicas, no change.", false, 3),
				table.Entry("CASE 4  | Should scale to 3 | Enough quota to scale up", false, 3),
			)
		})
	})

})

func CreateDeploymentRQ(deploymentInfo types.NamespacedName, casenumber int) v1.Deployment {
	var replicaCount int32
	var stateReplica string

	if casenumber == 1 {
		replicaCount = 1
	} else if casenumber == 2 {
		replicaCount = 4
	} else if casenumber == 3 {
		replicaCount = 3
	} else {
		replicaCount = 2
	}

	if casenumber == 1 {
		stateReplica = "10"
	} else {
		stateReplica = "3"
	}

	var appName = "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": "true",
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     stateReplica,
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	REQCPU := resource.NewQuantity(50, resource.DecimalSI)
	REQMEM := resource.NewQuantity(50, resource.BinarySI)

	LIMCPU := resource.NewQuantity(100, resource.DecimalSI)
	LIMMEM := resource.NewQuantity(100, resource.BinarySI)

	matchlabels := map[string]string{
		"app": appName,
	}
	var deploymentname string
	deploymentname = "case" + strconv.Itoa(casenumber)

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
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    *LIMCPU,
									corev1.ResourceMemory: *LIMMEM,
								},
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    *REQCPU,
									corev1.ResourceMemory: *REQMEM,
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

func CreateDeploymentConfigRQ(deploymentInfo types.NamespacedName, casenumber int) *ocv1.DeploymentConfig {
	var replicaCount int32
	var stateReplica string

	if casenumber == 1 {
		replicaCount = 1
	} else if casenumber == 2 {
		replicaCount = 4
	} else if casenumber == 3 {
		replicaCount = 3
	} else {
		replicaCount = 2
	}

	if casenumber == 1 {
		stateReplica = "5"
	} else {
		stateReplica = "3"
	}

	REQCPU := resource.NewQuantity(50, resource.DecimalSI)
	REQMEM := resource.NewQuantity(50, resource.BinarySI)

	LIMCPU := resource.NewQuantity(100, resource.DecimalSI)
	LIMMEM := resource.NewQuantity(100, resource.BinarySI)

	var appName = "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": "true",
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     stateReplica,
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	matchlabels := map[string]string{
		"app": appName,
	}
	var deploymentname string
	deploymentname = "case" + strconv.Itoa(casenumber)

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentname,
			Namespace:   deploymentInfo.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},

		Spec: ocv1.DeploymentConfigSpec{
			Replicas: replicaCount,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "chriscmsoft/random-generator:latest",
							Name:  deploymentInfo.Name,
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    *REQCPU,
									corev1.ResourceMemory: *REQMEM,
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    *LIMCPU,
									corev1.ResourceMemory: *LIMMEM,
								},
							},
						},
					},
				},
			},
		},
	}
	return deploymentConfig
}

func CreateRQ(deploymentInfo types.NamespacedName, casenumber int) corev1.ResourceQuota {

	HardLimCPU := resource.NewQuantity(450, resource.DecimalSI)
	HardReqCPU := resource.NewQuantity(300, resource.DecimalSI)
	HardLimMEM := resource.NewQuantity(450, resource.BinarySI)
	HardReqMEM := resource.NewQuantity(300, resource.BinarySI)

	rq := &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceLimitsCPU:      *HardLimCPU,
				corev1.ResourceLimitsMemory:   *HardLimMEM,
				corev1.ResourceRequestsCPU:    *HardReqCPU,
				corev1.ResourceRequestsMemory: *HardReqMEM,
			},
		},
		Status: corev1.ResourceQuotaStatus{},
	}

	return *rq
}

// This covers the case when someone external simply edits the replica count. Depending on the opt-in the operator needs to rectify this.
func ChangeReplicas(deployment v1.Deployment) v1.Deployment {
	var replicas int32 = 5

	spec2 := deployment.Spec
	spec2.Replicas = &replicas

	deployment.Spec = spec2
	return deployment
}

func ChangeReplicasDC(deploymentconfig ocv1.DeploymentConfig) ocv1.DeploymentConfig {
	var replicas int32 = 5

	spec2 := deploymentconfig.Spec
	spec2.Replicas = replicas

	deploymentconfig.Spec = spec2
	return deploymentconfig
}
