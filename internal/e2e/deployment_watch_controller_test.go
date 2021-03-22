package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/containersol/prescale-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Deployment Watch Controller", func() {

	const timeout = time.Second * 20
	const interval = time.Second * 1

	var deployment v1.Deployment
	var clusterscalingstate *v1alpha1.ClusterScalingState

	var key = types.NamespacedName{
		Name:      "test",
		Namespace: "default",
	}

	BeforeEach(func() {

		// failed test runs that don't clean up leave resources behind.
		var replicas int32 = 2
		var replicaPoint *int32 = &replicas
		deployment = CreateDeployment(*replicaPoint, key, true)
		clusterscalingstate = CreateClusterScalingState()

		By("Creating the ClusterScalingState object first")
		Expect(k8sClient.Create(context.Background(), clusterscalingstate)).Should(Succeed())

		By("Then creating the deployment object")
		Expect(k8sClient.Create(context.Background(), &deployment)).Should(Succeed())
		time.Sleep(time.Second * 2)

	})

	AfterEach(func() {
		// Tear down the deployment
		Expect(k8sClient.Delete(context.Background(), &deployment)).Should(Succeed())
		// Tear down ClusterScalingState
		Expect(k8sClient.Delete(context.Background(), clusterscalingstate)).Should(Succeed())
		time.Sleep(time.Second * 2)
	})

	Context("Deployment change", func() {
		When("A deployment is modified", func() {
			table.DescribeTable("TestDeployment", func(annotationchange bool, replicachange bool, optinOld bool, optinNew bool, expectedReplicas int) {
				println("ExpectedReplicas", expectedReplicas)
				println("Annotation", annotationchange)
				println("Replica", replicachange)
				println("LabelNew", optinOld)
				println("LabelOld", optinNew)

				// Get the deployment (created by the Before() step to edit it according to the test)
				fetched := v1.Deployment{}
				Eventually(func() v1.Deployment {
					k8sClient.Get(context.Background(), key, &fetched)
					return fetched
				}, timeout, interval).Should(Not(BeNil()))

				// Only modify deployment if either Annotation or Replica has changed. Otherwise do nothing. Expect 'bau' replica #
				if !annotationchange && !replicachange {
					if annotationchange {
						fetched = ChangeAnnotation(fetched)
					}

					if replicachange {
						fetched = ChangeReplicas(fetched)
					}
				}

				// Update with the new changes
				Expect(k8sClient.Update(context.Background(), &fetched)).Should(Succeed())

				time.Sleep(time.Second * 2)

				var replicas32 int32 = int32(expectedReplicas)

				Eventually(func() int32 {
					k8sClient.Get(context.Background(), key, fetched.DeepCopy())
					return *fetched.Spec.Replicas
				}, timeout, interval).Should(Equal(replicas32))
			},
				// Structure:  ("Description of the case" , annotationchange, replicachange, oldoptin, newoptin, expectedReplicas)
				table.Entry("CASE 1  | Positive, Should scale to 5 | Everything has changed. Change to 'bau'", true, true, true, true, 5),
				table.Entry("CASE 2  | Positive, Should scale to 3 | Deployment has been disabled fallback to default annotation", true, true, true, false, 3),
				table.Entry("CASE 3  | Positive, Should scale to 5 | Deployment opted in. Scale to 'bau'", true, true, false, true, 5),
				table.Entry("CASE 4  | Negative, Should NOT scale  | Deployment opted out", true, true, false, false, 2),
				table.Entry("CASE 5  | Positive, Should scale to 5 | Annotation has been modified.", true, false, true, true, 5),
				table.Entry("CASE 6  | Positive, Should scale to 3 | Deployment has been disabled fallback to default annotation", true, false, true, false, 3),
				table.Entry("CASE 7  | Positive, Should scale to 5 | Deployment opted in. Scale to 'bau'", true, false, false, true, 5),
				table.Entry("CASE 8  | Negative, Should NOT scale  | Deployment opted out", true, false, false, false, 2),
				table.Entry("CASE 9  | Positive, Should scale to 5 | Replica count has been modified. Rectify back to 'bau'", false, true, true, true, 5),
				table.Entry("CASE 10 | Positive, Should scale to 3 | Deployment has been disabled fallback to default annotation", false, true, true, false, 3),
				table.Entry("CASE 11 | Positive, Should scale to 5 | Deployment opted in. Scale to 'bau'", false, true, false, true, 5),
				table.Entry("CASE 12 | Negative, Should NOT scale  | Deployment opted out", false, true, false, false, 2),
				table.Entry("CASE 13 | Negative, Should NOT scale  | Something else on the deployment changed. Don't do anything", false, false, true, true, 5),
				table.Entry("CASE 14 | Positive, Should scale to 3 | Deployment has been disabled fallback to default annotation", false, false, true, false, 3),
				table.Entry("CASE 15 | Positive, Should scale to 5 | Deployment opted in. Scale to 'bau'", false, false, false, true, 5),
				table.Entry("CASE 18 | Negative, Should NOT scale  | Nothing happend", false, false, false, false, 2),
			)
		})
	})

})

func ChangeAnnotation(deployment v1.Deployment) v1.Deployment {
	annotations := map[string]string{
		"scaler/state-bau-replicas":     "4",
		"scaler/state-default-replicas": "1",
		"scaler/state-peak-replicas":    "7",
	}

	deployment.Annotations = annotations

	return deployment
}

func ChangeReplicas(deployment v1.Deployment) v1.Deployment {
	var replicas int32 = 3

	spec2 := deployment.Spec
	spec2.Replicas = &replicas

	deployment.Spec = spec2
	return deployment
}

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

func CreateDeployment(replicaCount int32, deploymentInfo types.NamespacedName, optIn bool) v1.Deployment {

	appName := "random-generator-1"
	labels := map[string]string{
		"app":           appName,
		"scaler/opt-in": strconv.FormatBool(optIn),
	}

	annotations := map[string]string{
		"scaler/state-bau-replicas":     "5",
		"scaler/state-default-replicas": "1",
		"scaler/state-peak-replicas":    "7",
	}

	matchlabels := map[string]string{
		"app": appName,
	}
	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentInfo.Name,
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
