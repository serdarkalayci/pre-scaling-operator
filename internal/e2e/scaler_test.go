package controllers

import (
	"context"
	"fmt"
	"reflect"
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
		Namespace: "e2e-tests-scaler" + strconv.Itoa(casenumber),
	}

	BeforeEach(func() {

		cssd = CreateClusterScalingStateDefinition()

		Expect(k8sClient.Create(context.Background(), &cssd)).Should(Succeed())

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
			Namespace: "e2e-tests-scaler" + strconv.Itoa(casenumber),
		}

		time.Sleep(time.Second * 1)
	})

	Context("Deployment scaling test", func() {
		When("a deployment is already in place", func() {
			table.DescribeTable("And then a deployment is being scaled..", func(scaleUpOrDown string, expectedReplicas int, stepscale bool) {

				css = CreateClusterScalingState("bau", stepscale)

				Expect(k8sClient.Create(context.Background(), &css)).Should(Succeed())

				key.Name = "case" + strconv.Itoa(casenumber)
				fetchedDeployment := v1.Deployment{}
				fetchedDeploymentConfig := ocv1.DeploymentConfig{}
				initialReplicas := ScalerTestCaseInitialReplicas(scaleUpOrDown)
				if OpenshiftCluster {

					deploymentconfig = *createDeploymentConfigScaler(key, scaleUpOrDown, casenumber)
					Expect(k8sClient.Create(context.Background(), &deploymentconfig)).Should(Succeed())

					// Wait for deploymentconfig to get ready
					time.Sleep(time.Second * 2)
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig.Status.AvailableReplicas
					}, timeout, interval).Should(Equal(initialReplicas))

					time.Sleep(time.Second * 5)

					Eventually(func() ocv1.DeploymentConfig {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig
					}, timeout, interval).Should(Not(BeNil()))

					fetchedDeploymentConfig = changeAnnotationDCReplicas(fetchedDeploymentConfig, int32(expectedReplicas))

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeploymentConfig)).Should(Succeed())

					replicasReadyList, replicasSpecList := GetReplicaAndReadyList(expectedReplicas, key, OpenshiftCluster)
					if stepscale {
						Expect(reflect.DeepEqual(replicasReadyList, replicasSpecList)).To(BeTrue(), "Step scale assertion failed! The lists are not equal!")
						Expect(AssessSmoothClimbOrDescend(replicasReadyList, scaleUpOrDown)).To(BeTrue(), "Step scale assertion failed! ReplicaReady list was not ascending or descending correctly")
						Expect(AssessSmoothClimbOrDescend(replicasSpecList, scaleUpOrDown)).To(BeTrue(), "Step scale assertion failed! ReplicaSpec list was not ascending or descending correctly!")
					} else {
						Expect(AssessRapidScaleReplicaList(replicasSpecList, initialReplicas, int32(expectedReplicas))).To(BeTrue(), "Rapid scale assertion failed! ReplicaReady list was not correct!")
					}
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
						return fetchedDeploymentConfig.Status.AvailableReplicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas)))

				} else {

					deployment = createDeploymentScaler(key, scaleUpOrDown, casenumber)
					Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())

					// Wait until deployment is ready
					time.Sleep(time.Second * 2)
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment.Status.ReadyReplicas
					}, timeout, interval).Should(Equal(int32(initialReplicas)))

					time.Sleep(time.Second * 5)

					Eventually(func() v1.Deployment {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment
					}, timeout, interval).Should(Not(BeNil()))

					fetchedDeployment = changeAnnotationDeploymentReplicas(fetchedDeployment, int32(expectedReplicas))

					// Update with the new changes
					By("Then a deployment is updated")
					Expect(k8sClient.Update(context.Background(), &fetchedDeployment)).Should(Succeed())

					By("The deployment scaling is being assessed")
					replicasReadyList, replicasSpecList := GetReplicaAndReadyList(expectedReplicas, key, OpenshiftCluster)
					if stepscale {
						Expect(reflect.DeepEqual(replicasReadyList, replicasSpecList)).To(BeTrue(), "Step scale assertion failed! The lists are not equal!")
						Expect(AssessSmoothClimbOrDescend(replicasReadyList, scaleUpOrDown)).To(BeTrue(), "Step scale assertion failed! ReplicaReady list was not ascending or descending correctly")
						Expect(AssessSmoothClimbOrDescend(replicasSpecList, scaleUpOrDown)).To(BeTrue(), "Step scale assertion failed! ReplicaSpec list was not ascending or descending correctly!")
					} else {
						Expect(AssessRapidScaleReplicaList(replicasSpecList, initialReplicas, int32(expectedReplicas))).To(BeTrue(), "Rapid scale assertion failed! ReplicaReady list was not correct!")
					}
					By("Then deployment arrives at the desiredreplicacount")
					Eventually(func() int32 {
						k8sClient.Get(context.Background(), key, &fetchedDeployment)
						return fetchedDeployment.Status.ReadyReplicas
					}, timeout, interval).Should(Equal(int32(expectedReplicas)))

				}

			},

				table.Entry("CASE 1  | Should step scale from 0 up to 8 |  ", "UP", 8, true),
				table.Entry("CASE 2  | Should step scale from 8 down to 1 |  ", "DOWN", 1, true),
				table.Entry("CASE 3  | Should rapid scale from 0 up to 8 |  ", "UP", 8, false),
				table.Entry("CASE 4  | Should rapid scale from 8 down to 1 |  ", "DOWN", 1, false),
			)
		})
	})

})

//check is the integers in the list are climbing like 1,2,3,4,5 or descending like 5,4,3,2,1
func AssessSmoothClimbOrDescend(list []int32, upordown string) bool {
	// return false if the previous neighbour difference is not equal to 1
	for i := 1; i < len(list); i++ {
		if upordown == "UP" {

			if list[i-1] != list[i]-1 {
				return false
			}
		} else if upordown == "DOWN" {
			if list[i-1] != list[i]+1 {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func GetReplicaAndReadyList(expectedReplicas int, key types.NamespacedName, openshiftCluster bool) ([]int32, []int32) {
	stay := true
	fetchedDeployment := v1.Deployment{}
	fetchedDeploymentConfig := ocv1.DeploymentConfig{}
	const timeoutAssessor = time.Second * 2
	const intervalAssessor = time.Millisecond * 200
	var replicasReadyList []int32
	var replicasSpecList []int32
	var latestReady int32 = -1
	var latestSpec int32 = -1
	if openshiftCluster {
		for stay {

			Eventually(func() ocv1.DeploymentConfig {
				k8sClient.Get(context.Background(), key, &fetchedDeploymentConfig)
				return fetchedDeploymentConfig
			}, timeoutAssessor, intervalAssessor).Should(Not(BeNil()))

			turnReadyReplicas := fetchedDeploymentConfig.Status.ReadyReplicas
			turnSpecReplicas := fetchedDeploymentConfig.Spec.Replicas
			// Only add unique values
			if latestReady != turnReadyReplicas {
				replicasReadyList = append(replicasReadyList, turnReadyReplicas)
				latestReady = turnReadyReplicas
			}

			if latestSpec != turnSpecReplicas {
				replicasSpecList = append(replicasSpecList, turnSpecReplicas)
				latestSpec = turnSpecReplicas
			}

			if turnReadyReplicas == int32(expectedReplicas) && turnSpecReplicas == int32(expectedReplicas) {
				stay = false
			}

		}
	} else {
		for stay {
			Eventually(func() v1.Deployment {
				k8sClient.Get(context.Background(), key, &fetchedDeployment)
				return fetchedDeployment
			}, timeoutAssessor, intervalAssessor).Should(Not(BeNil()))

			turnReadyReplicas := fetchedDeployment.Status.ReadyReplicas
			turnSpecReplicas := fetchedDeployment.Spec.Replicas
			// Only add unique values
			if latestReady != turnReadyReplicas {
				replicasReadyList = append(replicasReadyList, turnReadyReplicas)
				latestReady = turnReadyReplicas
			}

			if latestSpec != *turnSpecReplicas {
				replicasSpecList = append(replicasSpecList, *turnSpecReplicas)
				latestSpec = *turnSpecReplicas
			}

			if turnReadyReplicas == int32(expectedReplicas) && *turnSpecReplicas == int32(expectedReplicas) {
				stay = false
			}

		}
	}

	return replicasReadyList, replicasSpecList
}

func AssessRapidScaleReplicaList(list []int32, initialReplicas int32, expectedReplicas int32) bool {

	//First check if the list is ok
	if len(list) > 2 || len(list) < 2 {
		return false
	}

	if list[0] != initialReplicas || list[len(list)-1] != expectedReplicas {
		return false
	}

	return true
}

func ScalerTestCaseInitialReplicas(upordown string) int32 {
	if upordown == "UP" {
		return 0
	} else if upordown == "DOWN" {
		return 8
	}
	return 5
}

func changeAnnotationDeploymentReplicas(deployment v1.Deployment, replicas int32) v1.Deployment {

	deployment.Annotations = map[string]string{
		"scaler/state-bau-replicas":     fmt.Sprint(replicas), // That reflects the annotation change and will change replica # to 4
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	deployment.Labels = map[string]string{
		"scaler/opt-in": "true",
	}

	return deployment
}

func changeAnnotationDCReplicas(deploymentconfig ocv1.DeploymentConfig, replicas int32) ocv1.DeploymentConfig {

	deploymentconfig.Annotations = map[string]string{
		"scaler/state-bau-replicas":     fmt.Sprint(replicas),
		"scaler/state-default-replicas": "2",
		"scaler/state-peak-replicas":    "7",
	}

	deploymentconfig.Labels = map[string]string{
		"scaler/opt-in": "true",
	}

	return deploymentconfig
}

func createDeploymentScaler(deploymentInfo types.NamespacedName, upordown string, casenumber int) v1.Deployment {

	replicas := ScalerTestCaseInitialReplicas(upordown)
	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"app":           "random-generator-1",
				"scaler/opt-in": "false",
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     fmt.Sprint(replicas),
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
			Replicas: &replicas,
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

func createDeploymentConfigScaler(deploymentInfo types.NamespacedName, upordown string, casenumber int) *ocv1.DeploymentConfig {
	replicas := ScalerTestCaseInitialReplicas(upordown)

	deploymentConfig := &ocv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "case" + strconv.Itoa(casenumber),
			Namespace: deploymentInfo.Namespace,
			Labels: map[string]string{
				"app":           "random-generator-1",
				"scaler/opt-in": "false",
			},
			Annotations: map[string]string{
				"scaler/state-bau-replicas":     fmt.Sprint(replicas),
				"scaler/state-default-replicas": "2",
				"scaler/state-peak-replicas":    "7",
			},
		},

		Spec: ocv1.DeploymentConfigSpec{
			Replicas: replicas,
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
